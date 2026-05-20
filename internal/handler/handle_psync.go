package handler

import (
	"fmt"

	"github.com/codecrafters-io/redis-starter-go/internal/rdb"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func (h *Handler) handlePsync(_ []string) string {
	header := fmt.Sprintf("+FULLRESYNC %s %d\r\n", masterReplID, masterReplOffset)
	h.becameReplica = true
	return header + string(resp.File(rdb.Empty()))
}
