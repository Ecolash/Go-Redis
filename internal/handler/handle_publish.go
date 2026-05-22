package handler

import (
	"github.com/codecrafters-io/redis-starter-go/internal/errs"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func (h *Handler) handlePublish(parts []string) string {
	if len(parts) != 3 {
		return errs.WrongArgs
	}
	channel, message := parts[1], parts[2]
	encoded := "*3\r\n" + resp.BulkString("message") + resp.BulkString(channel) + resp.BulkString(message)
	count := h.pubsub.Publish(channel, encoded)
	return resp.Integer(count)
}
