package handler

import "github.com/codecrafters-io/redis-starter-go/internal/resp"

func (h *Handler) handleIncr(parts []string) string {
	if len(parts) < 2 {
		return errWrongArgs
	}
	val, err := h.store.Incr(parts[1])
	if err != nil {
		return resp.Error(err.Error())
	}
	return resp.Integer(int(val))
}

func (h *Handler) handleDecr(parts []string) string {
	if len(parts) < 2 {
		return errWrongArgs
	}
	val, err := h.store.Decr(parts[1])
	if err != nil {
		return resp.Error(err.Error())
	}
	return resp.Integer(int(val))
}
