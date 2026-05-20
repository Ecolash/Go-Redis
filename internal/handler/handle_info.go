package handler

import (
	"strings"

	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func (h *Handler) handleInfo(parts []string) string {
	section := ""
	if len(parts) >= 2 {
		section = strings.ToLower(parts[1])
	}

	var sb strings.Builder
	if section == "" || section == "replication" {
		sb.WriteString("# Replication\r\n")
		sb.WriteString("role:")
		sb.WriteString(h.role)
		sb.WriteString("\r\n")
	}
	return resp.BulkString(sb.String())
}
