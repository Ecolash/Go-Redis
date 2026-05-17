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

type commandFunc func(parts []string) string

type Handler struct {
	store    *store.Store
	commands map[command.Command]commandFunc
}

func New(s *store.Store) *Handler {
	h := &Handler{store: s}
	h.commands = map[command.Command]commandFunc{
		command.PING:  h.handlePing,
		command.ECHO:  h.handleEcho,
		command.SET:   h.handleSet,
		command.GET:   h.handleGet,
		command.LPUSH: h.handleLPush,
		command.RPUSH: h.handleRPush,
		command.LRANGE: h.handleLRange,
	}
	return h
}

// Handle parses a RESP-encoded command and dispatches to the appropriate handler.
func (h *Handler) Handle(data []byte) string {
	parts, err := resp.ParseArray(data)
	if err != nil || len(parts) == 0 {
		return errResponse
	}
	fn, ok := h.commands[command.Command(strings.ToUpper(parts[0]))]
	if !ok {
		return errResponse
	}
	return fn(parts)
}

func (h *Handler) handlePing(_ []string) string {
	return "+PONG\r\n"
}

func (h *Handler) handleEcho(parts []string) string {
	if len(parts) < 2 {
		return errWrongArgs
	}
	return resp.BulkString(parts[1])
}

func (h *Handler) handleSet(parts []string) string {
	if len(parts) < 3 {
		return errWrongArgs
	}
	h.store.Set(parts[1], parts[2], parseTTL(parts[3:]))
	return okResponse
}

func (h *Handler) handleGet(parts []string) string {
	if len(parts) < 2 {
		return errWrongArgs
	}
	val, ok := h.store.Get(parts[1])
	if !ok {
		return nullBulk
	}
	return resp.BulkString(val)
}

func (h *Handler) handleLPush(parts []string) string {
	if len(parts) < 3 {
		return errWrongArgs
	}
	n := h.store.LPush(parts[1], parts[2:]...)
	return resp.Integer(n)
}

func (h *Handler) handleRPush(parts []string) string {
	if len(parts) < 3 {
		return errWrongArgs
	}
	n := h.store.RPush(parts[1], parts[2:]...)
	return resp.Integer(n)
}

func (h *Handler) handleLRange(parts []string) string {
	if len(parts) < 4 {
		return errWrongArgs
	}
	start, err1 := strconv.Atoi(parts[2])
	stop, err2 := strconv.Atoi(parts[3])
	if err1 != nil || err2 != nil {
		return errWrongArgs
	}
	vals, _ := h.store.LRange(parts[1], start, stop)
	return resp.Array(vals)
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
