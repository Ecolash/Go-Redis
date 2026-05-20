package server

import (
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

	if _, err := conn.Write([]byte(resp.Array([]string{"PING"}))); err != nil {
		log.Printf("replication: failed to send PING: %v", err)
		return
	}
}
