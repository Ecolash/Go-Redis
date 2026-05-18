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

func TestStreamEntries(t *testing.T) {
	tests := []struct {
		name    string
		entries []resp.Entry
		want    string
	}{
		{
			name:    "empty slice returns empty outer array",
			entries: []resp.Entry{},
			want:    "*0\r\n",
		},
		{
			name: "single entry with two fields",
			entries: []resp.Entry{
				{ID: "0-1", Fields: []string{"k", "v"}},
			},
			want: "*1\r\n*2\r\n$3\r\n0-1\r\n*2\r\n$1\r\nk\r\n$1\r\nv\r\n",
		},
		{
			name: "two entries matches spec example",
			entries: []resp.Entry{
				{ID: "1526985054069-0", Fields: []string{"temperature", "36", "humidity", "95"}},
				{ID: "1526985054079-0", Fields: []string{"temperature", "37", "humidity", "94"}},
			},
			want: "*2\r\n" +
				"*2\r\n$15\r\n1526985054069-0\r\n" +
				"*4\r\n$11\r\ntemperature\r\n$2\r\n36\r\n$8\r\nhumidity\r\n$2\r\n95\r\n" +
				"*2\r\n$15\r\n1526985054079-0\r\n" +
				"*4\r\n$11\r\ntemperature\r\n$2\r\n37\r\n$8\r\nhumidity\r\n$2\r\n94\r\n",
		},
		{
			name: "entry with no fields encodes empty inner array",
			entries: []resp.Entry{
				{ID: "1-0", Fields: []string{}},
			},
			want: "*1\r\n*2\r\n$3\r\n1-0\r\n*0\r\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resp.StreamEntries(tt.entries); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
