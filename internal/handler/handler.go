package handler

import (
	"strings"

	"github.com/codecrafters-io/redis-starter-go/internal/command"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
	"github.com/codecrafters-io/redis-starter-go/internal/store"
)

const (
	errResponse  = "-ERR unknown command\r\n"
	errWrongArgs = "-ERR wrong number of arguments\r\n"
	okResponse   = "+OK\r\n"
	nullBulk     = "$-1\r\n"
)

type Handler struct {
	store *store.Store
}

func New(s *store.Store) *Handler {
	return &Handler{store: s}
}

// Handle parses a RESP-encoded command and returns a RESP-encoded response.
func (h *Handler) Handle(data []byte) string {
	parts, err := resp.ParseArray(data)
	if err != nil || len(parts) == 0 {
		return errResponse
	}

	switch command.Command(strings.ToUpper(parts[0])) {
	case command.PING:
		return "+PONG\r\n"
	case command.ECHO:
		if len(parts) < 2 {
			return errWrongArgs
		}
		return resp.BulkString(parts[1])
	case command.SET:
		if len(parts) < 3 {
			return errWrongArgs
		}
		h.store.Set(parts[1], parts[2])
		return okResponse
	case command.GET:
		if len(parts) < 2 {
			return errWrongArgs
		}
		val, ok := h.store.Get(parts[1])
		if !ok {
			return nullBulk
		}
		return resp.BulkString(val)
	default:
		return errResponse
	}
}
