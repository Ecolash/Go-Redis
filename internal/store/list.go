package store

import "time"

func (s *Store) RPush(key string, vals ...string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.data[key]
	if e.kind != kindList {
		e = entry{kind: kindList}
	}
	e.listVal = append(e.listVal, vals...)
	s.data[key] = e
	s.bumpVersionLocked(key)
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
	s.bumpVersionLocked(key)
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
	s.bumpVersionLocked(key)
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
	s.bumpVersionLocked(key)
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
	if count > 0 {
		s.bumpVersionLocked(key)
	}
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
	if count > 0 {
		s.bumpVersionLocked(key)
	}
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

func (s *Store) BLPopWait(keys []string) (<-chan BLPOPResult, func()) {
	w := &blpopWaiter{
		keys: keys,
		ch:   make(chan BLPOPResult, 1),
	}

	s.mu.Lock()
	for _, key := range keys {
		e, ok := s.data[key]
		if !ok || e.kind != kindList || len(e.listVal) == 0 {
			continue
		}
		val := e.listVal[0]
		e.listVal = e.listVal[1:]
		s.data[key] = e
		s.bumpVersionLocked(key)
		w.ch <- BLPOPResult{Key: key, Val: val}
		s.mu.Unlock()
		return w.ch, func() {}
	}

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
		s.bumpVersionLocked(key)
		for _, k := range waiter.keys {
			s.removeWaiter(k, waiter)
		}
		waiter.ch <- BLPOPResult{Key: key, Val: val}
	}
}

func (s *Store) removeWaiter(key string, w *blpopWaiter) {
	waiters := s.waiters[key]
	for i, ww := range waiters {
		if ww == w {
			s.waiters[key] = append(waiters[:i], waiters[i+1:]...)
			return
		}
	}
}
