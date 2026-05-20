package handler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/internal/rdb"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func (h *Handler) handleReplConf(parts []string) string {
	if len(parts) >= 2 && strings.EqualFold(parts[1], "GETACK") {
		h.replyToMaster = true
		offset := strconv.Itoa(h.offset)
		return resp.Array([]string{"REPLCONF", "ACK", offset})
	}
	return "+OK\r\n"
}

func (h *Handler) handlePsync(_ []string) string {
	header := fmt.Sprintf("+FULLRESYNC %s %d\r\n", masterReplID, masterReplOffset)
	h.replica = true
	return header + string(resp.File(rdb.Empty()))
}
