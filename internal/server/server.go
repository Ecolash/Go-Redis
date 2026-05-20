package server

import (
	"io"
	"log"
	"net"

	"github.com/codecrafters-io/redis-starter-go/internal/handler"
	"github.com/codecrafters-io/redis-starter-go/internal/store"
)

type Server struct {
	listener net.Listener
	store    *store.Store
	role     string
}

func New(addr, role string) (*Server, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &Server{
		listener: l,
		store:    store.New(),
		role:     role,
	}, nil
}

func (s *Server) Addr() string {
	return s.listener.Addr().String()
}

func (s *Server) Close() error {
	return s.listener.Close()
}

func (s *Server) Run() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	h := handler.New(s.store, s.role)
	buf := make([]byte, 512)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("read error: %v", err)
			}
			return
		}
		response := h.Handle(buf[:n])
		if _, err := conn.Write([]byte(response)); err != nil {
			log.Printf("write error: %v", err)
			return
		}
	}
}
