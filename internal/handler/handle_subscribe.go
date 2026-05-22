package handler

import (
	"strings"

	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func (h *Handler) handleSubscribe(parts []string) string {
	if len(parts) < 2 {
		return "-ERR wrong number of arguments for 'subscribe' command\r\n"
	}
	var sb strings.Builder
	for _, channel := range parts[1:] {
		count := h.pubsub.Subscribe(h.subscriberID, channel)
		sb.WriteString("*3\r\n")
		sb.WriteString(resp.BulkString("subscribe"))
		sb.WriteString(resp.BulkString(channel))
		sb.WriteString(resp.Integer(count))
	}
	return sb.String()
}
