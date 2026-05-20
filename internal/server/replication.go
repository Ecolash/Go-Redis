package server

import (
	"bufio"
	"fmt"
	"log"
	"net"

	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func (s *Server) handshakeWithMaster() {
	conn, err := net.Dial("tcp", s.masterAddr)
	if err != nil {
		log.Printf("replication: failed to dial master %s: %v", s.masterAddr, err)
		return
	}
	defer conn.Close()

	r := bufio.NewReader(conn)

	_, port, err := net.SplitHostPort(s.listener.Addr().String())
	if err != nil {
		log.Printf("replication: bad listener addr: %v", err)
		return
	}

	steps := [][]string{
		{"PING"},
		{"REPLCONF", "listening-port", port},
		{"REPLCONF", "capa", "psync2"},
		{"PSYNC", "?", "-1"},
	}
	for _, cmd := range steps {
		if err := sendAndAwait(conn, r, cmd); err != nil {
			log.Printf("replication: %s failed: %v", cmd[0], err)
			return
		}
	}
}

func sendAndAwait(conn net.Conn, r *bufio.Reader, args []string) error {
	if _, err := conn.Write([]byte(resp.Array(args))); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	line, err := r.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read reply: %w", err)
	}
	if len(line) == 0 || line[0] == '-' {
		return fmt.Errorf("unexpected reply %q", line)
	}
	return nil
}
