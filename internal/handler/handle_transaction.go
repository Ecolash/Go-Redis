package handler

import "github.com/codecrafters-io/redis-starter-go/internal/resp"

func (h *Handler) handleMulti(_ []string) string {
	h.inMulti = true
	h.queue = nil
	return okResponse
}

func (h *Handler) handleDiscard(_ []string) string {
	if !h.inMulti {
		return errDiscardNoMulti
	}
	h.inMulti = false
	h.queue = nil
	h.watching = nil
	return okResponse
}

func (h *Handler) handleExec(_ []string) string {
	if !h.inMulti {
		return errExecNoMulti
	}
	queued := h.queue
	watching := h.watching
	h.inMulti = false
	h.queue = nil
	h.watching = nil

	// If any watched key's version changed since WATCH, abort the transaction.
	for key, snapshot := range watching {
		if h.store.Version(key) != snapshot {
			return nullArray
		}
	}

	responses := make([]string, len(queued))
	for i, parts := range queued {
		responses[i] = h.dispatch(parts)
	}
	return resp.RawArray(responses)
}

// handleWatch snapshots current versions of the given keys. Subsequent EXEC
// aborts if any of those versions change. WATCH inside MULTI is rejected.
// Calling WATCH again accumulates keys; later WATCH on the same key
// re-snapshots it (matching Redis semantics).
func (h *Handler) handleWatch(parts []string) string {
	if len(parts) < 2 {
		return errWrongArgs
	}
	if h.inMulti {
		return errWatchInMulti
	}
	if h.watching == nil {
		h.watching = make(map[string]uint64)
	}
	for _, key := range parts[1:] {
		h.watching[key] = h.store.Version(key)
	}
	return okResponse
}
