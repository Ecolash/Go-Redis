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
		flags := h.defaultUser.Flags()
		var flagsEncoded string
		if len(flags) == 0 {
			flagsEncoded = "*0\r\n"
		} else {
			flagsEncoded = resp.Array(flags)
		}
		pws := h.defaultUser.Passwords()
		var pwEncoded string
		if len(pws) == 0 {
			pwEncoded = "*0\r\n"
		} else {
			pwEncoded = resp.Array(pws)
		}
		return resp.RawArray([]string{
			resp.BulkString("flags"),
			flagsEncoded,
			resp.BulkString("passwords"),
			pwEncoded,
		})
	case "SETUSER":
		if len(parts) < 4 {
			return errs.WrongArgs
		}
		if strings.ToLower(parts[2]) != "default" {
			return resp.Error("ERR unknown user")
		}
		for _, rule := range parts[3:] {
			if strings.HasPrefix(rule, ">") {
				h.defaultUser.AddPassword(rule[1:])
			}
		}
		return okResponse
	default:
		return resp.Error("ERR unknown subcommand '" + parts[1] + "' for 'acl' command")
	}
}

func (h *Handler) handleAuth(parts []string) string {
	if len(parts) < 3 {
		return errs.WrongArgs
	}
	username := strings.ToLower(parts[1])
	password := parts[2]
	if username != "default" {
		return resp.Error("WRONGPASS invalid username-password pair or user is disabled.")
	}
	if !h.defaultUser.Authenticate(password) {
		return resp.Error("WRONGPASS invalid username-password pair or user is disabled.")
	}
	h.authenticated = true
	return okResponse
}
