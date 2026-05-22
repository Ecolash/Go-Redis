package handler

import (
	"github.com/codecrafters-io/redis-starter-go/internal/errs"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func (h *Handler) handleKeys(parts []string) string {
	if len(parts) < 2 {
		return errs.WrongArgs
	}
	keys := h.store.Keys(parts[1])
	return resp.Array(keys)
}
