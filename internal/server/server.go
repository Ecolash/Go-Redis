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
	"github.com/codecrafters-io/redis-starter-go/internal/pubsub"
	"github.com/codecrafters-io/redis-starter-go/internal/rdb"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
	"github.com/codecrafters-io/redis-starter-go/internal/store"
)

type Server struct {
	listener        net.Listener
	store           *store.Store
	role            string
	masterAddr      string
	replicas        *Replicas
	pubsub          *pubsub.PubSub
	dir             string
	dbfilename      string
	configOverrides map[string]string
	config          map[string]string
	aofWriter       *aof.Writer
}

type ServerOption func(*Server)

func WithDir(dir string) ServerOption {
	return func(s *Server) { s.dir = dir }
}

func WithDBFilename(name string) ServerOption {
	return func(s *Server) { s.dbfilename = name }
}

// WithConfigOverrides sets config values that take precedence over defaults
// (e.g. AOF options supplied via command-line flags).
func WithConfigOverrides(overrides map[string]string) ServerOption {
	return func(s *Server) { s.configOverrides = overrides }
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
		pubsub:     pubsub.New(),
	}
	for _, opt := range opts {
		opt(s)
	}
	if s.dir == "" {
		if cwd, err := os.Getwd(); err == nil {
			s.dir = cwd
		}
	}
	s.config = aof.Defaults()
	s.config["dir"] = s.dir
	s.config["dbfilename"] = s.dbfilename
	for k, v := range s.configOverrides {
		s.config[k] = v
	}
	if s.config["appendonly"] == "yes" {
		if err := aof.Setup(s.dir, s.config["appenddirname"], s.config["appendfilename"]); err != nil {
			log.Printf("aof: failed to set up append-only directory: %v", err)
		} else if w, err := aof.NewWriter(s.dir, s.config["appenddirname"], s.config["appendfilename"]); err != nil {
			log.Printf("aof: failed to open append-only file: %v", err)
		} else {
			s.aofWriter = w
		}
	}
	if s.dir != "" && s.dbfilename != "" {
		s.loadRDB(filepath.Join(s.dir, s.dbfilename))
	}
	if s.config["appendonly"] == "yes" {
		if err := aof.Replay(s.dir, s.config["appenddirname"], s.config["appendfilename"], func(cmd []byte) error {
			h := handler.New(s.store, s.role,
				handler.WithPropagate(func(parts []string) {}),
				handler.WithReplicaCount(func() int { return 0 }),
				handler.WithReplicaWaiter(func(numReplicas int, timeout time.Duration) int { return 0 }),
				handler.WithConfig("dir", s.dir),
				handler.WithConfig("dbfilename", s.dbfilename),
			)
			h.Handle(cmd)
			return nil
		}); err != nil {
			log.Printf("aof: failed to replay append-only file: %v", err)
		}
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
		handler.WithPubSub(s.pubsub),
		handler.WithSubscriberID(conn.RemoteAddr().String()),
	}
	for k, v := range s.config {
		opts = append(opts, handler.WithConfig(k, v))
	}
	if s.aofWriter != nil {
		opts = append(opts, handler.WithAOFAppend(func(parts []string) {
			if err := s.aofWriter.Append(resp.Array(parts)); err != nil {
				log.Printf("aof: append failed: %v", err)
			}
		}))
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
		if h.InSubscribeMode() {
			s.handleSubscribedConn(conn, h)
			return
		}
	}
}

func (s *Server) handleSubscribedConn(conn net.Conn, h *handler.Handler) {
	defer conn.Close()

	cmds := make(chan []byte, 1)
	go func() {
		buf := make([]byte, 512)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				close(cmds)
				return
			}
			tmp := make([]byte, n)
			copy(tmp, buf[:n])
			cmds <- tmp
		}
	}()

	msgs := h.MessageChan()
	for {
		select {
		case data, ok := <-cmds:
			if !ok {
				return
			}
			response := h.Handle(data)
			if _, err := conn.Write([]byte(response)); err != nil {
				return
			}
		case msg := <-msgs:
			if _, err := conn.Write([]byte(msg)); err != nil {
				return
			}
		}
	}
}
