package handler_test

import (
	"testing"

	"github.com/codecrafters-io/redis-starter-go/internal/handler"
)

func TestHandleReturnsPoingForAnyCommand(t *testing.T) {
	response := handler.Handle([]byte("PING"))
	if response != "+PONG\r\n" {
		t.Errorf("expected +PONG\\r\\n, got %q", response)
	}
}

func TestHandleReturnsPoingForEmptyInput(t *testing.T) {
	response := handler.Handle([]byte{})
	if response != "+PONG\r\n" {
		t.Errorf("expected +PONG\\r\\n, got %q", response)
	}
}
