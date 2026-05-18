package store

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

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

func parseRangeID(id string, defaultSeq int64) (ms, seq int64, err error) {
	if id == "-" {
		return 0, 0, nil
	}
	if id == "+" {
		return math.MaxInt64, math.MaxInt64, nil
	}
	if strings.Contains(id, "-") {
		ms, seq, err = parseStreamID(id)
		return
	}
	ms, err = strconv.ParseInt(id, 10, 64)
	seq = defaultSeq
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
