package server

import (
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/codecrafters-io/redis-starter-go/internal/aof"
	"github.com/codecrafters-io/redis-starter-go/internal/handler"
	"github.com/codecrafters-io/redis-starter-go/internal/rdb"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
	"github.com/codecrafters-io/redis-starter-go/internal/store"
)

type Server struct {
	listener   net.Listener
	store      *store.Store
	role       string
	masterAddr string
	replicas   *Replicas
	dir        string
	dbfilename string
}

type ServerOption func(*Server)

func WithDir(dir string) ServerOption {
	return func(s *Server) { s.dir = dir }
}

func WithDBFilename(name string) ServerOption {
	return func(s *Server) { s.dbfilename = name }
}

func New(addr, role, masterAddr string, opts ...ServerOption) (*Server, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	s := &Server{
		listener:   l,
		store:      store.New(),
		role:       role,
		masterAddr: masterAddr,
		replicas:   newReplicas(),
	}
	for _, opt := range opts {
		opt(s)
	}
	if s.dir == "" {
		if cwd, err := os.Getwd(); err == nil {
			s.dir = cwd
		}
	}
	if s.dir != "" && s.dbfilename != "" {
		s.loadRDB(filepath.Join(s.dir, s.dbfilename))
	}
	return s, nil
}

func (s *Server) loadRDB(path string) {
	entries, err := rdb.Load(path)
	if err != nil {
		log.Printf("rdb: failed to load %s: %v", path, err)
		return
	}
	now := time.Now()
	for _, e := range entries {
		var ttl time.Duration
		if !e.ExpiresAt.IsZero() {
			ttl = e.ExpiresAt.Sub(now)
			if ttl <= 0 {
				continue // already expired
			}
		}
		s.store.Set(e.Key, e.Value, ttl)
	}
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
	opts := []handler.Option{
		handler.WithPropagate(propagate),
		handler.WithReplicaCount(s.replicas.Count),
		handler.WithReplicaWaiter(s.replicas.Wait),
		handler.WithConfig("dir", s.dir),
		handler.WithConfig("dbfilename", s.dbfilename),
	}
	for k, v := range aof.Defaults() {
		opts = append(opts, handler.WithConfig(k, v))
	}
	h := handler.New(s.store, s.role, opts...)

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
