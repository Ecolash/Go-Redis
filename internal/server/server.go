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
	handler  *handler.Handler
}

func New(addr string) (*Server, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	s := store.New()
	return &Server{
		listener: l,
		handler:  handler.New(s),
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
	buf := make([]byte, 512)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("read error: %v", err)
			}
			return
		}
		response := s.handler.Handle(buf[:n])
		if _, err := conn.Write([]byte(response)); err != nil {
			log.Printf("write error: %v", err)
			return
		}
	}
}
