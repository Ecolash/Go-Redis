package handler

import (
	"strings"

	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

const errResponse = "-ERR unknown command\r\n"

// Handle parses a RESP-encoded command and returns a RESP-encoded response.
func Handle(data []byte) string {
	parts, err := resp.ParseArray(data)
	if err != nil || len(parts) == 0 {
		return errResponse
	}

	switch strings.ToUpper(parts[0]) {
	case "PING":
		return "+PONG\r\n"
	case "ECHO":
		if len(parts) < 2 {
			return errResponse
		}
		return resp.BulkString(parts[1])
	default:
		return errResponse
	}
}
