package handler

import (
	"strings"

	"github.com/codecrafters-io/redis-starter-go/internal/command"
	"github.com/codecrafters-io/redis-starter-go/internal/errs"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
	"github.com/codecrafters-io/redis-starter-go/internal/store"
)

const (
	okResponse = "+OK\r\n"
	queuedResp = "+QUEUED\r\n"
	nullBulk   = "$-1\r\n"
	nullArray  = "*-1\r\n"
)

type commandFunc func(parts []string) string

// Handler holds the per-connection command dispatch state. The underlying
// store is shared across connections; transaction state (inMulti, queue) is
// per-connection, so callers must create one Handler per client connection.
type Handler struct {
	store *store.Store

	// commands are normal data commands. While inMulti is true, they are
	// queued instead of executed.
	commands map[command.Command]commandFunc

	// txCommands are transaction-control commands. They always execute
	// immediately and bypass the queue, even inside MULTI.
	txCommands map[command.Command]commandFunc

	inMulti bool
	queue   [][]string

	// watching maps each WATCH'd key to the store version at the time of
	// WATCH. EXEC aborts if any key's current version differs.
	watching map[string]uint64
}

func New(s *store.Store) *Handler {
	h := &Handler{store: s}
	h.commands = map[command.Command]commandFunc{
		command.PING:   h.handlePing,
		command.ECHO:   h.handleEcho,
		command.SET:    h.handleSet,
		command.GET:    h.handleGet,
		command.TYPE:   h.handleType,
		command.INCR:   h.handleIncr,
		command.DECR:   h.handleDecr,
		command.XADD:   h.handleXAdd,
		command.XRANGE: h.handleXRange,
		command.XREAD:  h.handleXRead,
		command.LPOP:   h.handleLPop,
		command.RPOP:   h.handleRPop,
		command.BLPOP:  h.handleBLPop,
		command.LLEN:   h.handleLLen,
		command.LPUSH:  h.handleLPush,
		command.RPUSH:  h.handleRPush,
		command.LRANGE: h.handleLRange,
	}
	h.txCommands = map[command.Command]commandFunc{
		command.MULTI:   h.handleMulti,
		command.EXEC:    h.handleExec,
		command.DISCARD: h.handleDiscard,
		command.WATCH:   h.handleWatch,
		command.UNWATCH: h.handleUnwatch,
	}
	return h
}

func (h *Handler) Handle(data []byte) string {
	parts, err := resp.ParseArray(data)
	if err != nil || len(parts) == 0 {
		return errs.UnknownCommand
	}
	cmd := command.Command(strings.ToUpper(parts[0]))

	if fn, ok := h.txCommands[cmd]; ok {
		return fn(parts)
	}

	if h.inMulti {
		h.queue = append(h.queue, parts)
		return queuedResp
	}
	return h.dispatch(parts)
}

func (h *Handler) dispatch(parts []string) string {
	fn, ok := h.commands[command.Command(strings.ToUpper(parts[0]))]
	if !ok {
		return errs.UnknownCommand
	}
	return fn(parts)
}
