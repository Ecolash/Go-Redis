package store

func (s *Store) ZAdd(key string, members []ZSetMember) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.data[key]
	if !ok || e.kind != kindZSet {
		e = entry{kind: kindZSet, zsetVal: newSkipList()}
	}
	added := 0
	for _, m := range members {
		if e.zsetVal.insert(m.Score, m.Member) {
			added++
		}
	}
	s.data[key] = e
	if added > 0 {
		s.bumpVersionLocked(key)
	}
	return added
}

func (s *Store) ZRange(key string, start, stop int) []ZSetMember {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.data[key]
	if !ok || e.kind != kindZSet {
		return []ZSetMember{}
	}
	n := e.zsetVal.length
	if start < 0 {
		start = n + start
	}
	if stop < 0 {
		stop = n + stop
	}
	if start < 0 {
		start = 0
	}
	return e.zsetVal.rangeByRank(start, stop)
}

func (s *Store) ZRank(key string, member string) (int, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.data[key]
	if !ok || e.kind != kindZSet {
		return 0, false
	}
	r := e.zsetVal.rank(member)
	if r == -1 {
		return 0, false
	}
	return r, true
}

func (s *Store) ZScore(key string, member string) (float64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.data[key]
	if !ok || e.kind != kindZSet {
		return 0, false
	}
	return e.zsetVal.score(member)
}

func (s *Store) ZRem(key string, members []string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.data[key]
	if !ok || e.kind != kindZSet {
		return 0
	}
	removed := 0
	for _, m := range members {
		score, exists := e.zsetVal.scores[m]
		if !exists {
			continue
		}
		if e.zsetVal.remove(score, m) {
			removed++
		}
	}
	if removed > 0 {
		s.bumpVersionLocked(key)
	}
	return removed
}

func (s *Store) ZCard(key string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.data[key]
	if !ok || e.kind != kindZSet {
		return 0
	}
	return e.zsetVal.length
}
