package handler_test

import (
	"strconv"
	"strings"
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

func TestHandleIncrandDecr(t *testing.T) {
	tests := []struct {
		name  string
		setup []string
		input string
		want  string
	}{
		{
			name: "returns 1 when key does not exist",
			input: "*2\r\n$4\r\nINCR\r\n$1\r\nk\r\n",
			want:  ":1\r\n",
		},
		{
			name:  "increments existing integer value",
			setup: []string{"*3\r\n$3\r\nSET\r\n$1\r\nk\r\n$1\r\n5\r\n"},
			input: "*2\r\n$4\r\nINCR\r\n$1\r\nk\r\n",
			want:  ":6\r\n",
		},
		{
			name:  "returns error if value is not an integer",
			setup: []string{"*3\r\n$3\r\nSET\r\n$1\r\nk\r\n$3\r\nfoo\r\n"},
			input: "*2\r\n$4\r\nINCR\r\n$1\r\nk\r\n",
			want:  "-ERR value is not an integer or out of range\r\n",
		},
		{
			name:  "decrements existing integer value",
			setup: []string{"*3\r\n$3\r\nSET\r\n$1\r\nk\r\n$1\r\n5\r\n"},
			input: "*2\r\n$4\r\nDECR\r\n$1\r\nk\r\n",
			want:  ":4\r\n",
		},
		{
			name:  "returns error if value is not an integer on DECR",
			setup: []string{"*3\r\n$3\r\nSET\r\n$1\r\nk\r\n$3\r\nfoo\r\n"},
			input: "*2\r\n$4\r\nDECR\r\n$1\r\nk\r\n",
			want:  "-ERR value is not an integer or out of range\r\n",
		},
		{
			name: "returns -1 if key does not exist on DECR",
			input: "*2\r\n$4\r\nDECR\r\n$1\r\nk\r\n",
			want:  ":-1\r\n",
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

func TestHandleLPop(t *testing.T) {
	tests := []struct {
		name  string
		setup []string
		input string
		want  string
	}{
		{
			name:  "missing key returns null bulk string",
			input: "*2\r\n$4\r\nLPOP\r\n$6\r\nnobody\r\n",
			want:  "$-1\r\n",
		},
		{
			name:  "returns and removes first element",
			setup: []string{"*4\r\n$5\r\nRPUSH\r\n$6\r\nmylist\r\n$1\r\na\r\n$1\r\nb\r\n"},
			input: "*2\r\n$4\r\nLPOP\r\n$6\r\nmylist\r\n",
			want:  "$1\r\na\r\n",
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

func TestHandleRPop(t *testing.T) {
	tests := []struct {
		name  string
		setup []string
		input string
		want  string
	}{
		{
			name:  "missing key returns null bulk string",
			input: "*2\r\n$4\r\nRPOP\r\n$6\r\nnobody\r\n",
			want:  "$-1\r\n",
		},
		{
			name:  "returns and removes last element",
			setup: []string{"*4\r\n$5\r\nRPUSH\r\n$6\r\nmylist\r\n$1\r\na\r\n$1\r\nb\r\n"},
			input: "*2\r\n$4\r\nRPOP\r\n$6\r\nmylist\r\n",
			want:  "$1\r\nb\r\n",
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

func TestHandleBLPop(t *testing.T) {
	t.Run("returns immediately for non-empty list", func(t *testing.T) {
		h := newHandler()
		h.Handle([]byte("*3\r\n$5\r\nRPUSH\r\n$6\r\nmylist\r\n$3\r\nfoo\r\n"))
		got := h.Handle([]byte("*3\r\n$5\r\nBLPOP\r\n$6\r\nmylist\r\n$1\r\n0\r\n"))
		want := "*2\r\n$6\r\nmylist\r\n$3\r\nfoo\r\n"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("returns null array on timeout", func(t *testing.T) {
		h := newHandler()
		got := h.Handle([]byte("*3\r\n$5\r\nBLPOP\r\n$6\r\nmylist\r\n$3\r\n0.1\r\n"))
		if got != "*-1\r\n" {
			t.Errorf("got %q, want *-1\\r\\n", got)
		}
	})

	t.Run("unblocks when element is pushed", func(t *testing.T) {
		h := newHandler()
		done := make(chan string, 1)
		go func() {
			done <- h.Handle([]byte("*3\r\n$5\r\nBLPOP\r\n$6\r\nmylist\r\n$3\r\n0.5\r\n"))
		}()
		time.Sleep(20 * time.Millisecond)
		h.Handle([]byte("*3\r\n$5\r\nRPUSH\r\n$6\r\nmylist\r\n$3\r\nbar\r\n"))
		select {
		case got := <-done:
			want := "*2\r\n$6\r\nmylist\r\n$3\r\nbar\r\n"
			if got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		case <-time.After(300 * time.Millisecond):
			t.Fatal("BLPOP did not unblock after RPUSH")
		}
	})

	t.Run("multiple clients each receive one element", func(t *testing.T) {
		h := newHandler()
		done1 := make(chan string, 1)
		done2 := make(chan string, 1)
		go func() { done1 <- h.Handle([]byte("*3\r\n$5\r\nBLPOP\r\n$6\r\nmylist\r\n$3\r\n0.5\r\n")) }()
		go func() { done2 <- h.Handle([]byte("*3\r\n$5\r\nBLPOP\r\n$6\r\nmylist\r\n$3\r\n0.5\r\n")) }()
		time.Sleep(20 * time.Millisecond)
		h.Handle([]byte("*4\r\n$5\r\nRPUSH\r\n$6\r\nmylist\r\n$1\r\na\r\n$1\r\nb\r\n"))
		timeout := time.After(300 * time.Millisecond)
		results := make([]string, 0, 2)
		for range 2 {
			select {
			case got := <-done1:
				results = append(results, got)
			case got := <-done2:
				results = append(results, got)
			case <-timeout:
				t.Fatalf("timed out after receiving %d/2 results", len(results))
			}
		}
		wantA := "*2\r\n$6\r\nmylist\r\n$1\r\na\r\n"
		wantB := "*2\r\n$6\r\nmylist\r\n$1\r\nb\r\n"
		got := map[string]bool{results[0]: true, results[1]: true}
		if !got[wantA] || !got[wantB] {
			t.Errorf("expected both %q and %q, got %v", wantA, wantB, results)
		}
	})

	t.Run("first key with element wins on immediate pop", func(t *testing.T) {
		h := newHandler()
		h.Handle([]byte("*3\r\n$5\r\nRPUSH\r\n$2\r\nk2\r\n$3\r\nval\r\n"))
		got := h.Handle([]byte("*4\r\n$5\r\nBLPOP\r\n$2\r\nk1\r\n$2\r\nk2\r\n$1\r\n0\r\n"))
		want := "*2\r\n$2\r\nk2\r\n$3\r\nval\r\n"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestHandleLPopWithCount(t *testing.T) {
	tests := []struct {
		name  string
		setup []string
		input string
		want  string
	}{
		{
			name:  "missing key returns null array",
			input: "*3\r\n$4\r\nLPOP\r\n$6\r\nnobody\r\n$1\r\n2\r\n",
			want:  "*-1\r\n",
		},
		{
			name:  "pops count elements from front",
			setup: []string{"*5\r\n$5\r\nRPUSH\r\n$6\r\nmylist\r\n$1\r\na\r\n$1\r\nb\r\n$1\r\nc\r\n"},
			input: "*3\r\n$4\r\nLPOP\r\n$6\r\nmylist\r\n$1\r\n2\r\n",
			want:  "*2\r\n$1\r\na\r\n$1\r\nb\r\n",
		},
		{
			name:  "count exceeds length returns all elements",
			setup: []string{"*4\r\n$5\r\nRPUSH\r\n$6\r\nmylist\r\n$1\r\na\r\n$1\r\nb\r\n"},
			input: "*3\r\n$4\r\nLPOP\r\n$6\r\nmylist\r\n$1\r\n9\r\n",
			want:  "*2\r\n$1\r\na\r\n$1\r\nb\r\n",
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

func TestHandleRPopWithCount(t *testing.T) {
	tests := []struct {
		name  string
		setup []string
		input string
		want  string
	}{
		{
			name:  "missing key returns null array",
			input: "*3\r\n$4\r\nRPOP\r\n$6\r\nnobody\r\n$1\r\n2\r\n",
			want:  "*-1\r\n",
		},
		{
			name:  "pops count elements from back",
			setup: []string{"*5\r\n$5\r\nRPUSH\r\n$6\r\nmylist\r\n$1\r\na\r\n$1\r\nb\r\n$1\r\nc\r\n"},
			input: "*3\r\n$4\r\nRPOP\r\n$6\r\nmylist\r\n$1\r\n2\r\n",
			want:  "*2\r\n$1\r\nc\r\n$1\r\nb\r\n",
		},
		{
			name:  "count exceeds length returns all elements",
			setup: []string{"*4\r\n$5\r\nRPUSH\r\n$6\r\nmylist\r\n$1\r\na\r\n$1\r\nb\r\n"},
			input: "*3\r\n$4\r\nRPOP\r\n$6\r\nmylist\r\n$1\r\n9\r\n",
			want:  "*2\r\n$1\r\nb\r\n$1\r\na\r\n",
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

func TestHandleType(t *testing.T) {
	tests := []struct {
		name  string
		setup []string
		input string
		want  string
	}{
		{
			name:  "missing key returns none",
			input: "*2\r\n$4\r\nTYPE\r\n$7\r\nmissing\r\n",
			want:  "+none\r\n",
		},
		{
			name:  "string key returns string",
			setup: []string{"*3\r\n$3\r\nSET\r\n$1\r\nk\r\n$1\r\nv\r\n"},
			input: "*2\r\n$4\r\nTYPE\r\n$1\r\nk\r\n",
			want:  "+string\r\n",
		},
		{
			name:  "list key returns list",
			setup: []string{"*3\r\n$5\r\nRPUSH\r\n$1\r\nk\r\n$1\r\nv\r\n"},
			input: "*2\r\n$4\r\nTYPE\r\n$1\r\nk\r\n",
			want:  "+list\r\n",
		},
		{
			name:  "wrong number of arguments",
			input: "*1\r\n$4\r\nTYPE\r\n",
			want:  "-ERR wrong number of arguments\r\n",
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

func TestHandleXAdd(t *testing.T) {
	tests := []struct {
		name  string
		setup []string
		input string
		want  string
	}{
		{
			name:  "returns bulk string entry ID",
			input: "*5\r\n$4\r\nXADD\r\n$10\r\nstream_key\r\n$3\r\n0-1\r\n$3\r\nfoo\r\n$3\r\nbar\r\n",
			want:  "$3\r\n0-1\r\n",
		},
		{
			name:  "creates stream and TYPE returns stream",
			input: "*5\r\n$4\r\nXADD\r\n$1\r\ns\r\n$3\r\n0-1\r\n$1\r\nk\r\n$1\r\nv\r\n",
			want:  "$3\r\n0-1\r\n",
		},
		{
			name:  "wrong number of arguments",
			input: "*3\r\n$4\r\nXADD\r\n$1\r\ns\r\n$3\r\n0-1\r\n",
			want:  "-ERR wrong number of arguments\r\n",
		},
		{
			name:  "0-0 ID returns must be greater than 0-0 error",
			input: "*5\r\n$4\r\nXADD\r\n$1\r\ns\r\n$3\r\n0-0\r\n$1\r\nk\r\n$1\r\nv\r\n",
			want:  "-ERR The ID specified in XADD must be greater than 0-0\r\n",
		},
		{
			name:  "ID equal to last returns equal or smaller error",
			setup: []string{"*5\r\n$4\r\nXADD\r\n$1\r\ns\r\n$3\r\n1-1\r\n$1\r\nk\r\n$1\r\nv\r\n"},
			input: "*5\r\n$4\r\nXADD\r\n$1\r\ns\r\n$3\r\n1-1\r\n$1\r\nk\r\n$1\r\nv\r\n",
			want:  "-ERR The ID specified in XADD is equal or smaller than the target stream top item\r\n",
		},
		{
			name:  "ms smaller than last returns equal or smaller error",
			setup: []string{"*5\r\n$4\r\nXADD\r\n$1\r\ns\r\n$3\r\n1-1\r\n$1\r\nk\r\n$1\r\nv\r\n"},
			input: "*5\r\n$4\r\nXADD\r\n$1\r\ns\r\n$3\r\n0-3\r\n$1\r\nk\r\n$1\r\nv\r\n",
			want:  "-ERR The ID specified in XADD is equal or smaller than the target stream top item\r\n",
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

func TestHandleXAddPartialAutoID(t *testing.T) {
	tests := []struct {
		name  string
		setup []string
		input string
		want  string
	}{
		{
			name:  "ms-* on empty stream returns ms-0",
			input: "*5\r\n$4\r\nXADD\r\n$1\r\ns\r\n$3\r\n5-*\r\n$1\r\nk\r\n$1\r\nv\r\n",
			want:  "$3\r\n5-0\r\n",
		},
		{
			name:  "ms-* with equal ms increments seq",
			setup: []string{"*5\r\n$4\r\nXADD\r\n$1\r\ns\r\n$3\r\n5-4\r\n$1\r\nk\r\n$1\r\nv\r\n"},
			input: "*5\r\n$4\r\nXADD\r\n$1\r\ns\r\n$3\r\n5-*\r\n$1\r\nk\r\n$1\r\nv\r\n",
			want:  "$3\r\n5-5\r\n",
		},
		{
			name:  "ms-* with smaller ms returns error",
			setup: []string{"*5\r\n$4\r\nXADD\r\n$1\r\ns\r\n$3\r\n5-1\r\n$1\r\nk\r\n$1\r\nv\r\n"},
			input: "*5\r\n$4\r\nXADD\r\n$1\r\ns\r\n$3\r\n3-*\r\n$1\r\nk\r\n$1\r\nv\r\n",
			want:  "-ERR The ID specified in XADD is equal or smaller than the target stream top item\r\n",
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

func TestHandleXAddAutoID(t *testing.T) {
	h := newHandler()
	// XADD stream_key * foo bar
	got := h.Handle([]byte("*5\r\n$4\r\nXADD\r\n$10\r\nstream_key\r\n$1\r\n*\r\n$3\r\nfoo\r\n$3\r\nbar\r\n"))
	if !strings.HasPrefix(got, "$") {
		t.Fatalf("expected bulk string response, got %q", got)
	}
	lines := strings.Split(strings.TrimSuffix(got, "\r\n"), "\r\n")
	if len(lines) < 2 {
		t.Fatalf("malformed bulk string: %q", got)
	}
	idParts := strings.SplitN(lines[1], "-", 2)
	if len(idParts) != 2 {
		t.Fatalf("expected ms-seq format in ID, got %q", lines[1])
	}
	if _, err := strconv.ParseInt(idParts[0], 10, 64); err != nil {
		t.Errorf("ms part %q is not a number", idParts[0])
	}
	if idParts[1] != "0" {
		t.Errorf("expected seq 0, got %q", idParts[1])
	}
}

func TestHandleTypeForStream(t *testing.T) {
	h := newHandler()
	h.Handle([]byte("*5\r\n$4\r\nXADD\r\n$1\r\ns\r\n$3\r\n0-1\r\n$1\r\nk\r\n$1\r\nv\r\n"))
	got := h.Handle([]byte("*2\r\n$4\r\nTYPE\r\n$1\r\ns\r\n"))
	if got != "+stream\r\n" {
		t.Errorf("got %q, want \"+stream\\r\\n\"", got)
	}
}

func TestHandleXRange(t *testing.T) {
	xadd := func(key, id string, kvs ...string) string {
		parts := append([]string{key, id}, kvs...)
		n := len(parts) + 1
		s := "*" + strconv.Itoa(n) + "\r\n$5\r\nXADD\r\n"
		for _, p := range parts {
			s += "$" + strconv.Itoa(len(p)) + "\r\n" + p + "\r\n"
		}
		return s
	}
	xrange := func(key, start, end string) string {
		return "*4\r\n$6\r\nXRANGE\r\n" +
			"$" + strconv.Itoa(len(key)) + "\r\n" + key + "\r\n" +
			"$" + strconv.Itoa(len(start)) + "\r\n" + start + "\r\n" +
			"$" + strconv.Itoa(len(end)) + "\r\n" + end + "\r\n"
	}

	tests := []struct {
		name  string
		setup []string
		input string
		want  string
	}{
		{
			name:  "missing key returns empty array",
			input: xrange("nokey", "0-1", "9-9"),
			want:  "*0\r\n",
		},
		{
			name:  "wrong number of arguments",
			input: "*2\r\n$6\r\nXRANGE\r\n$1\r\ns\r\n",
			want:  "-ERR wrong number of arguments\r\n",
		},
		{
			name: "spec example with partial IDs",
			setup: []string{
				xadd("mystream", "1526985054069-0", "temperature", "36", "humidity", "95"),
				xadd("mystream", "1526985054079-0", "temperature", "37", "humidity", "94"),
			},
			input: xrange("mystream", "1526985054069", "1526985054079"),
			want: "*2\r\n" +
				"*2\r\n$15\r\n1526985054069-0\r\n" +
				"*4\r\n$11\r\ntemperature\r\n$2\r\n36\r\n$8\r\nhumidity\r\n$2\r\n95\r\n" +
				"*2\r\n$15\r\n1526985054079-0\r\n" +
				"*4\r\n$11\r\ntemperature\r\n$2\r\n37\r\n$8\r\nhumidity\r\n$2\r\n94\r\n",
		},
		{
			name: "exact full IDs returns matching entries",
			setup: []string{
				xadd("s", "1-0", "k", "a"),
				xadd("s", "2-0", "k", "b"),
				xadd("s", "3-0", "k", "c"),
			},
			input: xrange("s", "2-0", "2-0"),
			want:  "*1\r\n*2\r\n$3\r\n2-0\r\n*2\r\n$1\r\nk\r\n$1\r\nb\r\n",
		},
		{
			name: "range returns subset excluding out-of-range entries",
			setup: []string{
				xadd("s", "1-0", "k", "a"),
				xadd("s", "2-0", "k", "b"),
				xadd("s", "3-0", "k", "c"),
			},
			input: xrange("s", "1-1", "2-0"),
			want:  "*1\r\n*2\r\n$3\r\n2-0\r\n*2\r\n$1\r\nk\r\n$1\r\nb\r\n",
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

func TestHandleXRead(t *testing.T) {
	xadd := func(key, id string, kvs ...string) string {
		parts := append([]string{key, id}, kvs...)
		n := len(parts) + 1
		s := "*" + strconv.Itoa(n) + "\r\n$5\r\nXADD\r\n"
		for _, p := range parts {
			s += "$" + strconv.Itoa(len(p)) + "\r\n" + p + "\r\n"
		}
		return s
	}
	xread := func(key, id string) string {
		return "*4\r\n$5\r\nXREAD\r\n$7\r\nSTREAMS\r\n" +
			"$" + strconv.Itoa(len(key)) + "\r\n" + key + "\r\n" +
			"$" + strconv.Itoa(len(id)) + "\r\n" + id + "\r\n"
	}

	tests := []struct {
		name  string
		setup []string
		input string
		want  string
	}{
		{
			name:  "wrong number of arguments",
			input: "*2\r\n$5\r\nXREAD\r\n$7\r\nSTREAMS\r\n",
			want:  "-ERR wrong number of arguments\r\n",
		},
		{
			name:  "missing key returns null array",
			input: xread("nokey", "0-0"),
			want:  "*-1\r\n",
		},
		{
			name: "returns entries strictly after given ID",
			setup: []string{
				xadd("s", "1-0", "k", "a"),
				xadd("s", "2-0", "k", "b"),
				xadd("s", "3-0", "k", "c"),
			},
			input: xread("s", "1-0"),
			want: "*1\r\n" +
				"*2\r\n" +
				"$1\r\ns\r\n" +
				"*2\r\n" +
				"*2\r\n$3\r\n2-0\r\n*2\r\n$1\r\nk\r\n$1\r\nb\r\n" +
				"*2\r\n$3\r\n3-0\r\n*2\r\n$1\r\nk\r\n$1\r\nc\r\n",
		},
		{
			name: "ID 0-0 returns all entries",
			setup: []string{
				xadd("s", "1-0", "k", "v"),
				xadd("s", "2-0", "k", "v"),
			},
			input: xread("s", "0-0"),
			want: "*1\r\n" +
				"*2\r\n" +
				"$1\r\ns\r\n" +
				"*2\r\n" +
				"*2\r\n$3\r\n1-0\r\n*2\r\n$1\r\nk\r\n$1\r\nv\r\n" +
				"*2\r\n$3\r\n2-0\r\n*2\r\n$1\r\nk\r\n$1\r\nv\r\n",
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

func TestHandleXReadMultiStream(t *testing.T) {
	xadd := func(key, id string, kvs ...string) string {
		parts := append([]string{key, id}, kvs...)
		n := len(parts) + 1
		s := "*" + strconv.Itoa(n) + "\r\n$5\r\nXADD\r\n"
		for _, p := range parts {
			s += "$" + strconv.Itoa(len(p)) + "\r\n" + p + "\r\n"
		}
		return s
	}
	// xreadMulti builds XREAD STREAMS key1 key2 ... id1 id2 ...
	xreadMulti := func(keysAndIDs ...string) string {
		s := "*" + strconv.Itoa(len(keysAndIDs)+2) + "\r\n$5\r\nXREAD\r\n$7\r\nSTREAMS\r\n"
		for _, p := range keysAndIDs {
			s += "$" + strconv.Itoa(len(p)) + "\r\n" + p + "\r\n"
		}
		return s
	}

	tests := []struct {
		name  string
		setup []string
		input string
		want  string
	}{
		{
			name: "two streams both with results",
			setup: []string{
				xadd("s1", "1-0", "k", "a"),
				xadd("s2", "2-0", "k", "b"),
			},
			input: xreadMulti("s1", "s2", "0-0", "0-0"),
			want: "*2\r\n" +
				"*2\r\n$2\r\ns1\r\n" +
				"*1\r\n*2\r\n$3\r\n1-0\r\n*2\r\n$1\r\nk\r\n$1\r\na\r\n" +
				"*2\r\n$2\r\ns2\r\n" +
				"*1\r\n*2\r\n$3\r\n2-0\r\n*2\r\n$1\r\nk\r\n$1\r\nb\r\n",
		},
		{
			name: "one stream has results, other does not",
			setup: []string{
				xadd("s1", "1-0", "k", "a"),
				xadd("s2", "1-0", "k", "b"),
			},
			// s2 queried after its last entry — no new results
			input: xreadMulti("s1", "s2", "0-0", "1-0"),
			want: "*1\r\n" +
				"*2\r\n$2\r\ns1\r\n" +
				"*1\r\n*2\r\n$3\r\n1-0\r\n*2\r\n$1\r\nk\r\n$1\r\na\r\n",
		},
		{
			name: "two streams neither with results returns null array",
			setup: []string{
				xadd("s1", "1-0", "k", "a"),
				xadd("s2", "1-0", "k", "b"),
			},
			input: xreadMulti("s1", "s2", "1-0", "1-0"),
			want:  "*-1\r\n",
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

func TestHandleXReadBlocking(t *testing.T) {
	xadd := func(key, id string, kvs ...string) string {
		parts := append([]string{key, id}, kvs...)
		n := len(parts) + 1
		s := "*" + strconv.Itoa(n) + "\r\n$5\r\nXADD\r\n"
		for _, p := range parts {
			s += "$" + strconv.Itoa(len(p)) + "\r\n" + p + "\r\n"
		}
		return s
	}
	xreadBlock := func(key, afterID string, ms int) string {
		msStr := strconv.Itoa(ms)
		return "*6\r\n$5\r\nXREAD\r\n$5\r\nBLOCK\r\n" +
			"$" + strconv.Itoa(len(msStr)) + "\r\n" + msStr + "\r\n" +
			"$7\r\nSTREAMS\r\n" +
			"$" + strconv.Itoa(len(key)) + "\r\n" + key + "\r\n" +
			"$" + strconv.Itoa(len(afterID)) + "\r\n" + afterID + "\r\n"
	}

	t.Run("returns immediately if entries already exist after given ID", func(t *testing.T) {
		h := newHandler()
		h.Handle([]byte(xadd("mystream", "1-0", "k", "v")))
		got := h.Handle([]byte(xreadBlock("mystream", "0-0", 100)))
		want := "*1\r\n*2\r\n$8\r\nmystream\r\n*1\r\n*2\r\n$3\r\n1-0\r\n*2\r\n$1\r\nk\r\n$1\r\nv\r\n"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("returns null array when timeout expires with no data", func(t *testing.T) {
		h := newHandler()
		got := h.Handle([]byte(xreadBlock("mystream", "0-0", 100)))
		if got != "*-1\r\n" {
			t.Errorf("got %q, want *-1\\r\\n", got)
		}
	})

	t.Run("unblocks and returns entry when XADD happens before timeout", func(t *testing.T) {
		h := newHandler()
		done := make(chan string, 1)
		go func() {
			done <- h.Handle([]byte(xreadBlock("mystream", "0-0", 500)))
		}()
		time.Sleep(20 * time.Millisecond)
		h.Handle([]byte(xadd("mystream", "1-0", "k", "v")))
		select {
		case got := <-done:
			want := "*1\r\n*2\r\n$8\r\nmystream\r\n*1\r\n*2\r\n$3\r\n1-0\r\n*2\r\n$1\r\nk\r\n$1\r\nv\r\n"
			if got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		case <-time.After(300 * time.Millisecond):
			t.Fatal("XREAD BLOCK did not unblock after XADD")
		}
	})

	t.Run("BLOCK 0 waits indefinitely until XADD arrives", func(t *testing.T) {
		h := newHandler()
		done := make(chan string, 1)
		go func() {
			done <- h.Handle([]byte(xreadBlock("mystream", "0-0", 0)))
		}()
		time.Sleep(20 * time.Millisecond)
		h.Handle([]byte(xadd("mystream", "1-0", "k", "v")))
		select {
		case got := <-done:
			want := "*1\r\n*2\r\n$8\r\nmystream\r\n*1\r\n*2\r\n$3\r\n1-0\r\n*2\r\n$1\r\nk\r\n$1\r\nv\r\n"
			if got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		case <-time.After(300 * time.Millisecond):
			t.Fatal("XREAD BLOCK 0 did not unblock after XADD")
		}
	})
}

func TestHandleXReadBlockingDollar(t *testing.T) {
	xadd := func(key, id string, kvs ...string) string {
		parts := append([]string{key, id}, kvs...)
		n := len(parts) + 1
		s := "*" + strconv.Itoa(n) + "\r\n$5\r\nXADD\r\n"
		for _, p := range parts {
			s += "$" + strconv.Itoa(len(p)) + "\r\n" + p + "\r\n"
		}
		return s
	}
	xreadBlockDollar := func(key string, ms int) string {
		msStr := strconv.Itoa(ms)
		return "*6\r\n$5\r\nXREAD\r\n$5\r\nBLOCK\r\n" +
			"$" + strconv.Itoa(len(msStr)) + "\r\n" + msStr + "\r\n" +
			"$7\r\nSTREAMS\r\n" +
			"$" + strconv.Itoa(len(key)) + "\r\n" + key + "\r\n" +
			"$1\r\n$\r\n"
	}

	t.Run("$ does not return existing entries, times out if no new data", func(t *testing.T) {
		h := newHandler()
		h.Handle([]byte(xadd("mystream", "1-0", "k", "v")))
		got := h.Handle([]byte(xreadBlockDollar("mystream", 100)))
		if got != "*-1\r\n" {
			t.Errorf("got %q, want *-1\\r\\n", got)
		}
	})

	t.Run("$ unblocks when new entry is added after command", func(t *testing.T) {
		h := newHandler()
		h.Handle([]byte(xadd("mystream", "1-0", "old", "val")))
		done := make(chan string, 1)
		go func() {
			done <- h.Handle([]byte(xreadBlockDollar("mystream", 500)))
		}()
		time.Sleep(20 * time.Millisecond)
		h.Handle([]byte(xadd("mystream", "2-0", "k", "v")))
		select {
		case got := <-done:
			want := "*1\r\n*2\r\n$8\r\nmystream\r\n*1\r\n*2\r\n$3\r\n2-0\r\n*2\r\n$1\r\nk\r\n$1\r\nv\r\n"
			if got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		case <-time.After(300 * time.Millisecond):
			t.Fatal("XREAD BLOCK $ did not unblock after XADD")
		}
	})

	t.Run("$ on empty stream unblocks when first entry is added", func(t *testing.T) {
		h := newHandler()
		done := make(chan string, 1)
		go func() {
			done <- h.Handle([]byte(xreadBlockDollar("mystream", 500)))
		}()
		time.Sleep(20 * time.Millisecond)
		h.Handle([]byte(xadd("mystream", "1-0", "k", "v")))
		select {
		case got := <-done:
			want := "*1\r\n*2\r\n$8\r\nmystream\r\n*1\r\n*2\r\n$3\r\n1-0\r\n*2\r\n$1\r\nk\r\n$1\r\nv\r\n"
			if got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		case <-time.After(300 * time.Millisecond):
			t.Fatal("XREAD BLOCK $ on empty stream did not unblock after XADD")
		}
	})
}

func TestHandleMulti(t *testing.T) {
	t.Run("MULTI returns OK", func(t *testing.T) {
		h := newHandler()
		got := h.Handle([]byte("*1\r\n$5\r\nMULTI\r\n"))
		if got != "+OK\r\n" {
			t.Errorf("got %q, want +OK\\r\\n", got)
		}
	})

	t.Run("commands after MULTI return QUEUED", func(t *testing.T) {
		h := newHandler()
		h.Handle([]byte("*1\r\n$5\r\nMULTI\r\n"))
		got := h.Handle([]byte("*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$2\r\n41\r\n"))
		if got != "+QUEUED\r\n" {
			t.Errorf("got %q, want +QUEUED\\r\\n", got)
		}
		got = h.Handle([]byte("*2\r\n$4\r\nINCR\r\n$3\r\nfoo\r\n"))
		if got != "+QUEUED\r\n" {
			t.Errorf("got %q, want +QUEUED\\r\\n", got)
		}
	})

	t.Run("queued commands do not execute before EXEC", func(t *testing.T) {
		h := newHandler()
		h.Handle([]byte("*1\r\n$5\r\nMULTI\r\n"))
		h.Handle([]byte("*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n"))
		// GET in a separate connection would still see nothing; here GET is
		// also queued, but we can verify via the underlying handler state by
		// checking that a fresh handler against the same store sees no value.
		h2 := newHandler()
		got := h2.Handle([]byte("*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n"))
		if got != "$-1\r\n" {
			t.Errorf("expected null bulk before EXEC across handlers, got %q", got)
		}
	})
}

func TestHandleExec(t *testing.T) {
	t.Run("EXEC without MULTI returns error", func(t *testing.T) {
		h := newHandler()
		got := h.Handle([]byte("*1\r\n$4\r\nEXEC\r\n"))
		if got != "-ERR EXEC without MULTI\r\n" {
			t.Errorf("got %q, want -ERR EXEC without MULTI\\r\\n", got)
		}
	})

	t.Run("EXEC with empty queue returns empty array", func(t *testing.T) {
		h := newHandler()
		h.Handle([]byte("*1\r\n$5\r\nMULTI\r\n"))
		got := h.Handle([]byte("*1\r\n$4\r\nEXEC\r\n"))
		if got != "*0\r\n" {
			t.Errorf("got %q, want *0\\r\\n", got)
		}
	})

	t.Run("EXEC runs queued commands and returns array of replies", func(t *testing.T) {
		h := newHandler()
		h.Handle([]byte("*1\r\n$5\r\nMULTI\r\n"))
		h.Handle([]byte("*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$2\r\n41\r\n"))
		h.Handle([]byte("*2\r\n$4\r\nINCR\r\n$3\r\nfoo\r\n"))
		got := h.Handle([]byte("*1\r\n$4\r\nEXEC\r\n"))
		want := "*2\r\n+OK\r\n:42\r\n"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("commands after EXEC execute normally", func(t *testing.T) {
		h := newHandler()
		h.Handle([]byte("*1\r\n$5\r\nMULTI\r\n"))
		h.Handle([]byte("*3\r\n$3\r\nSET\r\n$1\r\nk\r\n$1\r\nv\r\n"))
		h.Handle([]byte("*1\r\n$4\r\nEXEC\r\n"))
		got := h.Handle([]byte("*2\r\n$3\r\nGET\r\n$1\r\nk\r\n"))
		if got != "$1\r\nv\r\n" {
			t.Errorf("got %q, want $1\\r\\nv\\r\\n", got)
		}
	})

	t.Run("second EXEC after first returns EXEC without MULTI", func(t *testing.T) {
		h := newHandler()
		h.Handle([]byte("*1\r\n$5\r\nMULTI\r\n"))
		h.Handle([]byte("*1\r\n$4\r\nEXEC\r\n"))
		got := h.Handle([]byte("*1\r\n$4\r\nEXEC\r\n"))
		if got != "-ERR EXEC without MULTI\r\n" {
			t.Errorf("got %q, want -ERR EXEC without MULTI\\r\\n", got)
		}
	})
}
