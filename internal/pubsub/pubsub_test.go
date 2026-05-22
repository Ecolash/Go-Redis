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

func TestPublishToUnknownChannelReturnsZero(t *testing.T) {
	ps := pubsub.New()
	if got := ps.Publish("ghost", "hello"); got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

func TestPublishReturnsSubscriberCount(t *testing.T) {
	ps := pubsub.New()
	ps.Subscribe("c1", "news")
	ps.Subscribe("c2", "news")

	if got := ps.Publish("news", "encoded-msg"); got != 2 {
		t.Errorf("got %d, want 2", got)
	}
}

func TestPublishDeliversMsgToSubscriber(t *testing.T) {
	ps := pubsub.New()
	ps.Subscribe("c1", "news")
	ch := ps.MessageChan("c1")

	ps.Publish("news", "hello-encoded")

	select {
	case got := <-ch:
		if got != "hello-encoded" {
			t.Errorf("got %q, want %q", got, "hello-encoded")
		}
	default:
		t.Error("expected message on channel, got nothing")
	}
}

func TestMessageChanReturnsNilForUnknownSubscriber(t *testing.T) {
	ps := pubsub.New()
	if ch := ps.MessageChan("nobody"); ch != nil {
		t.Error("expected nil channel for unknown subscriber")
	}
}
