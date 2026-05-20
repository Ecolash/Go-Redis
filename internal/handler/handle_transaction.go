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
	return okResponse
}

func (h *Handler) handleExec(_ []string) string {
	if !h.inMulti {
		return errExecNoMulti
	}
	queued := h.queue
	h.inMulti = false
	h.queue = nil

	responses := make([]string, len(queued))
	for i, parts := range queued {
		responses[i] = h.dispatch(parts)
	}
	return resp.RawArray(responses)
}
