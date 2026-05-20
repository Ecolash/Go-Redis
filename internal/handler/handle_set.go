package handler

import (
	"strconv"
	"strings"
	"time"
)

func (h *Handler) handleSet(parts []string) string {
	if len(parts) < 3 {
		return errWrongArgs
	}
	h.store.Set(parts[1], parts[2], parseTTL(parts[3:]))
	return okResponse
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
