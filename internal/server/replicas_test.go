package server

import (
	"bytes"
	"errors"
	"testing"
)

type failWriter struct{ calls int }

func (f *failWriter) Write(p []byte) (int, error) { f.calls++; return 0, errors.New("nope") }

func TestReplicasBroadcastWritesToAll(t *testing.T) {
	r := newReplicas()
	var a, b bytes.Buffer
	r.Add(&a)
	r.Add(&b)
	r.Broadcast([]byte("hello"))
	if a.String() != "hello" || b.String() != "hello" {
		t.Errorf("a=%q b=%q, want both hello", a.String(), b.String())
	}
}

func TestReplicasBroadcastPrunesDeadWriters(t *testing.T) {
	r := newReplicas()
	good := &bytes.Buffer{}
	dead := &failWriter{}
	r.Add(dead)
	r.Add(good)
	r.Broadcast([]byte("x"))
	if r.Count() != 1 {
		t.Errorf("dead writer not pruned, count=%d", r.Count())
	}
	r.Broadcast([]byte("y"))
	if good.String() != "xy" {
		t.Errorf("good writer should have received both, got %q", good.String())
	}
	if dead.calls != 1 {
		t.Errorf("dead writer should be tried once then pruned, got %d", dead.calls)
	}
}
