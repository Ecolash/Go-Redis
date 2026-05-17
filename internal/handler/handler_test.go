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

func runCommands(h *handler.Handler, cmds []string) {
	for _, cmd := range cmds {
		h.Handle([]byte(cmd))
	}
}

func TestHandlePing(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple pong", "*1\r\n$4\r\nPING\r\n", "+PONG\r\n"},
		{"case insensitive", "*1\r\n$4\r\nping\r\n", "+PONG\r\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHandler()
			if got := h.Handle([]byte(tt.input)); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHandleEcho(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"returns bulk string argument", "*2\r\n$4\r\nECHO\r\n$3\r\nhey\r\n", "$3\r\nhey\r\n"},
		{"case insensitive", "*2\r\n$4\r\necho\r\n$5\r\nhello\r\n", "$5\r\nhello\r\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHandler()
			if got := h.Handle([]byte(tt.input)); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHandleSet(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"returns OK", "*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n", "+OK\r\n"},
		{"with PX returns OK", "*5\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n$2\r\nPX\r\n$3\r\n100\r\n", "+OK\r\n"},
		{"with EX returns OK", "*5\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n$2\r\nEX\r\n$2\r\n10\r\n", "+OK\r\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHandler()
			if got := h.Handle([]byte(tt.input)); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHandleGet(t *testing.T) {
	tests := []struct {
		name  string
		setup []string
		input string
		want  string
	}{
		{
			name:  "returns value after set",
			setup: []string{"*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n"},
			input: "*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n",
			want:  "$3\r\nbar\r\n",
		},
		{
			name:  "returns null bulk string for missing key",
			input: "*2\r\n$3\r\nGET\r\n$6\r\nnobody\r\n",
			want:  "$-1\r\n",
		},
		{
			name:  "returns value before PX expiry",
			setup: []string{"*5\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n$2\r\nPX\r\n$3\r\n200\r\n"},
			input: "*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n",
			want:  "$3\r\nbar\r\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHandler()
			runCommands(h, tt.setup)
			if got := h.Handle([]byte(tt.input)); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHandleGetReturnsNullAfterPXExpiry(t *testing.T) {
	h := newHandler()
	h.Handle([]byte("*5\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n$2\r\nPX\r\n$2\r\n20\r\n"))
	time.Sleep(30 * time.Millisecond)
	if got := h.Handle([]byte("*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n")); got != "$-1\r\n" {
		t.Errorf("got %q, want $-1\\r\\n", got)
	}
}

func TestHandleRPush(t *testing.T) {
	tests := []struct {
		name  string
		setup []string
		input string
		want  string
	}{
		{
			name:  "creates list and returns count",
			input: "*3\r\n$5\r\nRPUSH\r\n$6\r\nmylist\r\n$3\r\nfoo\r\n",
			want:  ":1\r\n",
		},
		{
			name:  "appends and returns updated count",
			setup: []string{"*3\r\n$5\r\nRPUSH\r\n$6\r\nmylist\r\n$3\r\nfoo\r\n"},
			input: "*3\r\n$5\r\nRPUSH\r\n$6\r\nmylist\r\n$3\r\nbar\r\n",
			want:  ":2\r\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHandler()
			runCommands(h, tt.setup)
			if got := h.Handle([]byte(tt.input)); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHandleLPush(t *testing.T) {
	tests := []struct {
		name  string
		setup []string
		input string
		want  string
	}{
		{
			name:  "creates list and returns count",
			input: "*3\r\n$5\r\nLPUSH\r\n$6\r\nmylist\r\n$3\r\nfoo\r\n",
			want:  ":1\r\n",
		},
		{
			name:  "prepends and returns updated count",
			setup: []string{"*3\r\n$5\r\nLPUSH\r\n$6\r\nmylist\r\n$3\r\nfoo\r\n"},
			input: "*3\r\n$5\r\nLPUSH\r\n$6\r\nmylist\r\n$3\r\nbar\r\n",
			want:  ":2\r\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHandler()
			runCommands(h, tt.setup)
			if got := h.Handle([]byte(tt.input)); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHandleLLen(t *testing.T) {
	tests := []struct {
		name  string
		setup []string
		input string
		want  string
	}{
		{
			name:  "returns 0 for missing key",
			input: "*2\r\n$4\r\nLLEN\r\n$6\r\nnobody\r\n",
			want:  ":0\r\n",
		},
		{
			name:  "returns length after rpush",
			setup: []string{"*4\r\n$5\r\nRPUSH\r\n$6\r\nmylist\r\n$1\r\na\r\n$1\r\nb\r\n"},
			input: "*2\r\n$4\r\nLLEN\r\n$6\r\nmylist\r\n",
			want:  ":2\r\n",
		},
		{
			name:  "returns length after lpush",
			setup: []string{"*5\r\n$5\r\nLPUSH\r\n$6\r\nmylist\r\n$1\r\na\r\n$1\r\nb\r\n$1\r\nc\r\n"},
			input: "*2\r\n$4\r\nLLEN\r\n$6\r\nmylist\r\n",
			want:  ":3\r\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHandler()
			runCommands(h, tt.setup)
			if got := h.Handle([]byte(tt.input)); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHandleLRange(t *testing.T) {
	rpushABC := []string{
		"*3\r\n$5\r\nRPUSH\r\n$6\r\nmylist\r\n$1\r\na\r\n",
		"*3\r\n$5\r\nRPUSH\r\n$6\r\nmylist\r\n$1\r\nb\r\n",
		"*3\r\n$5\r\nRPUSH\r\n$6\r\nmylist\r\n$1\r\nc\r\n",
	}
	tests := []struct {
		name  string
		setup []string
		input string
		want  string
	}{
		{
			name:  "returns elements in range",
			setup: rpushABC,
			input: "*4\r\n$6\r\nLRANGE\r\n$6\r\nmylist\r\n$1\r\n0\r\n$1\r\n1\r\n",
			want:  "*2\r\n$1\r\na\r\n$1\r\nb\r\n",
		},
		{
			name:  "negative indices",
			setup: rpushABC,
			input: "*4\r\n$6\r\nLRANGE\r\n$6\r\nmylist\r\n$2\r\n-2\r\n$2\r\n-1\r\n",
			want:  "*2\r\n$1\r\nb\r\n$1\r\nc\r\n",
		},
		{
			name:  "missing key returns empty array",
			input: "*4\r\n$6\r\nLRANGE\r\n$6\r\nunknown\r\n$1\r\n0\r\n$1\r\n-1\r\n",
			want:  "*0\r\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHandler()
			runCommands(h, tt.setup)
			if got := h.Handle([]byte(tt.input)); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
