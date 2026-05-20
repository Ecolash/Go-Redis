package handler

import (
	"github.com/codecrafters-io/redis-starter-go/internal/errs"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func (h *Handler) handleGet(parts []string) string {
	if len(parts) < 2 {
		return errs.WrongArgs
	}
	val, ok := h.store.Get(parts[1])
	if !ok {
		return nullBulk
	}
	return resp.BulkString(val)
}
