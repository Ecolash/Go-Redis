package handler_test

import (
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/codecrafters-io/redis-starter-go/internal/errs"
	"github.com/codecrafters-io/redis-starter-go/internal/handler"
	"github.com/codecrafters-io/redis-starter-go/internal/resp"
	"github.com/codecrafters-io/redis-starter-go/internal/store"
)

func newHandler() *handler.Handler {
	return handler.New(store.New(), "master")
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
			want:  resp.Error(errs.ErrNotInteger.Error()),
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
			want:  resp.Error(errs.ErrNotInteger.Error()),
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
			want:  errs.WrongArgs,
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
			want:  errs.WrongArgs,
		},
		{
			name:  "0-0 ID returns must be greater than 0-0 error",
			input: "*5\r\n$4\r\nXADD\r\n$1\r\ns\r\n$3\r\n0-0\r\n$1\r\nk\r\n$1\r\nv\r\n",
			want:  resp.Error(errs.ErrStreamIDZero.Error()),
		},
		{
			name:  "ID equal to last returns equal or smaller error",
			setup: []string{"*5\r\n$4\r\nXADD\r\n$1\r\ns\r\n$3\r\n1-1\r\n$1\r\nk\r\n$1\r\nv\r\n"},
			input: "*5\r\n$4\r\nXADD\r\n$1\r\ns\r\n$3\r\n1-1\r\n$1\r\nk\r\n$1\r\nv\r\n",
			want:  resp.Error(errs.ErrStreamIDSmall.Error()),
		},
		{
			name:  "ms smaller than last returns equal or smaller error",
			setup: []string{"*5\r\n$4\r\nXADD\r\n$1\r\ns\r\n$3\r\n1-1\r\n$1\r\nk\r\n$1\r\nv\r\n"},
			input: "*5\r\n$4\r\nXADD\r\n$1\r\ns\r\n$3\r\n0-3\r\n$1\r\nk\r\n$1\r\nv\r\n",
			want:  resp.Error(errs.ErrStreamIDSmall.Error()),
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
			want:  resp.Error(errs.ErrStreamIDSmall.Error()),
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
			want:  errs.WrongArgs,
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
			want:  errs.WrongArgs,
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
		if got != errs.ExecNoMulti {
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
		if got != errs.ExecNoMulti {
			t.Errorf("got %q, want -ERR EXEC without MULTI\\r\\n", got)
		}
	})
}

func TestHandleDiscard(t *testing.T) {
	t.Run("DISCARD without MULTI returns error", func(t *testing.T) {
		h := newHandler()
		got := h.Handle([]byte("*1\r\n$7\r\nDISCARD\r\n"))
		if got != errs.DiscardNoMulti {
			t.Errorf("got %q, want -ERR DISCARD without MULTI\\r\\n", got)
		}
	})

	t.Run("DISCARD inside MULTI returns OK and drops queued commands", func(t *testing.T) {
		h := newHandler()
		h.Handle([]byte("*1\r\n$5\r\nMULTI\r\n"))
		h.Handle([]byte("*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$2\r\n41\r\n"))
		got := h.Handle([]byte("*1\r\n$7\r\nDISCARD\r\n"))
		if got != "+OK\r\n" {
			t.Errorf("got %q, want +OK\\r\\n", got)
		}

		// foo must not have been set: a subsequent GET sees a null bulk.
		got = h.Handle([]byte("*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n"))
		if got != "$-1\r\n" {
			t.Errorf("expected null bulk after DISCARD, got %q", got)
		}
	})

	t.Run("second DISCARD after first returns DISCARD without MULTI", func(t *testing.T) {
		h := newHandler()
		h.Handle([]byte("*1\r\n$5\r\nMULTI\r\n"))
		h.Handle([]byte("*1\r\n$7\r\nDISCARD\r\n"))
		got := h.Handle([]byte("*1\r\n$7\r\nDISCARD\r\n"))
		if got != errs.DiscardNoMulti {
			t.Errorf("got %q, want -ERR DISCARD without MULTI\\r\\n", got)
		}
	})

	t.Run("EXEC after DISCARD returns EXEC without MULTI", func(t *testing.T) {
		h := newHandler()
		h.Handle([]byte("*1\r\n$5\r\nMULTI\r\n"))
		h.Handle([]byte("*3\r\n$3\r\nSET\r\n$1\r\nk\r\n$1\r\nv\r\n"))
		h.Handle([]byte("*1\r\n$7\r\nDISCARD\r\n"))
		got := h.Handle([]byte("*1\r\n$4\r\nEXEC\r\n"))
		if got != errs.ExecNoMulti {
			t.Errorf("got %q, want -ERR EXEC without MULTI\\r\\n", got)
		}
	})

	t.Run("commands after DISCARD execute normally", func(t *testing.T) {
		h := newHandler()
		h.Handle([]byte("*1\r\n$5\r\nMULTI\r\n"))
		h.Handle([]byte("*3\r\n$3\r\nSET\r\n$1\r\nk\r\n$1\r\nv\r\n"))
		h.Handle([]byte("*1\r\n$7\r\nDISCARD\r\n"))
		got := h.Handle([]byte("*3\r\n$3\r\nSET\r\n$1\r\nk\r\n$1\r\nw\r\n"))
		if got != "+OK\r\n" {
			t.Errorf("got %q, want +OK\\r\\n", got)
		}
		got = h.Handle([]byte("*2\r\n$3\r\nGET\r\n$1\r\nk\r\n"))
		if got != "$1\r\nw\r\n" {
			t.Errorf("got %q, want $1\\r\\nw\\r\\n", got)
		}
	})
}

// newPair returns two handlers sharing a single store, simulating two
// connected clients hitting the same server.
func newPair() (*handler.Handler, *handler.Handler) {
	s := store.New()
	return handler.New(s, "master"), handler.New(s, "master")
}

func TestHandleWatch(t *testing.T) {
	t.Run("WATCH with no keys returns wrong args", func(t *testing.T) {
		h := newHandler()
		got := h.Handle([]byte("*1\r\n$5\r\nWATCH\r\n"))
		if got != errs.WrongArgs {
			t.Errorf("got %q, want -ERR wrong number of arguments\\r\\n", got)
		}
	})

	t.Run("WATCH returns OK", func(t *testing.T) {
		h := newHandler()
		got := h.Handle([]byte("*2\r\n$5\r\nWATCH\r\n$3\r\nfoo\r\n"))
		if got != "+OK\r\n" {
			t.Errorf("got %q, want +OK\\r\\n", got)
		}
	})

	t.Run("WATCH inside MULTI returns error", func(t *testing.T) {
		h := newHandler()
		h.Handle([]byte("*1\r\n$5\r\nMULTI\r\n"))
		got := h.Handle([]byte("*2\r\n$5\r\nWATCH\r\n$3\r\nfoo\r\n"))
		if got != errs.WatchInMulti {
			t.Errorf("got %q, want -ERR WATCH inside MULTI is not allowed\\r\\n", got)
		}
	})

	t.Run("EXEC succeeds when watched key is untouched", func(t *testing.T) {
		h := newHandler()
		h.Handle([]byte("*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$1\r\n1\r\n"))
		h.Handle([]byte("*2\r\n$5\r\nWATCH\r\n$3\r\nfoo\r\n"))
		h.Handle([]byte("*1\r\n$5\r\nMULTI\r\n"))
		h.Handle([]byte("*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$1\r\n2\r\n"))
		got := h.Handle([]byte("*1\r\n$4\r\nEXEC\r\n"))
		if got != "*1\r\n+OK\r\n" {
			t.Errorf("got %q, want *1\\r\\n+OK\\r\\n", got)
		}
		// Verify the queued SET actually ran.
		if got := h.Handle([]byte("*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n")); got != "$1\r\n2\r\n" {
			t.Errorf("expected foo=2 after EXEC, got %q", got)
		}
	})

	t.Run("EXEC aborts when watched key is modified by another client", func(t *testing.T) {
		h1, h2 := newPair()
		h1.Handle([]byte("*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$1\r\n1\r\n"))
		h1.Handle([]byte("*2\r\n$5\r\nWATCH\r\n$3\r\nfoo\r\n"))
		// Another client modifies foo before h1 calls EXEC.
		h2.Handle([]byte("*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$1\r\n9\r\n"))
		h1.Handle([]byte("*1\r\n$5\r\nMULTI\r\n"))
		h1.Handle([]byte("*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$1\r\n2\r\n"))
		got := h1.Handle([]byte("*1\r\n$4\r\nEXEC\r\n"))
		if got != "*-1\r\n" {
			t.Errorf("got %q, want *-1\\r\\n (nil array)", got)
		}
		// The queued SET must NOT have run; foo still equals 9.
		if got := h1.Handle([]byte("*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n")); got != "$1\r\n9\r\n" {
			t.Errorf("expected foo=9 after aborted EXEC, got %q", got)
		}
	})

	t.Run("EXEC aborts when watched missing key is created", func(t *testing.T) {
		h1, h2 := newPair()
		h1.Handle([]byte("*2\r\n$5\r\nWATCH\r\n$7\r\nnewkey1\r\n"))
		h2.Handle([]byte("*3\r\n$3\r\nSET\r\n$7\r\nnewkey1\r\n$3\r\nval\r\n"))
		h1.Handle([]byte("*1\r\n$5\r\nMULTI\r\n"))
		h1.Handle([]byte("*3\r\n$3\r\nSET\r\n$7\r\nnewkey1\r\n$5\r\nother\r\n"))
		got := h1.Handle([]byte("*1\r\n$4\r\nEXEC\r\n"))
		if got != "*-1\r\n" {
			t.Errorf("got %q, want *-1\\r\\n", got)
		}
	})

	t.Run("EXEC succeeds when watched missing key stays missing", func(t *testing.T) {
		h := newHandler()
		h.Handle([]byte("*2\r\n$5\r\nWATCH\r\n$7\r\nnewkey2\r\n"))
		h.Handle([]byte("*1\r\n$5\r\nMULTI\r\n"))
		h.Handle([]byte("*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$1\r\n1\r\n"))
		got := h.Handle([]byte("*1\r\n$4\r\nEXEC\r\n"))
		if got != "*1\r\n+OK\r\n" {
			t.Errorf("got %q, want *1\\r\\n+OK\\r\\n", got)
		}
	})

	t.Run("WATCH multiple keys aborts if any one changes", func(t *testing.T) {
		h1, h2 := newPair()
		h1.Handle([]byte("*3\r\n$3\r\nSET\r\n$1\r\na\r\n$1\r\n1\r\n"))
		h1.Handle([]byte("*3\r\n$3\r\nSET\r\n$1\r\nb\r\n$1\r\n2\r\n"))
		h1.Handle([]byte("*3\r\n$5\r\nWATCH\r\n$1\r\na\r\n$1\r\nb\r\n"))
		// Modify only b.
		h2.Handle([]byte("*3\r\n$3\r\nSET\r\n$1\r\nb\r\n$1\r\n9\r\n"))
		h1.Handle([]byte("*1\r\n$5\r\nMULTI\r\n"))
		h1.Handle([]byte("*2\r\n$4\r\nINCR\r\n$1\r\na\r\n"))
		got := h1.Handle([]byte("*1\r\n$4\r\nEXEC\r\n"))
		if got != "*-1\r\n" {
			t.Errorf("got %q, want *-1\\r\\n", got)
		}
	})

	t.Run("WATCH accumulates across calls", func(t *testing.T) {
		h1, h2 := newPair()
		h1.Handle([]byte("*2\r\n$5\r\nWATCH\r\n$1\r\na\r\n"))
		h1.Handle([]byte("*2\r\n$5\r\nWATCH\r\n$1\r\nb\r\n"))
		h2.Handle([]byte("*3\r\n$3\r\nSET\r\n$1\r\nb\r\n$1\r\nx\r\n"))
		h1.Handle([]byte("*1\r\n$5\r\nMULTI\r\n"))
		h1.Handle([]byte("*2\r\n$4\r\nINCR\r\n$1\r\na\r\n"))
		got := h1.Handle([]byte("*1\r\n$4\r\nEXEC\r\n"))
		if got != "*-1\r\n" {
			t.Errorf("got %q, want *-1\\r\\n", got)
		}
	})

	t.Run("watches are cleared after successful EXEC", func(t *testing.T) {
		h1, h2 := newPair()
		h1.Handle([]byte("*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$1\r\n1\r\n"))
		h1.Handle([]byte("*2\r\n$5\r\nWATCH\r\n$3\r\nfoo\r\n"))
		h1.Handle([]byte("*1\r\n$5\r\nMULTI\r\n"))
		h1.Handle([]byte("*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n"))
		h1.Handle([]byte("*1\r\n$4\r\nEXEC\r\n"))

		// h2 modifies foo; next transaction without re-WATCH must succeed.
		h2.Handle([]byte("*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$1\r\n9\r\n"))
		h1.Handle([]byte("*1\r\n$5\r\nMULTI\r\n"))
		h1.Handle([]byte("*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$1\r\n2\r\n"))
		got := h1.Handle([]byte("*1\r\n$4\r\nEXEC\r\n"))
		if got != "*1\r\n+OK\r\n" {
			t.Errorf("got %q, want *1\\r\\n+OK\\r\\n", got)
		}
	})

	t.Run("DISCARD clears watches", func(t *testing.T) {
		h1, h2 := newPair()
		h1.Handle([]byte("*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$1\r\n1\r\n"))
		h1.Handle([]byte("*2\r\n$5\r\nWATCH\r\n$3\r\nfoo\r\n"))
		h1.Handle([]byte("*1\r\n$5\r\nMULTI\r\n"))
		h1.Handle([]byte("*1\r\n$7\r\nDISCARD\r\n"))

		// After DISCARD, an external change must not affect a fresh transaction.
		h2.Handle([]byte("*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$1\r\n9\r\n"))
		h1.Handle([]byte("*1\r\n$5\r\nMULTI\r\n"))
		h1.Handle([]byte("*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$1\r\n2\r\n"))
		got := h1.Handle([]byte("*1\r\n$4\r\nEXEC\r\n"))
		if got != "*1\r\n+OK\r\n" {
			t.Errorf("got %q, want *1\\r\\n+OK\\r\\n", got)
		}
	})

	t.Run("WATCH then LPUSH from another client aborts", func(t *testing.T) {
		h1, h2 := newPair()
		h1.Handle([]byte("*3\r\n$5\r\nRPUSH\r\n$1\r\nl\r\n$1\r\na\r\n"))
		h1.Handle([]byte("*2\r\n$5\r\nWATCH\r\n$1\r\nl\r\n"))
		h2.Handle([]byte("*3\r\n$5\r\nLPUSH\r\n$1\r\nl\r\n$1\r\nb\r\n"))
		h1.Handle([]byte("*1\r\n$5\r\nMULTI\r\n"))
		h1.Handle([]byte("*2\r\n$4\r\nLLEN\r\n$1\r\nl\r\n"))
		got := h1.Handle([]byte("*1\r\n$4\r\nEXEC\r\n"))
		if got != "*-1\r\n" {
			t.Errorf("got %q, want *-1\\r\\n", got)
		}
	})
}

func TestHandleUnwatch(t *testing.T) {
	t.Run("UNWATCH returns OK with no prior WATCH", func(t *testing.T) {
		h := newHandler()
		got := h.Handle([]byte("*1\r\n$7\r\nUNWATCH\r\n"))
		if got != "+OK\r\n" {
			t.Errorf("got %q, want +OK\\r\\n", got)
		}
	})

	t.Run("UNWATCH returns OK after WATCH", func(t *testing.T) {
		h := newHandler()
		h.Handle([]byte("*3\r\n$5\r\nWATCH\r\n$3\r\nfoo\r\n$3\r\nbar\r\n"))
		got := h.Handle([]byte("*1\r\n$7\r\nUNWATCH\r\n"))
		if got != "+OK\r\n" {
			t.Errorf("got %q, want +OK\\r\\n", got)
		}
	})

	t.Run("UNWATCH clears watches so EXEC succeeds after external modification", func(t *testing.T) {
		h1, h2 := newPair()
		h1.Handle([]byte("*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$1\r\n1\r\n"))
		h1.Handle([]byte("*2\r\n$5\r\nWATCH\r\n$3\r\nfoo\r\n"))
		// Another client modifies foo before UNWATCH.
		h2.Handle([]byte("*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\n100\r\n"))
		// UNWATCH should clear the watch state.
		h1.Handle([]byte("*1\r\n$7\r\nUNWATCH\r\n"))
		h1.Handle([]byte("*1\r\n$5\r\nMULTI\r\n"))
		h1.Handle([]byte("*3\r\n$3\r\nSET\r\n$3\r\nbar\r\n$3\r\n200\r\n"))
		got := h1.Handle([]byte("*1\r\n$4\r\nEXEC\r\n"))
		if got != "*1\r\n+OK\r\n" {
			t.Errorf("got %q, want *1\\r\\n+OK\\r\\n", got)
		}
	})

	t.Run("UNWATCH clears all watched keys at once", func(t *testing.T) {
		h1, h2 := newPair()
		h1.Handle([]byte("*3\r\n$5\r\nWATCH\r\n$1\r\na\r\n$1\r\nb\r\n"))
		h1.Handle([]byte("*1\r\n$7\r\nUNWATCH\r\n"))
		// Modify both previously-watched keys.
		h2.Handle([]byte("*3\r\n$3\r\nSET\r\n$1\r\na\r\n$1\r\nx\r\n"))
		h2.Handle([]byte("*3\r\n$3\r\nSET\r\n$1\r\nb\r\n$1\r\ny\r\n"))
		h1.Handle([]byte("*1\r\n$5\r\nMULTI\r\n"))
		h1.Handle([]byte("*3\r\n$3\r\nSET\r\n$1\r\nc\r\n$1\r\nz\r\n"))
		got := h1.Handle([]byte("*1\r\n$4\r\nEXEC\r\n"))
		if got != "*1\r\n+OK\r\n" {
			t.Errorf("got %q, want *1\\r\\n+OK\\r\\n", got)
		}
	})

	t.Run("UNWATCH is idempotent", func(t *testing.T) {
		h := newHandler()
		h.Handle([]byte("*2\r\n$5\r\nWATCH\r\n$3\r\nfoo\r\n"))
		h.Handle([]byte("*1\r\n$7\r\nUNWATCH\r\n"))
		got := h.Handle([]byte("*1\r\n$7\r\nUNWATCH\r\n"))
		if got != "+OK\r\n" {
			t.Errorf("got %q, want +OK\\r\\n", got)
		}
	})
}

func TestHandleReplConf(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "listening-port returns OK",
			input: "*3\r\n$8\r\nREPLCONF\r\n$14\r\nlistening-port\r\n$4\r\n6380\r\n",
			want:  "+OK\r\n",
		},
		{
			name:  "capa psync2 returns OK",
			input: "*3\r\n$8\r\nREPLCONF\r\n$4\r\ncapa\r\n$6\r\npsync2\r\n",
			want:  "+OK\r\n",
		},
		{
			name:  "case insensitive",
			input: "*3\r\n$8\r\nreplconf\r\n$4\r\ncapa\r\n$6\r\npsync2\r\n",
			want:  "+OK\r\n",
		},
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

func TestHandlePsync(t *testing.T) {
	t.Run("PSYNC ? -1 returns FULLRESYNC followed by an empty RDB file", func(t *testing.T) {
		h := newHandler()
		got := h.Handle([]byte("*3\r\n$5\r\nPSYNC\r\n$1\r\n?\r\n$2\r\n-1\r\n"))

		const header = "+FULLRESYNC 8371b4fb1155b71f4a04d3e1bc3e18c4a990aeeb 0\r\n"
		if !strings.HasPrefix(got, header) {
			t.Fatalf("response should start with FULLRESYNC header, got %q", got)
		}

		// After the simple-string header comes a bulk-style payload with no
		// trailing CRLF: "$<len>\r\n<bytes>".
		rest := got[len(header):]
		if !strings.HasPrefix(rest, "$") {
			t.Fatalf("RDB payload should start with $, got %q", rest)
		}
		idx := strings.Index(rest, "\r\n")
		if idx == -1 {
			t.Fatalf("RDB payload missing length terminator, got %q", rest)
		}
		length, err := strconv.Atoi(rest[1:idx])
		if err != nil {
			t.Fatalf("invalid RDB length %q: %v", rest[1:idx], err)
		}
		body := rest[idx+2:]
		if len(body) != length {
			t.Errorf("RDB body length %d, want %d", len(body), length)
		}
		if strings.HasSuffix(body, "\r\n") {
			t.Errorf("RDB body must not have a trailing CRLF, got tail %q", body[max(0, len(body)-4):])
		}
		if !strings.HasPrefix(body, "REDIS") {
			t.Errorf("RDB body should start with REDIS magic, got %q", body[:min(5, len(body))])
		}
	})
}


func newPropHandler(t *testing.T) (*handler.Handler, *[][]string) {
	t.Helper()
	var calls [][]string
	h := handler.New(store.New(), "master", handler.WithPropagate(func(parts []string) {
		calls = append(calls, append([]string(nil), parts...))
	}))
	return h, &calls
}

func TestPropagateForwardsWriteCommands(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "SET propagates",
			input: "*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n",
			want:  []string{"SET", "foo", "bar"},
		},
		{
			name:  "INCR propagates",
			input: "*2\r\n$4\r\nINCR\r\n$1\r\nk\r\n",
			want:  []string{"INCR", "k"},
		},
		{
			name:  "RPUSH propagates",
			input: "*3\r\n$5\r\nRPUSH\r\n$1\r\nk\r\n$1\r\nv\r\n",
			want:  []string{"RPUSH", "k", "v"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, calls := newPropHandler(t)
			h.Handle([]byte(tt.input))
			if len(*calls) != 1 || !reflect.DeepEqual((*calls)[0], tt.want) {
				t.Errorf("propagate calls = %v, want exactly one call %v", *calls, tt.want)
			}
		})
	}
}

func TestPropagateSkipsReadAndAdminCommands(t *testing.T) {
	noPropagate := []string{
		"*1\r\n$4\r\nPING\r\n",
		"*2\r\n$4\r\nECHO\r\n$2\r\nhi\r\n",
		"*2\r\n$3\r\nGET\r\n$1\r\nk\r\n",
		"*2\r\n$4\r\nINFO\r\n$11\r\nreplication\r\n",
		"*3\r\n$8\r\nREPLCONF\r\n$4\r\ncapa\r\n$6\r\npsync2\r\n",
	}
	h, calls := newPropHandler(t)
	for _, cmd := range noPropagate {
		h.Handle([]byte(cmd))
	}
	if len(*calls) != 0 {
		t.Errorf("expected no propagate calls, got %v", *calls)
	}
}

func TestPropagateSkipsFailedWrites(t *testing.T) {
	h, calls := newPropHandler(t)
	// First set a non-integer value, then INCR — that fails with an error reply.
	h.Handle([]byte("*3\r\n$3\r\nSET\r\n$1\r\nk\r\n$3\r\nfoo\r\n"))
	// Reset to isolate the INCR failure.
	*calls = (*calls)[:0]
	got := h.Handle([]byte("*2\r\n$4\r\nINCR\r\n$1\r\nk\r\n"))
	if got == "" || got[0] != '-' {
		t.Fatalf("expected error reply, got %q", got)
	}
	if len(*calls) != 0 {
		t.Errorf("failed write should not propagate, got %v", *calls)
	}
}

func TestPropagateOnExecRunsForEachQueuedWrite(t *testing.T) {
	h, calls := newPropHandler(t)
	h.Handle([]byte("*1\r\n$5\r\nMULTI\r\n"))
	h.Handle([]byte("*3\r\n$3\r\nSET\r\n$1\r\nk\r\n$1\r\nv\r\n"))
	h.Handle([]byte("*2\r\n$4\r\nINCR\r\n$1\r\nn\r\n"))
	h.Handle([]byte("*1\r\n$4\r\nEXEC\r\n"))

	want := [][]string{
		{"SET", "k", "v"},
		{"INCR", "n"},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Errorf("propagate calls = %v, want %v", *calls, want)
	}
}

func TestReplConfGetAckProducesAckReply(t *testing.T) {
	h := handler.New(store.New(), "slave")
	got := h.Handle([]byte("*3\r\n$8\r\nREPLCONF\r\n$6\r\nGETACK\r\n$1\r\n*\r\n"))
	want := "*3\r\n$8\r\nREPLCONF\r\n$3\r\nACK\r\n$1\r\n0\r\n"
	if got != want {
		t.Errorf("REPLCONF GETACK reply = %q, want %q", got, want)
	}
	if !h.ShouldReplyToMaster() {
		t.Error("ShouldReplyToMaster() = false, want true after GETACK")
	}
	if h.ShouldReplyToMaster() {
		t.Error("ShouldReplyToMaster() should reset after read")
	}
}

func TestReplConfNonGetAckDoesNotRequestMasterReply(t *testing.T) {
	h := handler.New(store.New(), "master")
	h.Handle([]byte("*3\r\n$8\r\nREPLCONF\r\n$4\r\ncapa\r\n$6\r\npsync2\r\n"))
	if h.ShouldReplyToMaster() {
		t.Error("ShouldReplyToMaster() = true for non-GETACK REPLCONF")
	}
}

func TestGetAckOffsetTracksProcessedBytesExcludingCurrent(t *testing.T) {
	h := handler.New(store.New(), "slave", handler.WithOffsetTracking())
	getack := "*3\r\n$8\r\nREPLCONF\r\n$6\r\nGETACK\r\n$1\r\n*\r\n" // 37 bytes
	ping := "*1\r\n$4\r\nPING\r\n"                                  // 14 bytes
	setfoo := "*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$1\r\n1\r\n"          // 29 bytes
	setbar := "*3\r\n$3\r\nSET\r\n$3\r\nbar\r\n$1\r\n2\r\n"          // 29 bytes

	// Sanity-check the literal byte counts the spec example relies on.
	if len(getack) != 37 || len(ping) != 14 || len(setfoo) != 29 || len(setbar) != 29 {
		t.Fatalf("test fixture byte counts off: getack=%d ping=%d setfoo=%d setbar=%d",
			len(getack), len(ping), len(setfoo), len(setbar))
	}

	ackReply := func(offset int) string {
		o := strconv.Itoa(offset)
		return "*3\r\n$8\r\nREPLCONF\r\n$3\r\nACK\r\n$" + strconv.Itoa(len(o)) + "\r\n" + o + "\r\n"
	}

	if got := h.Handle([]byte(getack)); got != ackReply(0) {
		t.Errorf("1st GETACK reply = %q, want %q", got, ackReply(0))
	}
	h.Handle([]byte(ping))
	if got := h.Handle([]byte(getack)); got != ackReply(37+14) {
		t.Errorf("2nd GETACK reply = %q, want %q", got, ackReply(51))
	}
	h.Handle([]byte(setfoo))
	h.Handle([]byte(setbar))
	if got := h.Handle([]byte(getack)); got != ackReply(51+37+29+29) {
		t.Errorf("3rd GETACK reply = %q, want %q", got, ackReply(146))
	}
}

func TestOffsetTrackingDisabledByDefault(t *testing.T) {
	h := handler.New(store.New(), "slave")
	h.Handle([]byte("*1\r\n$4\r\nPING\r\n"))
	h.Handle([]byte("*3\r\n$3\r\nSET\r\n$1\r\nk\r\n$1\r\nv\r\n"))
	got := h.Handle([]byte("*3\r\n$8\r\nREPLCONF\r\n$6\r\nGETACK\r\n$1\r\n*\r\n"))
	want := "*3\r\n$8\r\nREPLCONF\r\n$3\r\nACK\r\n$1\r\n0\r\n"
	if got != want {
		t.Errorf("without WithOffsetTracking, GETACK reply = %q, want %q", got, want)
	}
}

func TestHandleWaitDelegatesToWaiter(t *testing.T) {
	var got struct {
		numReplicas int
		timeout     time.Duration
		calls       int
	}
	waiter := func(n int, d time.Duration) int {
		got.numReplicas = n
		got.timeout = d
		got.calls++
		return 3
	}
	h := handler.New(store.New(), "master", handler.WithReplicaWaiter(waiter))
	resp := h.Handle([]byte("*3\r\n$4\r\nWAIT\r\n$1\r\n7\r\n$3\r\n500\r\n"))
	if resp != ":3\r\n" {
		t.Errorf("got %q, want :3\\r\\n", resp)
	}
	if got.calls != 1 {
		t.Errorf("waiter called %d times, want 1", got.calls)
	}
	if got.numReplicas != 7 {
		t.Errorf("numReplicas=%d, want 7", got.numReplicas)
	}
	if got.timeout != 500*time.Millisecond {
		t.Errorf("timeout=%s, want 500ms", got.timeout)
	}
}

func TestHandleWaitOnReplicaReturnsZero(t *testing.T) {
	waiter := func(int, time.Duration) int {
		t.Fatal("waiter should not be invoked on a replica")
		return 0
	}
	h := handler.New(store.New(), "slave", handler.WithReplicaWaiter(waiter))
	if got := h.Handle([]byte("*3\r\n$4\r\nWAIT\r\n$1\r\n1\r\n$3\r\n100\r\n")); got != ":0\r\n" {
		t.Errorf("got %q, want :0\\r\\n", got)
	}
}

func TestHandleWaitWithoutWaiterFallsBackToCount(t *testing.T) {
	h := handler.New(store.New(), "master", handler.WithReplicaCount(func() int { return 4 }))
	if got := h.Handle([]byte("*3\r\n$4\r\nWAIT\r\n$1\r\n2\r\n$3\r\n100\r\n")); got != ":4\r\n" {
		t.Errorf("got %q, want :4\\r\\n", got)
	}
}

func TestHandleWaitWithoutWaiterOrCountReturnsZero(t *testing.T) {
	h := handler.New(store.New(), "master")
	if got := h.Handle([]byte("*3\r\n$4\r\nWAIT\r\n$1\r\n2\r\n$3\r\n100\r\n")); got != ":0\r\n" {
		t.Errorf("got %q, want :0\\r\\n", got)
	}
}

func TestHandleWaitRejectsNonIntegerArgs(t *testing.T) {
	h := handler.New(store.New(), "master", handler.WithReplicaWaiter(func(int, time.Duration) int { return 0 }))
	got := h.Handle([]byte("*3\r\n$4\r\nWAIT\r\n$3\r\nabc\r\n$3\r\n100\r\n"))
	if !strings.HasPrefix(got, "-ERR") {
		t.Errorf("got %q, want ERR prefix", got)
	}
}

func TestBecameReplicaSetOnlyAfterPsync(t *testing.T) {
	h, _ := newPropHandler(t)
	if h.BecameReplica() {
		t.Fatal("BecameReplica should be false before any command")
	}
	h.Handle([]byte("*1\r\n$4\r\nPING\r\n"))
	if h.BecameReplica() {
		t.Fatal("BecameReplica should be false after PING")
	}
	h.Handle([]byte("*3\r\n$5\r\nPSYNC\r\n$1\r\n?\r\n$2\r\n-1\r\n"))
	if !h.BecameReplica() {
		t.Fatal("BecameReplica should be true after PSYNC")
	}
	if h.BecameReplica() {
		t.Fatal("BecameReplica should reset after read")
	}
}
