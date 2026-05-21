package server

import (
	"io"
	"log"
	"net"

	"github.com/codecrafters-io/redis-starter-go/internal/handler"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
	"github.com/codecrafters-io/redis-starter-go/internal/store"
)

type Server struct {
	listener   net.Listener
	store      *store.Store
	role       string
	masterAddr string
	replicas   *Replicas
}

func New(addr, role, masterAddr string) (*Server, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &Server{
		listener:   l,
		store:      store.New(),
		role:       role,
		masterAddr: masterAddr,
		replicas:   newReplicas(),
	}, nil
}

func (s *Server) Addr() string {
	return s.listener.Addr().String()
}

func (s *Server) Close() error {
	return s.listener.Close()
}

func (s *Server) Run() {
	if s.role == "slave" {
		go s.handshakeWithMaster()
	}
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	handedOff := false
	defer func() {
		if !handedOff {
			conn.Close()
		}
	}()

	propagate := func(parts []string) {
		s.replicas.Broadcast([]byte(resp.Array(parts)))
	}
	h := handler.New(s.store, s.role,
		handler.WithPropagate(propagate),
		handler.WithReplicaCount(s.replicas.Count),
		handler.WithReplicaWaiter(s.replicas.Wait),
	)

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
		if h.BecameReplica() {
			s.replicas.Add(conn)
			handedOff = true
			return
		}
	}
}
