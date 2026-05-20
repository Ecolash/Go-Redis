package handler

import (
	"strconv"
	"strings"
	"time"

	"github.com/codecrafters-io/redis-starter-go/internal/resp"
	"github.com/codecrafters-io/redis-starter-go/internal/store"
)

func (h *Handler) handleXAdd(parts []string) string {
	// XADD key id field value [field value ...]
	if len(parts) < 5 || (len(parts)-3)%2 != 0 {
		return errWrongArgs
	}
	id, err := h.store.XAdd(parts[1], parts[2], parts[3:])
	if err != nil {
		return resp.Error(err.Error())
	}
	return resp.BulkString(id)
}

func (h *Handler) handleXRange(parts []string) string {
	if len(parts) < 4 {
		return errWrongArgs
	}
	entries, err := h.store.XRange(parts[1], parts[2], parts[3])
	if err != nil {
		return resp.Error(err.Error())
	}
	respEntries := make([]resp.Entry, len(entries))
	for i, e := range entries {
		respEntries[i] = resp.Entry{ID: e.ID, Fields: e.Fields}
	}
	return resp.StreamEntries(respEntries)
}

func (h *Handler) handleXRead(parts []string) string {
	// XREAD [BLOCK <ms>] STREAMS <key1>...<keyN> <id1>...<idN>
	if len(parts) < 4 {
		return errWrongArgs
	}

	blocking := strings.EqualFold(parts[1], "BLOCK")
	if blocking {
		return h.handleXReadBlocking(parts)
	}

	if !strings.EqualFold(parts[1], "STREAMS") {
		return errWrongArgs
	}
	return h.xreadStreams(parts[2:])
}

func (h *Handler) handleXReadBlocking(parts []string) string {
	// parts: [XREAD, BLOCK, <ms>, STREAMS, <key>, <id>]
	if len(parts) < 6 || !strings.EqualFold(parts[3], "STREAMS") {
		return errWrongArgs
	}
	ms, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || ms < 0 {
		return errWrongArgs
	}

	key, afterID := parts[4], parts[5]

	ch, cancel := h.store.XReadWait(key, afterID)
	defer cancel()

	var entries []store.StreamEntry
	if ms == 0 {
		entries = <-ch
	} else {
		timer := time.NewTimer(time.Duration(ms) * time.Millisecond)
		defer timer.Stop()
		select {
		case entries = <-ch:
		case <-timer.C:
			return nullArray
		}
	}

	respEntries := make([]resp.Entry, len(entries))
	for i, e := range entries {
		respEntries[i] = resp.Entry{ID: e.ID, Fields: e.Fields}
	}
	return resp.StreamResults([]resp.XReadResult{{Key: key, Entries: respEntries}})
}

func (h *Handler) xreadStreams(rest []string) string {
	if len(rest)%2 != 0 {
		return errWrongArgs
	}
	n := len(rest) / 2
	keys := rest[:n]
	afterIDs := rest[n:]

	var results []resp.XReadResult
	for i, key := range keys {
		entries, err := h.store.XRead(key, afterIDs[i])
		if err != nil {
			return resp.Error(err.Error())
		}
		if len(entries) == 0 {
			continue
		}
		respEntries := make([]resp.Entry, len(entries))
		for j, e := range entries {
			respEntries[j] = resp.Entry{ID: e.ID, Fields: e.Fields}
		}
		results = append(results, resp.XReadResult{Key: key, Entries: respEntries})
	}
	if len(results) == 0 {
		return nullArray
	}
	return resp.StreamResults(results)
}
