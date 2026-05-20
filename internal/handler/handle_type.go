package handler

import "github.com/codecrafters-io/redis-starter-go/internal/errs"

func (h *Handler) handleType(parts []string) string {
	if len(parts) < 2 {
		return errs.WrongArgs
	}
	return "+" + h.store.Type(parts[1]) + "\r\n"
}
