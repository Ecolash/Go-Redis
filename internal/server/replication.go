package server

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/internal/handler"
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

	initSteps := [][]string{
		{"PING"},
		{"REPLCONF", "listening-port", port},
		{"REPLCONF", "capa", "psync2"},
	}
	for _, cmd := range initSteps {
		if err := sendAndAwait(conn, r, cmd); err != nil {
			log.Printf("replication: %s failed: %v", cmd[0], err)
			return
		}
	}

	if err := completePsync(conn, r); err != nil {
		log.Printf("replication: PSYNC failed: %v", err)
		return
	}

	s.processPropagated(conn, r)
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

// completePsync sends PSYNC and consumes both the +FULLRESYNC line and the
// trailing RDB bulk payload that the master streams immediately after.
func completePsync(conn net.Conn, r *bufio.Reader) error {
	if _, err := conn.Write([]byte(resp.Array([]string{"PSYNC", "?", "-1"}))); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	line, err := r.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read FULLRESYNC: %w", err)
	}
	if len(line) == 0 || line[0] != '+' {
		return fmt.Errorf("unexpected PSYNC reply %q", line)
	}

	header, err := r.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read RDB header: %w", err)
	}
	if len(header) == 0 || header[0] != '$' {
		return fmt.Errorf("unexpected RDB header %q", header)
	}
	n, err := strconv.Atoi(strings.TrimRight(header[1:], "\r\n"))
	if err != nil {
		return fmt.Errorf("invalid RDB length %q: %w", header, err)
	}
	// RDB payload is a length-prefixed bulk WITHOUT trailing CRLF.
	if _, err := io.CopyN(io.Discard, r, int64(n)); err != nil {
		return fmt.Errorf("read RDB body: %w", err)
	}
	return nil
}

func (s *Server) processPropagated(conn net.Conn, r *bufio.Reader) {
	h := handler.New(s.store, s.role)
	for {
		cmd, err := readArray(r)
		if err != nil {
			if err != io.EOF {
				log.Printf("replication: read propagated command: %v", err)
			}
			return
		}
		reply := h.Handle(cmd)
		if h.ShouldReplyToMaster() {
			if _, werr := conn.Write([]byte(reply)); werr != nil {
				log.Printf("replication: write reply to master: %v", werr)
				return
			}
		}
	}
}

func readArray(r *bufio.Reader) ([]byte, error) {
	header, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if len(header) == 0 || header[0] != '*' {
		return nil, fmt.Errorf("expected array, got %q", header)
	}
	count, err := strconv.Atoi(strings.TrimRight(header[1:], "\r\n"))
	if err != nil {
		return nil, fmt.Errorf("invalid array length %q: %w", header, err)
	}
	buf := []byte(header)
	for i := 0; i < count; i++ {
		lenLine, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		if len(lenLine) == 0 || lenLine[0] != '$' {
			return nil, fmt.Errorf("expected bulk, got %q", lenLine)
		}
		size, err := strconv.Atoi(strings.TrimRight(lenLine[1:], "\r\n"))
		if err != nil {
			return nil, fmt.Errorf("invalid bulk length %q: %w", lenLine, err)
		}
		buf = append(buf, lenLine...)
		body := make([]byte, size+2)
		if _, err := io.ReadFull(r, body); err != nil {
			return nil, err
		}
		buf = append(buf, body...)
	}
	return buf, nil
}
