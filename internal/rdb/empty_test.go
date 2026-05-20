package rdb_test

import (
	"bytes"
	"testing"

	"github.com/codecrafters-io/redis-starter-go/internal/rdb"
)

func TestEmptyStartsWithMagic(t *testing.T) {
	got := rdb.Empty()
	if !bytes.HasPrefix(got, []byte("REDIS")) {
		t.Errorf("empty RDB should start with REDIS magic, got % x", got[:min(5, len(got))])
	}
}

func TestEmptyEndsWithEOFMarker(t *testing.T) {
	got := rdb.Empty()
	if len(got) == 0 || got[len(got)-9] != 0xff {
		t.Errorf("empty RDB should contain 0xff EOF marker before 8-byte checksum, got tail % x", got[max(0, len(got)-10):])
	}
}
