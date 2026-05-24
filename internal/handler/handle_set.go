package handler

import (
	"strconv"
	"strings"
	"time"

	"github.com/codecrafters-io/redis-starter-go/internal/errs"
)

func (h *Handler) handleSet(parts []string) string {
	if len(parts) < 3 {
		return errs.WrongArgs
	}
	h.store.Set(parts[1], parts[2], parseTTL(parts[3:]))
	return okResponse
}

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
