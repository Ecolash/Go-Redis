package server_test

import (
	"bufio"
	"net"
	"testing"
	"time"

	"github.com/codecrafters-io/redis-starter-go/internal/server"
)

func TestServerRespondsToPing(t *testing.T) {
	srv, err := server.New("127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer srv.Close()

	go srv.Run()

	conn, err := net.DialTimeout("tcp", srv.Addr(), time.Second)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	_, err = conn.Write([]byte("PING\r\n"))
	if err != nil {
		t.Fatalf("failed to send PING: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(time.Second))
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	if line != "+PONG\r\n" {
		t.Errorf("expected +PONG\\r\\n, got %q", line)
	}
}
