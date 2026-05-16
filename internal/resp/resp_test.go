package resp_test

import (
	"testing"

	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func TestParseArrayWithSingleElement(t *testing.T) {
	input := "*1\r\n$4\r\nPING\r\n"
	got, err := resp.ParseArray([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0] != "PING" {
		t.Errorf("expected [PING], got %v", got)
	}
}

func TestParseArrayWithTwoElements(t *testing.T) {
	input := "*2\r\n$4\r\nECHO\r\n$3\r\nhey\r\n"
	got, err := resp.ParseArray([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != "ECHO" || got[1] != "hey" {
		t.Errorf("expected [ECHO hey], got %v", got)
	}
}

func TestBulkString(t *testing.T) {
	got := resp.BulkString("hey")
	if got != "$3\r\nhey\r\n" {
		t.Errorf("expected $3\\r\\nhey\\r\\n, got %q", got)
	}
}

func TestBulkStringEmpty(t *testing.T) {
	got := resp.BulkString("")
	if got != "$0\r\n\r\n" {
		t.Errorf("expected $0\\r\\n\\r\\n, got %q", got)
	}
}
