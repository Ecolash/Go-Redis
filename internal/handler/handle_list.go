package handler

import (
	"strconv"
	"time"

	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func (h *Handler) handleLPop(parts []string) string {
	if len(parts) < 2 {
		return errWrongArgs
	}
	if len(parts) == 2 {
		val, ok := h.store.LPop(parts[1])
		if !ok {
			return nullBulk
		}
		return resp.BulkString(val)
	}
	count, err := strconv.Atoi(parts[2])
	if err != nil || count < 0 {
		return errWrongArgs
	}
	vals, ok := h.store.LPopCount(parts[1], count)
	if !ok {
		return nullArray
	}
	return resp.Array(vals)
}

func (h *Handler) handleRPop(parts []string) string {
	if len(parts) < 2 {
		return errWrongArgs
	}
	if len(parts) == 2 {
		val, ok := h.store.RPop(parts[1])
		if !ok {
			return nullBulk
		}
		return resp.BulkString(val)
	}
	count, err := strconv.Atoi(parts[2])
	if err != nil || count < 0 {
		return errWrongArgs
	}
	vals, ok := h.store.RPopCount(parts[1], count)
	if !ok {
		return nullArray
	}
	return resp.Array(vals)
}

func (h *Handler) handleBLPop(parts []string) string {
	// parts: [BLPOP, key1, ..., keyN, timeout]
	if len(parts) < 3 {
		return errWrongArgs
	}
	keys := parts[1 : len(parts)-1]
	timeoutSecs, err := strconv.ParseFloat(parts[len(parts)-1], 64)
	if err != nil || timeoutSecs < 0 {
		return errWrongArgs
	}

	channel, cancel := h.store.BLPopWait(keys)
	defer cancel()

	if timeoutSecs == 0 {
		result := <-channel
		return resp.Array([]string{result.Key, result.Val})
	}

	timer := time.NewTimer(time.Duration(float64(time.Second) * timeoutSecs))
	defer timer.Stop()

	select {
	case result := <-channel:
		return resp.Array([]string{result.Key, result.Val})
	case <-timer.C:
		return nullArray
	}
}

func (h *Handler) handleLLen(parts []string) string {
	if len(parts) < 2 {
		return errWrongArgs
	}
	return resp.Integer(h.store.LLen(parts[1]))
}

func (h *Handler) handleLPush(parts []string) string {
	if len(parts) < 3 {
		return errWrongArgs
	}
	n := h.store.LPush(parts[1], parts[2:]...)
	return resp.Integer(n)
}

func (h *Handler) handleRPush(parts []string) string {
	if len(parts) < 3 {
		return errWrongArgs
	}
	n := h.store.RPush(parts[1], parts[2:]...)
	return resp.Integer(n)
}

func (h *Handler) handleLRange(parts []string) string {
	if len(parts) < 4 {
		return errWrongArgs
	}
	start, err1 := strconv.Atoi(parts[2])
	stop, err2 := strconv.Atoi(parts[3])
	if err1 != nil || err2 != nil {
		return errWrongArgs
	}
	vals, _ := h.store.LRange(parts[1], start, stop)
	return resp.Array(vals)
}
