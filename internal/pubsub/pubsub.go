package pubsub

import "sync"

type subscriber struct {
	id       string
	channels map[string]bool
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
		sub = &subscriber{id: subscriberID, channels: make(map[string]bool)}
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
