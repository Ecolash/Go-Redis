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

// BLPOPResult is the key+value returned by a blocking pop.
type BLPOPResult struct {
	Key string
	Val string
}

// blpopWaiter tracks a single pending BLPOP across one or more keys.
type blpopWaiter struct {
	keys []string
	ch   chan BLPOPResult // buffered size 1
}

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

func (s *Store) RPush(key string, vals ...string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.data[key]
	if e.kind != kindList {
		e = entry{kind: kindList}
	}
	e.listVal = append(e.listVal, vals...)
	s.data[key] = e
	n := len(e.listVal)
	s.deliverToWaitersLocked(key)
	return n
}

func (s *Store) LPush(key string, vals ...string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.data[key]
	if e.kind != kindList {
		e = entry{kind: kindList}
	}
	prepend := make([]string, len(vals))
	for i, v := range vals {
		prepend[len(vals)-1-i] = v
	}
	e.listVal = append(prepend, e.listVal...)
	s.data[key] = e
	n := len(e.listVal)
	s.deliverToWaitersLocked(key)
	return n
}

func (s *Store) LPop(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.data[key]
	if !ok || e.kind != kindList || len(e.listVal) == 0 {
		return "", false
	}
	val := e.listVal[0]
	e.listVal = e.listVal[1:]
	s.data[key] = e
	return val, true
}

func (s *Store) RPop(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.data[key]
	if !ok || e.kind != kindList || len(e.listVal) == 0 {
		return "", false
	}
	n := len(e.listVal)
	val := e.listVal[n-1]
	e.listVal = e.listVal[:n-1]
	s.data[key] = e
	return val, true
}

func (s *Store) LPopCount(key string, count int) ([]string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.data[key]
	if !ok || e.kind != kindList {
		return nil, false
	}
	if count > len(e.listVal) {
		count = len(e.listVal)
	}
	vals := make([]string, count)
	copy(vals, e.listVal[:count])
	e.listVal = e.listVal[count:]
	s.data[key] = e
	return vals, true
}

func (s *Store) RPopCount(key string, count int) ([]string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.data[key]
	if !ok || e.kind != kindList {
		return nil, false
	}
	n := len(e.listVal)
	if count > n {
		count = n
	}
	vals := make([]string, count)
	for i := range count {
		vals[i] = e.listVal[n-1-i]
	}
	e.listVal = e.listVal[:n-count]
	s.data[key] = e
	return vals, true
}

func (s *Store) LLen(key string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.data[key]
	if !ok || e.kind != kindList {
		return 0
	}
	return len(e.listVal)
}

func (s *Store) LRange(key string, start, stop int) ([]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.data[key]
	if !ok || e.kind != kindList {
		return nil, false
	}
	if !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
		return nil, false
	}
	n := len(e.listVal)
	if start < 0 {
		start = n + start
	}
	if stop < 0 {
		stop = n + stop
	}
	if start < 0 {
		start = 0
	}
	if stop >= n {
		stop = n - 1
	}
	if start > stop {
		return []string{}, true
	}
	return e.listVal[start : stop+1], true
}



/*
BLPopWait atomically checks each key in priority order for an available element.
If found, the result is pre-loaded into the returned channel (no blocking needed).
If not, the caller is registered as a waiter; the channel receives a value when
any push delivers an element. The returned cancel must be called on timeout/cleanup.

ch <- 10      // send 10 into channel
x := <- ch    // receive from channel

*/
func (s *Store) BLPopWait(keys []string) (<-chan BLPOPResult, func()) {
	w := &blpopWaiter{
		keys: keys,
		ch:   make(chan BLPOPResult, 1),
	}

	// Try immediate pop in key priority order.
	s.mu.Lock()
	for _, key := range keys {
		e, ok := s.data[key]
		if !ok || e.kind != kindList || len(e.listVal) == 0 {
			continue
		}
		val := e.listVal[0]
		e.listVal = e.listVal[1:]
		s.data[key] = e
		w.ch <- BLPOPResult{Key: key, Val: val}
		s.mu.Unlock()
		return w.ch, func() {}
	}

	// No immediate element, register waiter for all keys.
	s.wmu.Lock()
	for _, key := range keys {
		s.waiters[key] = append(s.waiters[key], w)
	}
	s.wmu.Unlock()
	s.mu.Unlock()

	cancel := func() {
		s.wmu.Lock()
		defer s.wmu.Unlock()
		for _, key := range keys {
			s.removeWaiter(key, w)
		}
	}
	return w.ch, cancel
}

// deliverToWaitersLocked pops elements from key and delivers them to registered waiters.
// Must be called while holding s.mu write lock.
func (s *Store) deliverToWaitersLocked(key string) {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	for {
		if len(s.waiters[key]) == 0 {
			return
		}
		e := s.data[key]
		if len(e.listVal) == 0 {
			return
		}
		waiter := s.waiters[key][0]
		val := e.listVal[0]
		e.listVal = e.listVal[1:]
		s.data[key] = e
		for _, k := range waiter.keys {
			s.removeWaiter(k, waiter)
		}
		waiter.ch <- BLPOPResult{Key: key, Val: val}
	}
}

// removeWaiter removes w from s.waiters[key]. Caller must hold s.wmu.
func (s *Store) removeWaiter(key string, w *blpopWaiter) {
	waiters := s.waiters[key]
	for i, ww := range waiters {
		if ww == w {
			s.waiters[key] = append(waiters[:i], waiters[i+1:]...)
			return
		}
	}
}
