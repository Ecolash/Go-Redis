package handler

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/codecrafters-io/redis-starter-go/internal/errs"
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

func (h *Handler) handleWait(parts []string) string {
	if len(parts) < 3 {
		return errs.WrongArgs
	}
	numReplicas, err := strconv.Atoi(parts[1])
	if err != nil {
		return resp.Error(errs.ErrNotInteger.Error())
	}
	timeoutMs, err := strconv.Atoi(parts[2])
	if err != nil {
		return resp.Error(errs.ErrNotInteger.Error())
	}
	if h.role == "master" {
		if h.replicaWaiter != nil {
			return resp.Integer(h.replicaWaiter(numReplicas, time.Duration(timeoutMs)*time.Millisecond))
		}
		if h.replicaCount != nil {
			return resp.Integer(h.replicaCount())
		}
	}
	return resp.Integer(0)
}