package handler

import (
	"fmt"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

const (
	masterReplID     = "8371b4fb1155b71f4a04d3e1bc3e18c4a990aeeb"
	masterReplOffset = 0
)

func (h *Handler) handleInfo(parts []string) string {
	section := ""
	if len(parts) >= 2 {
		section = strings.ToLower(parts[1])
	}

	var sb strings.Builder
	if section == "" || section == "replication" {
		sb.WriteString("# Replication\r\n")
		fmt.Fprintf(&sb, "role:%s\r\n", h.role)
		fmt.Fprintf(&sb, "master_replid:%s\r\n", masterReplID)
		fmt.Fprintf(&sb, "master_repl_offset:%d\r\n", masterReplOffset)
	}
	return resp.BulkString(sb.String())
}
