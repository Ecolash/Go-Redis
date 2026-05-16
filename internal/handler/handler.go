package handler

import (
	"strconv"
	"strings"
	"time"

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
		ttl := parseTTL(parts[3:])
		h.store.Set(parts[1], parts[2], ttl)
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

// parseTTL extracts the TTL duration from optional SET arguments (PX <ms> or EX <s>).
// Returns 0 if no TTL option is present or the value is invalid.
func parseTTL(opts []string) time.Duration {
	if len(opts) < 2 {
		return 0
	}
	n, err := strconv.ParseInt(opts[1], 10, 64)
	if err != nil || n <= 0 {
		return 0
	}
	switch strings.ToUpper(opts[0]) {
	case "PX":
		return time.Duration(n) * time.Millisecond
	case "EX":
		return time.Duration(n) * time.Second
	}
	return 0
}
