package state

import (
	"sync"
	"time"
)

type Session struct {
	mu   sync.RWMutex
	Host string
	Port int

	Latency   time.Duration
	Connected bool

	InTx    bool
	TxQueue []string

	Subscriptions  []string
	PSubscriptions []string
	InPubSub       bool
}

func New(host string, port int) *Session {
	return &Session{Host: host, Port: port, Connected: true}
}

func (s *Session) EnterTx() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.InTx = true
	s.TxQueue = s.TxQueue[:0]
}

func (s *Session) ExitTx() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.InTx = false
	s.TxQueue = nil
}

func (s *Session) QueueCmd(cmd string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TxQueue = append(s.TxQueue, cmd)
}

func (s *Session) Subscribe(ch string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Subscriptions = append(s.Subscriptions, ch)
	s.InPubSub = true
}

func (s *Session) PSubscribe(pattern string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PSubscriptions = append(s.PSubscriptions, pattern)
	s.InPubSub = true
}

func (s *Session) Unsubscribe(ch string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	subs := s.Subscriptions[:0]
	for _, c := range s.Subscriptions {
		if c != ch {
			subs = append(subs, c)
		}
	}
	s.Subscriptions = subs
	s.InPubSub = len(s.Subscriptions)+len(s.PSubscriptions) > 0
}

func (s *Session) UpdateLatency(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Latency = d
}

func (s *Session) SetConnected(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Connected = v
}
