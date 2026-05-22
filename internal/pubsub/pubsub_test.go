package pubsub_test

import (
	"testing"

	"github.com/codecrafters-io/redis-starter-go/internal/pubsub"
)

func TestSubscribeReturnsChannelCount(t *testing.T) {
	ps := pubsub.New()

	count := ps.Subscribe("client1", "news")
	if count != 1 {
		t.Errorf("first subscribe: got count %d, want 1", count)
	}

	count = ps.Subscribe("client1", "sports")
	if count != 2 {
		t.Errorf("second subscribe: got count %d, want 2", count)
	}
}

func TestSubscribeIsIdempotent(t *testing.T) {
	ps := pubsub.New()

	ps.Subscribe("client1", "news")
	count := ps.Subscribe("client1", "news")
	if count != 1 {
		t.Errorf("duplicate subscribe: got count %d, want 1", count)
	}
}

func TestSubscribeIsolatesClients(t *testing.T) {
	ps := pubsub.New()

	ps.Subscribe("client1", "news")
	ps.Subscribe("client1", "sports")
	count := ps.Subscribe("client2", "news")
	if count != 1 {
		t.Errorf("client2 first subscribe: got count %d, want 1", count)
	}
}
