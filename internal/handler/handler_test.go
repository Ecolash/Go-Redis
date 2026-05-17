package handler_test

import (
	"testing"
	"time"

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

func TestHandleSetWithPXReturnsOK(t *testing.T) {
	h := newHandler()
	// SET foo bar PX 100
	got := h.Handle([]byte("*5\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n$2\r\nPX\r\n$3\r\n100\r\n"))
	if got != "+OK\r\n" {
		t.Errorf("expected +OK\\r\\n, got %q", got)
	}
}

func TestHandleGetReturnsValueBeforePXExpiry(t *testing.T) {
	h := newHandler()
	h.Handle([]byte("*5\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n$2\r\nPX\r\n$3\r\n200\r\n"))
	if got := h.Handle([]byte("*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n")); got != "$3\r\nbar\r\n" {
		t.Errorf("expected $3\\r\\nbar\\r\\n before expiry, got %q", got)
	}
}

func TestHandleGetReturnsNullAfterPXExpiry(t *testing.T) {
	h := newHandler()
	h.Handle([]byte("*5\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n$2\r\nPX\r\n$2\r\n20\r\n"))
	time.Sleep(30 * time.Millisecond)
	if got := h.Handle([]byte("*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n")); got != "$-1\r\n" {
		t.Errorf("expected $-1\\r\\n after expiry, got %q", got)
	}
}

func TestHandleSetWithEXReturnsOK(t *testing.T) {
	h := newHandler()
	// SET foo bar EX 10
	got := h.Handle([]byte("*5\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n$2\r\nEX\r\n$2\r\n10\r\n"))
	if got != "+OK\r\n" {
		t.Errorf("expected +OK\\r\\n, got %q", got)
	}
}

func TestHandleRPushCreatesListAndReturnsCount(t *testing.T) {
	h := newHandler()
	// RPUSH mylist foo
	got := h.Handle([]byte("*3\r\n$5\r\nRPUSH\r\n$6\r\nmylist\r\n$3\r\nfoo\r\n"))
	if got != ":1\r\n" {
		t.Errorf("expected :1\\r\\n, got %q", got)
	}
}

func TestHandleRPushAppendsAndReturnsUpdatedCount(t *testing.T) {
	h := newHandler()
	h.Handle([]byte("*3\r\n$5\r\nRPUSH\r\n$6\r\nmylist\r\n$3\r\nfoo\r\n"))
	got := h.Handle([]byte("*3\r\n$5\r\nRPUSH\r\n$6\r\nmylist\r\n$3\r\nbar\r\n"))
	if got != ":2\r\n" {
		t.Errorf("expected :2\\r\\n, got %q", got)
	}
}

func TestHandleLRangeReturnsElementsInRange(t *testing.T) {
	h := newHandler()
	h.Handle([]byte("*3\r\n$5\r\nRPUSH\r\n$6\r\nmylist\r\n$1\r\na\r\n"))
	h.Handle([]byte("*3\r\n$5\r\nRPUSH\r\n$6\r\nmylist\r\n$1\r\nb\r\n"))
	h.Handle([]byte("*3\r\n$5\r\nRPUSH\r\n$6\r\nmylist\r\n$1\r\nc\r\n"))
	got := h.Handle([]byte("*4\r\n$6\r\nLRANGE\r\n$6\r\nmylist\r\n$1\r\n0\r\n$1\r\n1\r\n"))
	if got != "*2\r\n$1\r\na\r\n$1\r\nb\r\n" {
		t.Errorf("expected *2\\r\\n$1\\r\\na\\r\\n$1\\r\\nb\\r\\n, got %q", got)
	}
}

func TestHandleLRangeWithNegativeIndices(t *testing.T) {
	h := newHandler()
	h.Handle([]byte("*3\r\n$5\r\nRPUSH\r\n$6\r\nmylist\r\n$1\r\na\r\n"))
	h.Handle([]byte("*3\r\n$5\r\nRPUSH\r\n$6\r\nmylist\r\n$1\r\nb\r\n"))
	h.Handle([]byte("*3\r\n$5\r\nRPUSH\r\n$6\r\nmylist\r\n$1\r\nc\r\n"))
	got := h.Handle([]byte("*4\r\n$6\r\nLRANGE\r\n$6\r\nmylist\r\n$2\r\n-2\r\n$2\r\n-1\r\n"))
	if got != "*2\r\n$1\r\nb\r\n$1\r\nc\r\n" {
		t.Errorf("expected *2\\r\\n$1\\r\\nb\\r\\n$1\\r\\nc\\r\\n, got %q", got)
	}
}

func TestHandleLRangeWithMissingKeyReturnsEmptyArray(t *testing.T) {
	h := newHandler()
	got := h.Handle([]byte("*4\r\n$6\r\nLRANGE\r\n$6\r\nunknown\r\n$1\r\n0\r\n$1\r\n-1\r\n"))
	if got != "*0\r\n" {
		t.Errorf("expected *0\\r\\n for missing key, got %q", got)
	}
}