package store

import (
	"sync"
	"time"
)

type valueKind int

const (
	kindString valueKind = iota
	kindList
)

type entry struct {
	kind      valueKind
	strVal    string
	listVal   []string
	expiresAt time.Time // zero means no expiry
}

type Store struct {
	mu   sync.RWMutex
	data map[string]entry
}

func New() *Store {
	return &Store{data: make(map[string]entry)}
}

// Set stores key=value with an optional TTL. ttl=0 means the key never expires.
func (s *Store) Set(key, value string, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := entry{kind: kindString, strVal: value}
	if ttl > 0 {
		e.expiresAt = time.Now().Add(ttl)
	}
	s.data[key] = e
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.data[key]
	if !ok || e.kind != kindString {
		return "", false
	}
	if !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
		return "", false
	}
	return e.strVal, true
}

// RPush appends vals to the list at key, creating it if it doesn't exist.
// Returns the number of elements in the list after appending.
func (s *Store) RPush(key string, vals ...string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.data[key]
	if e.kind != kindList {
		e = entry{kind: kindList}
	}
	e.listVal = append(e.listVal, vals...)
	s.data[key] = e
	return len(e.listVal)
}
