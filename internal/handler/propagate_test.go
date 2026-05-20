package handler_test

import (
	"reflect"
	"testing"

	"github.com/codecrafters-io/redis-starter-go/internal/handler"
	"github.com/codecrafters-io/redis-starter-go/internal/store"
)

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
