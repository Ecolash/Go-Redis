package handler

import (
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func (h *Handler) handlePing(_ []string) string {
	if h.inSubscribe {
		return resp.Array([]string{"PONG", ""})
	}
	return "+PONG\r\n"
}
