package store

import (
	"sync"
	"time"
)

// LOCKING STRATEGY
// - s.mu  protects all access to s.data and s.waiters, including checking expirations.
// - s.wmu protects only the logic around waiting and notifying BLPOP waiters. 

type Store struct {
	mu      sync.RWMutex
	data    map[string]entry
	wmu     sync.Mutex
	waiters map[string][]*blpopWaiter
	xreadWaiters map[string][]*xreadWaiter
}

func New() *Store {
	return &Store{
		data:    make(map[string]entry),
		waiters: make(map[string][]*blpopWaiter),
		xreadWaiters: make(map[string][]*xreadWaiter),
	}
}

func (s *Store) Type(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.data[key]
	if !ok {
		return "none"
	}
	if !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
		return "none"
	}

	kindToString := map[valueKind]string{
		kindString: "string",
		kindList:   "list",
		kindStream: "stream",
	}
	if typeStr, ok := kindToString[e.kind]; ok {
		return typeStr
	}
	return "none"
}
