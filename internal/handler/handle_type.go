package handler

func (h *Handler) handleType(parts []string) string {
	if len(parts) < 2 {
		return errWrongArgs
	}
	return "+" + h.store.Type(parts[1]) + "\r\n"
}
