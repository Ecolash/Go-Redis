package server_test

import (
	"bufio"
	"net"
	"testing"
	"time"

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
