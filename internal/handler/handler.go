package handler

const pong = "+PONG\r\n"

// Handle processes an incoming Redis command and returns a RESP-encoded response.
// For this stage, all commands return PONG.
func Handle(_ []byte) string {
	return pong
}
