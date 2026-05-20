package store

import (
	"time"
)

type valueKind int

const (
	kindString valueKind = iota
	kindList
	kindStream
)

// StreamEntry is a single entry in a Redis stream.
// Fields is a flat alternating list of keys and values, e.g. ["k1","v1","k2","v2"].
type StreamEntry struct {
	ID     string
	Fields []string 
}

type entry struct {
	kind      valueKind
	strVal    string
	listVal   []string
	streamVal []StreamEntry
	expiresAt time.Time // zero means no expiry
}

// BLPOPResult is the key+value returned by a blocking pop.
type BLPOPResult struct {
	Key string
	Val string
}

// blpopWaiter tracks a single pending BLPOP across one or more keys.
type blpopWaiter struct {
	keys []string
	ch   chan BLPOPResult // buffered size 1
}

// xreadWaiter tracks a single pending XREAD across one or more streams.
type xreadWaiter struct {
    key     string
    afterMs  int64
    afterSeq int64
    ch      chan []StreamEntry // buffered 1
}