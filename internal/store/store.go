package store

import (
	"sync"
	"time"
)

// LOCKING STRATEGY
// - s.mu  protects all access to s.data and s.waiters, including checking expirations.
// - s.wmu protects only the logic around waiting and notifying BLPOP waiters. 

type Store struct {
	data    map[string]entry
	mu      sync.RWMutex
	wmu     sync.Mutex

	versions map[string]uint64
	waiters map[string][]*blpopWaiter
	xreadWaiters map[string][]*xreadWaiter
}

func New() *Store {
	return &Store{
		data:    make(map[string]entry),
		versions: make(map[string]uint64),
		waiters: make(map[string][]*blpopWaiter),
		xreadWaiters: make(map[string][]*xreadWaiter),
	}
}

func (s *Store) Version(key string) uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.versions[key]
}

func (s *Store) bumpVersionLocked(key string) {
	// NOTE: To be called only when s.mu() is LOCKED
	s.versions[key]++
}

func (s *Store) Keys(pattern string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now()
	out := make([]string, 0, len(s.data))
	for k, e := range s.data {
		if !e.expiresAt.IsZero() && now.After(e.expiresAt) {
			continue
		}
		if pattern == "*" || pattern == k {
			out = append(out, k)
		}
	}
	return out
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
