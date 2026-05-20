package handler

import "fmt"

func (h *Handler) handlePsync(_ []string) string {
	return fmt.Sprintf("+FULLRESYNC %s %d\r\n", masterReplID, masterReplOffset)
}
