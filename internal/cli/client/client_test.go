package client_test

import (
	"bufio"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/codecrafters-io/redis-starter-go/internal/cli/client"
)

// fakeServer starts a TCP server that responds with the same canned RESP to every line received.
func fakeServer(t *testing.T, response string) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		for {
			_, err := r.ReadString('\n')
			if err != nil {
				return
			}
			fmt.Fprint(conn, response)
		}
	}()
	return ln.Addr().String(), func() { ln.Close() }
}

func splitAddr(t *testing.T, addr string) (string, int) {
	t.Helper()
	ln, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	return ln.IP.String(), ln.Port
}

func TestDoSimpleString(t *testing.T) {
	addr, stop := fakeServer(t, "+PONG\r\n")
	defer stop()

	host, port := splitAddr(t, addr)
	c, err := client.New(host, port, "")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	resp, err := c.Do("PING")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Value != "PONG" {
		t.Fatalf("expected PONG, got %q", resp.Value)
	}
}

func TestLatencyTracked(t *testing.T) {
	addr, stop := fakeServer(t, "+OK\r\n")
	defer stop()

	host, port := splitAddr(t, addr)
	c, err := client.New(host, port, "")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	c.Do("PING")
	if c.Latency() < 0 || c.Latency() > time.Second {
		t.Fatalf("latency out of range: %v", c.Latency())
	}
}
