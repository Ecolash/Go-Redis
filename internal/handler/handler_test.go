package handler_test

import (
	"testing"

	"github.com/codecrafters-io/redis-starter-go/internal/handler"
)

func TestHandlePingReturnsSimplePong(t *testing.T) {
	response := handler.Handle([]byte("*1\r\n$4\r\nPING\r\n"))
	if response != "+PONG\r\n" {
		t.Errorf("expected +PONG\\r\\n, got %q", response)
	}
}

func TestHandlePingIsCaseInsensitive(t *testing.T) {
	response := handler.Handle([]byte("*1\r\n$4\r\nping\r\n"))
	if response != "+PONG\r\n" {
		t.Errorf("expected +PONG\\r\\n, got %q", response)
	}
}

func TestHandleEchoReturnsBulkStringArgument(t *testing.T) {
	response := handler.Handle([]byte("*2\r\n$4\r\nECHO\r\n$3\r\nhey\r\n"))
	if response != "$3\r\nhey\r\n" {
		t.Errorf("expected $3\\r\\nhey\\r\\n, got %q", response)
	}
}

func TestHandleEchoIsCaseInsensitive(t *testing.T) {
	response := handler.Handle([]byte("*2\r\n$4\r\necho\r\n$5\r\nhello\r\n"))
	if response != "$5\r\nhello\r\n" {
		t.Errorf("expected $5\\r\\nhello\\r\\n, got %q", response)
	}
}
