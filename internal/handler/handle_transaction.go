package handler

import (
	"github.com/codecrafters-io/redis-starter-go/internal/errs"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func (h *Handler) handleMulti(_ []string) string {
	h.inMulti = true
	h.queue = nil
	return okResponse
}

func (h *Handler) handleDiscard(_ []string) string {
	if !h.inMulti {
		return errs.DiscardNoMulti
	}
	h.inMulti = false
	h.queue = nil
	h.watching = nil
	return okResponse
}

func (h *Handler) handleExec(_ []string) string {
	if !h.inMulti {
		return errs.ExecNoMulti
	}
	queued := h.queue
	watching := h.watching
	h.inMulti = false
	h.queue = nil
	h.watching = nil

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

func (h *Handler) handleUnwatch(_ []string) string {
	h.watching = nil
	return okResponse
}

func (h *Handler) handleWatch(parts []string) string {
	if len(parts) < 2 {
		return errs.WrongArgs
	}
	if h.inMulti {
		return errs.WatchInMulti
	}
	if h.watching == nil {
		h.watching = make(map[string]uint64)
	}
	for _, key := range parts[1:] {
		h.watching[key] = h.store.Version(key)
	}
	return okResponse
}
