package store

// BLPopWait atomically checks each key in priority order for an available element.
// If found, the result is pre-loaded into the returned channel (no blocking needed).
// If not, the caller is registered as a waiter; the channel receives a value when
// any push delivers an element. The returned cancel must be called on timeout/cleanup.
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
