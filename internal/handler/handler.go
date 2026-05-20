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
	role  string

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

	// onPropagate is invoked with the command parts after a successful write
	// command runs, so the server can forward it to connected replicas.
	onPropagate func(parts []string)

	// becameReplica is set true when this connection just completed PSYNC.
	// The server checks it after writing the response and, if set, hands the
	// connection over to the replica registry.
	becameReplica bool
}

// Option configures a Handler at construction time.
type Option func(*Handler)

// WithPropagate registers a callback invoked for each successful write command.
func WithPropagate(fn func(parts []string)) Option {
	return func(h *Handler) { h.onPropagate = fn }
}

func New(s *store.Store, role string, opts ...Option) *Handler {
	h := &Handler{store: s, role: role}
	for _, opt := range opts {
		opt(h)
	}
	h.commands = map[command.Command]commandFunc{
		command.PING:   h.handlePing,
		command.ECHO:   h.handleEcho,
		command.INFO:   h.handleInfo,
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
		command.REPLCONF: h.handleReplConf,
		command.PSYNC:    h.handlePsync,
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
	cmd := command.Command(strings.ToUpper(parts[0]))
	fn, ok := h.commands[cmd]
	if !ok {
		return errs.UnknownCommand
	}
	result := fn(parts)
	if h.onPropagate != nil && writeCommands[cmd] && !strings.HasPrefix(result, "-") {
		h.onPropagate(parts)
	}
	return result
}

// BecameReplica returns true exactly once after a PSYNC has been handled on
// this connection, signalling the server to hand the conn over to the replica
// registry instead of continuing to read commands from it.
func (h *Handler) BecameReplica() bool {
	b := h.becameReplica
	h.becameReplica = false
	return b
}

// writeCommands are the commands that mutate the dataset and therefore must
// be forwarded to connected replicas.
var writeCommands = map[command.Command]bool{
	command.SET:   true,
	command.INCR:  true,
	command.DECR:  true,
	command.LPUSH: true,
	command.RPUSH: true,
	command.LPOP:  true,
	command.RPOP:  true,
	command.BLPOP: true,
	command.XADD:  true,
}
