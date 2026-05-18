package store

import (
	"sync"
	"time"
)

type Store struct {
	mu      sync.RWMutex
	data    map[string]entry
	wmu     sync.Mutex // always acquired after mu when both are needed
	waiters map[string][]*blpopWaiter
}

func New() *Store {
	return &Store{
		data:    make(map[string]entry),
		waiters: make(map[string][]*blpopWaiter),
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
