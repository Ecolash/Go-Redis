package handler_test

import (
	"testing"

	"github.com/codecrafters-io/redis-starter-go/internal/handler"
	"github.com/codecrafters-io/redis-starter-go/internal/store"
)

func newHandler() *handler.Handler {
	return handler.New(store.New())
}

func TestHandlePingReturnsSimplePong(t *testing.T) {
	h := newHandler()
	if got := h.Handle([]byte("*1\r\n$4\r\nPING\r\n")); got != "+PONG\r\n" {
		t.Errorf("expected +PONG\\r\\n, got %q", got)
	}
}

func TestHandlePingIsCaseInsensitive(t *testing.T) {
	h := newHandler()
	if got := h.Handle([]byte("*1\r\n$4\r\nping\r\n")); got != "+PONG\r\n" {
		t.Errorf("expected +PONG\\r\\n, got %q", got)
	}
}

func TestHandleEchoReturnsBulkStringArgument(t *testing.T) {
	h := newHandler()
	if got := h.Handle([]byte("*2\r\n$4\r\nECHO\r\n$3\r\nhey\r\n")); got != "$3\r\nhey\r\n" {
		t.Errorf("expected $3\\r\\nhey\\r\\n, got %q", got)
	}
}

func TestHandleEchoIsCaseInsensitive(t *testing.T) {
	h := newHandler()
	if got := h.Handle([]byte("*2\r\n$4\r\necho\r\n$5\r\nhello\r\n")); got != "$5\r\nhello\r\n" {
		t.Errorf("expected $5\\r\\nhello\\r\\n, got %q", got)
	}
}

func TestHandleSetReturnsOK(t *testing.T) {
	h := newHandler()
	if got := h.Handle([]byte("*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n")); got != "+OK\r\n" {
		t.Errorf("expected +OK\\r\\n, got %q", got)
	}
}

func TestHandleGetReturnsValueAfterSet(t *testing.T) {
	h := newHandler()
	h.Handle([]byte("*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n"))
	if got := h.Handle([]byte("*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n")); got != "$3\r\nbar\r\n" {
		t.Errorf("expected $3\\r\\nbar\\r\\n, got %q", got)
	}
}

func TestHandleGetReturnsNullBulkStringForMissingKey(t *testing.T) {
	h := newHandler()
	if got := h.Handle([]byte("*2\r\n$3\r\nGET\r\n$6\r\nnobody\r\n")); got != "$-1\r\n" {
		t.Errorf("expected $-1\\r\\n, got %q", got)
	}
}
