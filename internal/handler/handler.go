package handler

import (
	"strings"
	"time"

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

/*

Handler is responsible for parsing incoming RESP commands, executing them against the store, and returning RESP responses. 
It also tracks transaction state for MULTI/EXEC and watches for WATCHed keys to implement optimistic locking.
It can be configured with a callback to propagate write commands to connected replicas.

- store: the underlying data store for executing commands
- role: the server role (master | slave) for Leader-Follower Replication
- commands: mapping of command names to respective handler functions
- txCommands: mapping of transaction-related commands (MULTI/EXEC/WATCH) to their handlers
- inMulti: whether the connection is currently in a MULTI block
- queue: queued commands during a MULTI block
- watching: keys being WATCHed for optimistic locking
- onPropagate: callback func for propagating write commands to replicas
- replica: whether this connection has become a replica after PSYNC
- replyToMaster: whether the next response should be forwarded to the master (e.g. for REPLCONF GETACK)
- trackOffset: whether to track the byte length of processed commands for REPLCONF GETACK
- offset: accumulated byte length of processed commands for REPLCONF GETACK

- Handle() : main entry point for processing incoming RESP commands.
- dispatch() : looks up the command handler and executes it, also handles propagation for write commands.
- BecameReplica() : checks if the connection has become a replica (after PSYNC) and resets the replica flag.
*/

type Handler struct {
	store *store.Store
	role  string
	inMulti bool
	queue   [][]string
	watching map[string]uint64
	onPropagate func(parts []string)
	replicaCount func() int
	replicaWaiter func(numReplicas int, timeout time.Duration) int
	replica bool
	replyToMaster bool
	trackOffset bool
	offset int
	config map[string]string

	commands map[command.Command]commandFunc
	txCommands map[command.Command]commandFunc
}

type Option func(*Handler)

func WithPropagate(fn func(parts []string)) Option {
	return func(h *Handler) { h.onPropagate = fn }
}

// WithReplicaCount lets the handler report the number of currently-connected
// replicas (used by WAIT). The callback is consulted at request time so the
// count is always live.
func WithReplicaCount(fn func() int) Option {
	return func(h *Handler) { h.replicaCount = fn }
}

// WithReplicaWaiter wires the master's WAIT implementation: a blocking call
// that returns the number of replicas which have acknowledged all previously
// propagated writes within the given timeout.
func WithReplicaWaiter(fn func(numReplicas int, timeout time.Duration) int) Option {
	return func(h *Handler) { h.replicaWaiter = fn }
}

// WithOffsetTracking makes the handler accumulate the byte length of every
// command passed to Handle. Used by replicas so REPLCONF GETACK can report the
// number of bytes processed before the current request.
func WithOffsetTracking() Option {
	return func(h *Handler) { h.trackOffset = true }
}

func WithConfig(key, value string) Option {
	return func(h *Handler) {
		if h.config == nil {
			h.config = make(map[string]string)
		}
		h.config[key] = value
	}
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
		command.WAIT:     h.handleWait,
		command.CONFIG:   h.handleConfig,
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
	if h.trackOffset {
		// Increment AFTER dispatch so the current command's bytes aren't
		// reflected in its own reply (matters for REPLCONF GETACK).
		defer func() { h.offset += len(data) }()
	}
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

func (h *Handler) BecameReplica() bool {
	b := h.replica
	h.replica = false
	return b
}

// ShouldReplyToMaster reports whether the previous Handle call produced a reply
// that the replica must forward back over its master connection (e.g. the ACK
// for REPLCONF GETACK). The flag is consumed on read.
func (h *Handler) ShouldReplyToMaster() bool {
	b := h.replyToMaster
	h.replyToMaster = false
	return b
}

// writeCommands are the commands that mutate the dataset & must be forwarded to connected replicas.
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
