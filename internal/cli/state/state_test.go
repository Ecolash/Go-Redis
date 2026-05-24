package state_test

import (
	"testing"
	"time"

	"github.com/codecrafters-io/redis-starter-go/internal/cli/state"
)

func TestTxLifecycle(t *testing.T) {
	s := state.New("127.0.0.1", 6379)
	if s.InTx {
		t.Fatal("should not start in TX")
	}
	s.EnterTx()
	if !s.InTx {
		t.Fatal("should be in TX after EnterTx")
	}
	s.QueueCmd("SET foo bar")
	s.QueueCmd("GET foo")
	if len(s.TxQueue) != 2 {
		t.Fatalf("expected 2 queued cmds, got %d", len(s.TxQueue))
	}
	s.ExitTx()
	if s.InTx {
		t.Fatal("should not be in TX after ExitTx")
	}
	if len(s.TxQueue) != 0 {
		t.Fatal("TxQueue should be empty after ExitTx")
	}
}

func TestSubscribeLifecycle(t *testing.T) {
	s := state.New("127.0.0.1", 6379)
	s.Subscribe("news")
	s.Subscribe("alerts")
	if !s.InPubSub {
		t.Fatal("should be in PubSub after Subscribe")
	}
	if len(s.Subscriptions) != 2 {
		t.Fatalf("expected 2 subs, got %d", len(s.Subscriptions))
	}
	s.Unsubscribe("news")
	if len(s.Subscriptions) != 1 {
		t.Fatalf("expected 1 sub, got %d", len(s.Subscriptions))
	}
	s.Unsubscribe("alerts")
	if s.InPubSub {
		t.Fatal("should not be in PubSub when no subscriptions remain")
	}
}

func TestLatencyUpdate(t *testing.T) {
	s := state.New("127.0.0.1", 6379)
	s.UpdateLatency(5 * time.Millisecond)
	if s.Latency != 5*time.Millisecond {
		t.Fatalf("expected 5ms, got %v", s.Latency)
	}
}
