package handler

import (
	"strings"

	"github.com/codecrafters-io/redis-starter-go/internal/errs"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func (h *Handler) handleConfig(parts []string) string {
	if len(parts) < 3 {
		return errs.WrongArgs
	}
	subCmd := strings.ToUpper(parts[1])
	if subCmd != "GET" {
		return resp.Error("ERR unknown subcommand '" + parts[1] + "' for 'config' command")
	}
	key := strings.ToLower(parts[2])
	val, ok := h.config[key]
	if !ok {
		return resp.Array([]string{})
	}
	return resp.Array([]string{key, val})
}
