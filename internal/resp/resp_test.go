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

func TestInteger(t *testing.T) {
	if got := resp.Integer(1); got != ":1\r\n" {
		t.Errorf("expected :1\\r\\n, got %q", got)
	}
}

func TestIntegerZero(t *testing.T) {
	if got := resp.Integer(0); got != ":0\r\n" {
		t.Errorf("expected :0\\r\\n, got %q", got)
	}
}

func TestArray(t *testing.T) {
	got := resp.Array([]string{"foo", "bar"})
	expected := "*2\r\n$3\r\nfoo\r\n$3\r\nbar\r\n"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestError(t *testing.T) {
	got := resp.Error("ERR something went wrong")
	if got != "-ERR something went wrong\r\n" {
		t.Errorf("got %q, want \"-ERR something went wrong\\r\\n\"", got)
	}
}

func TestArrayEmpty(t *testing.T) {
	got := resp.Array([]string{})
	expected := "*0\r\n"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}
