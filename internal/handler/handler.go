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
	nullArray    = "*-1\r\n"
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
		command.TYPE:  h.handleType,
		command.XADD:  h.handleXAdd,
		command.LPOP:  h.handleLPop,
		command.RPOP:  h.handleRPop,
		command.BLPOP: h.handleBLPop,
		command.LLEN:  h.handleLLen,
		command.LPUSH: h.handleLPush,
		command.RPUSH: h.handleRPush,
		command.LRANGE: h.handleLRange,
	}
	return h
}

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

func (h *Handler) handleXAdd(parts []string) string {
	// XADD key id field value [field value ...]
	if len(parts) < 5 || (len(parts)-3)%2 != 0 {
		return errWrongArgs
	}
	id, err := h.store.XAdd(parts[1], parts[2], parts[3:])
	if err != nil {
		return resp.Error(err.Error())
	}
	return resp.BulkString(id)
}

func (h *Handler) handleType(parts []string) string {
	if len(parts) < 2 {
		return errWrongArgs
	}
	return "+" + h.store.Type(parts[1]) + "\r\n"
}

func (h *Handler) handleLPop(parts []string) string {
	if len(parts) < 2 {
		return errWrongArgs
	}
	if len(parts) == 2 {
		val, ok := h.store.LPop(parts[1])
		if !ok {
			return nullBulk
		}
		return resp.BulkString(val)
	}
	count, err := strconv.Atoi(parts[2])
	if err != nil || count < 0 {
		return errWrongArgs
	}
	vals, ok := h.store.LPopCount(parts[1], count)
	if !ok {
		return nullArray
	}
	return resp.Array(vals)
}

func (h *Handler) handleRPop(parts []string) string {
	if len(parts) < 2 {
		return errWrongArgs
	}
	if len(parts) == 2 {
		val, ok := h.store.RPop(parts[1])
		if !ok {
			return nullBulk
		}
		return resp.BulkString(val)
	}
	count, err := strconv.Atoi(parts[2])
	if err != nil || count < 0 {
		return errWrongArgs
	}
	vals, ok := h.store.RPopCount(parts[1], count)
	if !ok {
		return nullArray
	}
	return resp.Array(vals)
}

func (h *Handler) handleBLPop(parts []string) string {
	// parts: [BLPOP, key1, ..., keyN, timeout]
	if len(parts) < 3 {
		return errWrongArgs
	}
	keys := parts[1 : len(parts)-1]
	timeoutSecs, err := strconv.ParseFloat(parts[len(parts)-1], 64)
	if err != nil || timeoutSecs < 0 {
		return errWrongArgs
	}

	channel, cancel := h.store.BLPopWait(keys)
	defer cancel()

	if timeoutSecs == 0 {
		result := <-channel
		return resp.Array([]string{result.Key, result.Val})
	}

	timer := time.NewTimer(time.Duration(float64(time.Second) * timeoutSecs))
	defer timer.Stop()

	select {
	case result := <-channel:
		return resp.Array([]string{result.Key, result.Val})
	case <-timer.C:
		return nullArray
	}
}

func (h *Handler) handleLLen(parts []string) string {
	if len(parts) < 2 {
		return errWrongArgs
	}
	return resp.Integer(h.store.LLen(parts[1]))
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
