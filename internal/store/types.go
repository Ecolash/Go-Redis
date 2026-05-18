package store

import (
	"errors"
	"time"
)

var (
	errStreamIDZero  = errors.New("ERR The ID specified in XADD must be greater than 0-0")
	errStreamIDSmall = errors.New("ERR The ID specified in XADD is equal or smaller than the target stream top item")
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
