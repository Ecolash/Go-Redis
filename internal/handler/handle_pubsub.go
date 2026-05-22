package handler

import (
	"strings"

	"github.com/codecrafters-io/redis-starter-go/internal/errs"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func (h *Handler) handleSubscribe(parts []string) string {
	if len(parts) < 2 {
		return "-ERR wrong number of arguments for 'subscribe' command\r\n"
	}
	h.inSubscribe = true
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


func (h *Handler) handlePublish(parts []string) string {
	if len(parts) != 3 {
		return errs.WrongArgs
	}
	channel, message := parts[1], parts[2]
	encoded := "*3\r\n" + resp.BulkString("message") + resp.BulkString(channel) + resp.BulkString(message)
	count := h.pubsub.Publish(channel, encoded)
	return resp.Integer(count)
}

func (h *Handler) handleUnsubscribe(parts []string) string {
	if len(parts) < 2 {
		return "-ERR wrong number of arguments for 'unsubscribe' command\r\n"
	}
	var sb strings.Builder	
	channel := parts[1]
	count := h.pubsub.Unsubscribe(h.subscriberID, channel)
	sb.WriteString("*3\r\n")
	sb.WriteString(resp.BulkString("unsubscribe"))
	sb.WriteString(resp.BulkString(channel))
	sb.WriteString(resp.Integer(count))
	if count == 0 {
		h.inSubscribe = false
	}
	return sb.String()
}

	
