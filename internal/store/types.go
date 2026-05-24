package store

import (
	"time"
)

type valueKind int

const (
	kindString valueKind = iota
	kindList
	kindStream
	kindZSet
)

type StreamEntry struct {
	ID     string
	Fields []string
}

type entry struct {
	kind      valueKind
	strVal    string
	listVal   []string
	streamVal []StreamEntry
	zsetVal   *skipList
	expiresAt time.Time
}

type BLPOPResult struct {
	Key string
	Val string
}

type blpopWaiter struct {
	keys []string
	ch   chan BLPOPResult
}

type xreadWaiter struct {
	key      string
	afterMs  int64
	afterSeq int64
	ch       chan []StreamEntry
}

type ZSetMember struct {
	Score  float64
	Member string
}

type GeoMember struct {
	Lon    float64
	Lat    float64
	Member string
}

type GeoPosResult struct {
	Lon float64
	Lat float64
}
