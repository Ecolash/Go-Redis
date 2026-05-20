package store

import (
	"strconv"
	"time"

	"github.com/codecrafters-io/redis-starter-go/internal/errs"
)

func parseInt(s string) (int64, error) {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, errs.ErrNotInteger
	}
	return n, nil
}

func intToStr(n int64) string {
	return strconv.FormatInt(n, 10)
}

func (s *Store) Set(key, value string, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := entry{kind: kindString, strVal: value}
	if ttl > 0 {
		e.expiresAt = time.Now().Add(ttl)
	}
	s.data[key] = e
	s.bumpVersionLocked(key)
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

func (s *Store) Incr(key string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.data[key]
	if ok && e.kind != kindString {
		return 0, errs.ErrWrongType
	}
	var val int64
	if ok {
		var err error
		val, err = parseInt(e.strVal)
		if err != nil {
			return 0, err
		}
	}
	val++
	s.data[key] = entry{kind: kindString, strVal: intToStr(val)}
	s.bumpVersionLocked(key)
	return val, nil
}

func (s *Store) Decr(key string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.data[key]
	if ok && e.kind != kindString {
		return 0, errs.ErrWrongType
	}
	var val int64
	if ok {
		var err error
		val, err = parseInt(e.strVal)
		if err != nil {
			return 0, err
		}
	}
	val--
	s.data[key] = entry{kind: kindString, strVal: intToStr(val)}
	s.bumpVersionLocked(key)
	return val, nil
}
