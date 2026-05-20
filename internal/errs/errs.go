// Package errs centralizes Redis error values and RESP-encoded error
// responses used throughout the server. Domain errors (returned from the
// store as Go error values) are surfaced as exported variables; protocol-level
// error replies sent directly back to clients are exported as RESP-encoded
// string constants.
package errs

import "errors"

// Domain errors returned from the store package. Their Error() strings are
// the exact payloads Redis clients expect to see after the leading '-'.
var (
	ErrStreamIDZero    = errors.New("ERR The ID specified in XADD must be greater than 0-0")
	ErrStreamIDSmall   = errors.New("ERR The ID specified in XADD is equal or smaller than the target stream top item")
	ErrWrongType       = errors.New("WRONGTYPE Operation against a key holding the wrong kind of value")
	ErrNotInteger      = errors.New("ERR value is not an integer or out of range")
	ErrInvalidStreamID = errors.New("invalid stream ID")
)

// RESP-encoded error replies returned directly from handlers.
const (
	UnknownCommand = "-ERR unknown command\r\n"
	WrongArgs      = "-ERR wrong number of arguments\r\n"
	ExecNoMulti    = "-ERR EXEC without MULTI\r\n"
	DiscardNoMulti = "-ERR DISCARD without MULTI\r\n"
	WatchInMulti   = "-ERR WATCH inside MULTI is not allowed\r\n"
)
