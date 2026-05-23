package handler

import (
	"math"
	"strconv"

	"github.com/codecrafters-io/redis-starter-go/internal/errs"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
	"github.com/codecrafters-io/redis-starter-go/internal/store"
)

func (h *Handler) handleZAdd(parts []string) string {
	// ZADD key score member [score member ...]
	if len(parts) < 4 || (len(parts)-2)%2 != 0 {
		return errs.WrongArgs
	}
	key := parts[1]
	members := make([]store.ZSetMember, 0, (len(parts)-2)/2)
	for i := 2; i < len(parts); i += 2 {
		score, err := strconv.ParseFloat(parts[i], 64)
		if err != nil || math.IsNaN(score) {
			return resp.Error("ERR value is not a valid float")
		}
		members = append(members, store.ZSetMember{Score: score, Member: parts[i+1]})
	}
	n := h.store.ZAdd(key, members)
	return resp.Integer(n)
}

func (h *Handler) handleZRange(parts []string) string {
	// ZRANGE key start stop
	if len(parts) != 4 {
		return errs.WrongArgs
	}
	start, err1 := strconv.Atoi(parts[2])
	stop, err2 := strconv.Atoi(parts[3])
	if err1 != nil || err2 != nil {
		return errs.WrongArgs
	}
	members := h.store.ZRange(parts[1], start, stop)
	strs := make([]string, len(members))
	for i, m := range members {
		strs[i] = m.Member
	}
	return resp.Array(strs)
}

func (h *Handler) handleZRank(parts []string) string {
	// ZRANK key member
	if len(parts) != 3 {
		return errs.WrongArgs
	}
	rank, ok := h.store.ZRank(parts[1], parts[2])
	if !ok {
		return nullBulk
	}
	return resp.Integer(rank)
}

func (h *Handler) handleZScore(parts []string) string {
	// ZSCORE key member
	if len(parts) != 3 {
		return errs.WrongArgs
	}
	score, ok := h.store.ZScore(parts[1], parts[2])
	if !ok {
		return nullBulk
	}
	return resp.BulkString(strconv.FormatFloat(score, 'f', -1, 64))
}
