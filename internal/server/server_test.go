package server_test

import (
	"bufio"
	"net"
	"testing"
	"time"

	"github.com/codecrafters-io/redis-starter-go/internal/server"
)

func newTestServer(t *testing.T) (*server.Server, *bufio.Reader, net.Conn) {
	t.Helper()
	srv, err := server.New("127.0.0.1:0", "master")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	t.Cleanup(func() { srv.Close() })
	go srv.Run()

	conn, err := net.DialTimeout("tcp", srv.Addr(), time.Second)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	return srv, bufio.NewReader(conn), conn
}

func TestServerRespondsToPing(t *testing.T) {
	_, reader, conn := newTestServer(t)

	conn.Write([]byte("*1\r\n$4\r\nPING\r\n"))

	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}
	if line != "+PONG\r\n" {
		t.Errorf("expected +PONG\\r\\n, got %q", line)
	}
}

func TestServerReturnsEchoArgument(t *testing.T) {
	_, reader, conn := newTestServer(t)

	conn.Write([]byte("*2\r\n$4\r\nECHO\r\n$3\r\nhey\r\n"))

	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("failed to read first line: %v", err)
	}
	if line != "$3\r\n" {
		t.Errorf("expected $3\\r\\n, got %q", line)
	}
	body, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	if body != "hey\r\n" {
		t.Errorf("expected hey\\r\\n, got %q", body)
	}
}

func TestServerSetReturnsOK(t *testing.T) {
	_, reader, conn := newTestServer(t)

	conn.Write([]byte("*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n"))

	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}
	if line != "+OK\r\n" {
		t.Errorf("expected +OK\\r\\n, got %q", line)
	}
}

func TestServerGetReturnsValueAfterSet(t *testing.T) {
	_, reader, conn := newTestServer(t)

	conn.Write([]byte("*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n"))
	reader.ReadString('\n') // consume +OK\r\n

	conn.Write([]byte("*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n"))

	header, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("failed to read header: %v", err)
	}
	if header != "$3\r\n" {
		t.Errorf("expected $3\\r\\n, got %q", header)
	}
	body, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	if body != "bar\r\n" {
		t.Errorf("expected bar\\r\\n, got %q", body)
	}
}

func TestServerGetReturnsNullBulkForMissingKey(t *testing.T) {
	_, reader, conn := newTestServer(t)

	conn.Write([]byte("*2\r\n$3\r\nGET\r\n$7\r\nunknown\r\n"))

	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}
	if line != "$-1\r\n" {
		t.Errorf("expected $-1\\r\\n, got %q", line)
	}
}
