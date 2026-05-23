package store

// ZSetMember is a single member of a sorted set.
type ZSetMember struct {
	Score  float64
	Member string
}

// ZAdd adds or updates members in the sorted set at key.
// Returns the count of newly added members (updates are not counted).
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

// ZRange returns members in rank [start, stop] (0-based, inclusive).
// Negative indices are resolved Redis-style: -1 is the last element.
// Returns an empty slice if the key does not exist.
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

// ZRank returns the 0-based rank of member in the sorted set at key.
// ok is false if the key or member does not exist.
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

// ZScore returns the score of member in the sorted set at key.
// ok is false if the key or member does not exist.
func (s *Store) ZScore(key string, member string) (float64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.data[key]
	if !ok || e.kind != kindZSet {
		return 0, false
	}
	return e.zsetVal.score(member)
}

// ZRem removes members from the sorted set at key.
// Returns the number of members actually removed.
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