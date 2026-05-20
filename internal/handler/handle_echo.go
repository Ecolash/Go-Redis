package handler

import "github.com/codecrafters-io/redis-starter-go/internal/resp"

func (h *Handler) handleEcho(parts []string) string {
	if len(parts) < 2 {
		return errWrongArgs
	}
	return resp.BulkString(parts[1])
}
