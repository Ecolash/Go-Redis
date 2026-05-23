package handler

import (
	"strings"

	"github.com/codecrafters-io/redis-starter-go/internal/errs"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func (h *Handler) handleACL(parts []string) string {
	if len(parts) < 2 {
		return errs.WrongArgs
	}
	sub := strings.ToUpper(parts[1])
	switch sub {
	case "WHOAMI":
		return resp.BulkString("default")
	case "GETUSER":
		if len(parts) < 3 {
			return errs.WrongArgs
		}
		if strings.ToLower(parts[2]) != "default" {
			return nullBulk
		}
		// Return: ["flags", []]
		return resp.RawArray([]string{
			resp.BulkString("flags"),
			"*0\r\n",
		})
	default:
		return resp.Error("ERR unknown subcommand '" + parts[1] + "' for 'acl' command")
	}
}
