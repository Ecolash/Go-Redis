package server_test

import (
	"bufio"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/codecrafters-io/redis-starter-go/internal/rdb"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
	"github.com/codecrafters-io/redis-starter-go/internal/server"
)

// TestReplicaAppliesPropagatedCommands wires a real master+replica pair via TCP
// and verifies that a SET issued on the master is applied to the replica's
// store, with no reply sent back over the replication link.
func TestReplicaAppliesPropagatedCommands(t *testing.T) {
	master, err := server.New("127.0.0.1:0", "master", "")
	if err != nil {
		t.Fatalf("create master: %v", err)
	}
	t.Cleanup(func() { master.Close() })
	go master.Run()

	replica, err := server.New("127.0.0.1:0", "slave", master.Addr())
	if err != nil {
		t.Fatalf("create replica: %v", err)
	}
	t.Cleanup(func() { replica.Close() })
	go replica.Run()

	// Give the replica handshake (PING / REPLCONF / PSYNC + RDB) time to land
	// before we send the propagated write — otherwise the SET could be issued
	// before the replica is registered on the master.
	time.Sleep(200 * time.Millisecond)

	masterConn, err := net.DialTimeout("tcp", master.Addr(), time.Second)
	if err != nil {
		t.Fatalf("dial master: %v", err)
	}
	t.Cleanup(func() { masterConn.Close() })
	masterConn.SetDeadline(time.Now().Add(2 * time.Second))
	masterConn.Write([]byte("*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$1\r\n1\r\n"))
	if line, err := bufio.NewReader(masterConn).ReadString('\n'); err != nil || line != "+OK\r\n" {
		t.Fatalf("master SET reply = %q, err=%v", line, err)
	}

	replicaConn, err := net.DialTimeout("tcp", replica.Addr(), time.Second)
	if err != nil {
		t.Fatalf("dial replica: %v", err)
	}
	t.Cleanup(func() { replicaConn.Close() })
	replicaReader := bufio.NewReader(replicaConn)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		replicaConn.SetDeadline(time.Now().Add(500 * time.Millisecond))
		replicaConn.Write([]byte("*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n"))
		header, err := replicaReader.ReadString('\n')
		if err != nil {
			t.Fatalf("replica GET header: %v", err)
		}
		if header == "$1\r\n" {
			body, err := replicaReader.ReadString('\n')
			if err != nil {
				t.Fatalf("replica GET body: %v", err)
			}
			if body != "1\r\n" {
				t.Fatalf("replica GET body = %q, want %q", body, "1\r\n")
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("replica never observed propagated SET")
}

// TestReplicaRespondsToGetAck stands up a fake master that drives the replica
// through the handshake, then sends REPLCONF GETACK * and asserts the replica
// replies with REPLCONF ACK 0 over the same connection.
func TestReplicaRespondsToGetAck(t *testing.T) {
	ml, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("fake master listen: %v", err)
	}
	t.Cleanup(func() { ml.Close() })

	connCh := make(chan net.Conn, 1)
	go func() {
		c, err := ml.Accept()
		if err != nil {
			return
		}
		connCh <- c
	}()

	replica, err := server.New("127.0.0.1:0", "slave", ml.Addr().String())
	if err != nil {
		t.Fatalf("create replica: %v", err)
	}
	t.Cleanup(func() { replica.Close() })
	go replica.Run()

	var mconn net.Conn
	select {
	case mconn = <-connCh:
	case <-time.After(2 * time.Second):
		t.Fatal("replica never dialed fake master")
	}
	t.Cleanup(func() { mconn.Close() })
	mconn.SetDeadline(time.Now().Add(5 * time.Second))
	mr := bufio.NewReader(mconn)

	// Handshake: PING / REPLCONF listening-port / REPLCONF capa / PSYNC.
	drainArray(t, mr, 1)
	mconn.Write([]byte("+PONG\r\n"))
	drainArray(t, mr, 3)
	mconn.Write([]byte("+OK\r\n"))
	drainArray(t, mr, 3)
	mconn.Write([]byte("+OK\r\n"))
	drainArray(t, mr, 3)
	mconn.Write([]byte("+FULLRESYNC 0000000000000000000000000000000000000000 0\r\n"))
	mconn.Write(resp.File(rdb.Empty()))

	// Send GETACK and verify the replica responds.
	mconn.Write([]byte("*3\r\n$8\r\nREPLCONF\r\n$6\r\nGETACK\r\n$1\r\n*\r\n"))

	want := "*3\r\n$8\r\nREPLCONF\r\n$3\r\nACK\r\n$1\r\n0\r\n"
	got := make([]byte, len(want))
	if _, err := io.ReadFull(mr, got); err != nil {
		t.Fatalf("read ACK reply: %v", err)
	}
	if string(got) != want {
		t.Errorf("ACK reply = %q, want %q", string(got), want)
	}
}

// drainArray reads a single RESP array of n bulk strings off r and discards it.
func drainArray(t *testing.T, r *bufio.Reader, n int) {
	t.Helper()
	header, err := r.ReadString('\n')
	if err != nil {
		t.Fatalf("read array header: %v", err)
	}
	if !strings.HasPrefix(header, "*") {
		t.Fatalf("expected *, got %q", header)
	}
	for i := 0; i < n; i++ {
		lenLine, err := r.ReadString('\n')
		if err != nil {
			t.Fatalf("read bulk len: %v", err)
		}
		size, err := strconv.Atoi(strings.TrimRight(lenLine[1:], "\r\n"))
		if err != nil {
			t.Fatalf("parse bulk len %q: %v", lenLine, err)
		}
		body := make([]byte, size+2)
		if _, err := io.ReadFull(r, body); err != nil {
			t.Fatalf("read bulk body: %v", err)
		}
	}
}
