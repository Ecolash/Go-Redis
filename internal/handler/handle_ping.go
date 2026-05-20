package handler

func (h *Handler) handlePing(_ []string) string {
	return "+PONG\r\n"
}
