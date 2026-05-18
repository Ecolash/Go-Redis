package store

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
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

func parseStreamID(id string) (ms, seq int64, err error) {
	parts := strings.SplitN(id, "-", 2)
	if len(parts) != 2 {
		return 0, 0, errors.New("invalid stream ID")
	}
	ms, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return
	}
	seq, err = strconv.ParseInt(parts[1], 10, 64)
	return
}

func (s *Store) XAdd(key, id string, fields []string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	e := s.data[key]
	if e.kind != kindStream {
		e = entry{kind: kindStream}
	}

	if id == "*" {
		ms := time.Now().UnixMilli()
		seq := int64(0)
		if len(e.streamVal) > 0 {
			lastMs, lastSeq, _ := parseStreamID(e.streamVal[len(e.streamVal)-1].ID)
			if lastMs == ms {
				seq = lastSeq + 1
			}
		}
		id = fmt.Sprintf("%d-%d", ms, seq)
	} else if strings.HasSuffix(id, "-*") {
		newMs, err := strconv.ParseInt(id[:len(id)-2], 10, 64)
		if err != nil {
			return "", err
		}
		seq := int64(0)
		if newMs == 0 {
			seq = 1
		}
		if len(e.streamVal) > 0 {
			lastMs, lastSeq, _ := parseStreamID(e.streamVal[len(e.streamVal)-1].ID)
			if newMs < lastMs {
				return "", errStreamIDSmall
			}
			if newMs == lastMs {
				seq = lastSeq + 1
			}
		}
		id = fmt.Sprintf("%d-%d", newMs, seq)
	} else {
		newMs, newSeq, err := parseStreamID(id)
		if err != nil {
			return "", err
		}
		if newMs == 0 && newSeq == 0 {
			return "", errStreamIDZero
		}
		if len(e.streamVal) > 0 {
			lastMs, lastSeq, _ := parseStreamID(e.streamVal[len(e.streamVal)-1].ID)
			if newMs < lastMs || (newMs == lastMs && newSeq <= lastSeq) {
				return "", errStreamIDSmall
			}
		}
	}

	e.streamVal = append(e.streamVal, StreamEntry{ID: id, Fields: fields})
	s.data[key] = e
	return id, nil
}

// parseRangeID parses a stream ID for XRANGE bounds.
// If the ID has no "-", defaultSeq is used as the sequence number.
func parseRangeID(id string, defaultSeq int64) (ms, seq int64, err error) {
	if strings.Contains(id, "-") {
		ms, seq, err = parseStreamID(id)
		return
	}
	ms, err = strconv.ParseInt(id, 10, 64)
	seq = defaultSeq
	return
}

func (s *Store) XRange(key, startID, endID string) ([]StreamEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	e, ok := s.data[key]
	if !ok || e.kind != kindStream {
		return []StreamEntry{}, nil
	}

	startMs, startSeq, err := parseRangeID(startID, 0)
	if err != nil {
		return nil, err
	}
	endMs, endSeq, err := parseRangeID(endID, math.MaxInt64)
	if err != nil {
		return nil, err
	}

	var result []StreamEntry
	for _, entry := range e.streamVal {
		ms, seq, _ := parseStreamID(entry.ID)
		if (ms > startMs || (ms == startMs && seq >= startSeq)) &&
			(ms < endMs || (ms == endMs && seq <= endSeq)) {
			result = append(result, entry)
		}
	}
	return result, nil
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
