package handler

import (
	"crypto/sha256"
	"fmt"
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
		flags := h.defaultUser.flags()
		var flagsEncoded string
		if len(flags) == 0 {
			flagsEncoded = "*0\r\n"
		} else {
			flagsEncoded = resp.Array(flags)
		}
		var pwEncoded string
		if len(h.defaultUser.passwords) == 0 {
			pwEncoded = "*0\r\n"
		} else {
			pwEncoded = resp.Array(h.defaultUser.passwords)
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
				password := rule[1:]
				hash := fmt.Sprintf("%x", sha256.Sum256([]byte(password)))
				h.defaultUser.nopass = false
				h.defaultUser.passwords = append(h.defaultUser.passwords, hash)
			}
		}
		return okResponse
	default:
		return resp.Error("ERR unknown subcommand '" + parts[1] + "' for 'acl' command")
	}
}
