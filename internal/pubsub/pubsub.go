package pubsub

import "sync"

const msgBufSize = 64

type subscriber struct {
	id       string
	channels map[string]bool
	msgs     chan string
}

type PubSub struct {
	mu          sync.RWMutex
	channels    map[string]map[string]*subscriber
	subscribers map[string]*subscriber
}

func New() *PubSub {
	return &PubSub{
		channels:    make(map[string]map[string]*subscriber),
		subscribers: make(map[string]*subscriber),
	}
}

func (ps *PubSub) Subscribe(subscriberID, channel string) int {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	sub, ok := ps.subscribers[subscriberID]
	if !ok {
		sub = &subscriber{
			id:       subscriberID,
			channels: make(map[string]bool),
			msgs:     make(chan string, msgBufSize),
		}
		ps.subscribers[subscriberID] = sub
	}

	if !sub.channels[channel] {
		sub.channels[channel] = true
		if ps.channels[channel] == nil {
			ps.channels[channel] = make(map[string]*subscriber)
		}
		ps.channels[channel][subscriberID] = sub
	}

	return len(sub.channels)
}

// MessageChan returns the receive-only message channel for the subscriber.
// Returns nil if the subscriber does not exist.
func (ps *PubSub) MessageChan(subscriberID string) <-chan string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	sub, ok := ps.subscribers[subscriberID]
	if !ok {
		return nil
	}
	return sub.msgs
}

// Publish sends encodedMsg to all subscribers of channel and returns the count.
func (ps *PubSub) Publish(channel, encodedMsg string) int {
	ps.mu.RLock()
	subs := ps.channels[channel]
	targets := make([]*subscriber, 0, len(subs))
	for _, sub := range subs {
		targets = append(targets, sub)
	}
	ps.mu.RUnlock()

	for _, sub := range targets {
		select {
		case sub.msgs <- encodedMsg:
		default:
			// subscriber channel full — drop message rather than block publisher
		}
	}
	return len(targets)
}

func (ps *PubSub) Unsubscribe(subscriberID, channel string) int {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	sub, ok := ps.subscribers[subscriberID]
	if !ok {
		return 0
	}
	delete(sub.channels, channel)
	if subs := ps.channels[channel]; subs != nil {
		delete(subs, subscriberID)
	}
	return len(sub.channels)
}

func (ps *PubSub) UnsubscribeAll(subscriberID string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	sub, ok := ps.subscribers[subscriberID]
	if !ok {
		return
	}
	for ch := range sub.channels {
		delete(ps.channels[ch], subscriberID)
	}
	delete(ps.subscribers, subscriberID)
}
