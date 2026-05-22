package rdb

import (
	"encoding/binary"
	"testing"
)

// build assembles a minimal valid RDB byte stream from a header, the given
// body bytes, and an EOF marker + 8-byte checksum.
func build(body []byte) []byte {
	out := []byte("REDIS0011")
	out = append(out, body...)
	out = append(out, 0xFF)
	out = append(out, make([]byte, 8)...) // checksum (ignored by parser)
	return out
}

func lenStr(s string) []byte {
	b := []byte{byte(len(s))}
	return append(b, s...)
}

func TestParseLengthPrefixedString(t *testing.T) {
	body := []byte{0x00} // value type: string
	body = append(body, lenStr("foo")...)
	body = append(body, lenStr("bar")...)

	entries, err := parse(build(body))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(entries) != 1 || entries[0].Key != "foo" || entries[0].Value != "bar" {
		t.Fatalf("got %+v, want foo=bar", entries)
	}
}

func TestParseIntegerEncodedValues(t *testing.T) {
	int16Bytes := func(v int16) []byte {
		b := make([]byte, 2)
		binary.LittleEndian.PutUint16(b, uint16(v))
		return b
	}
	int32Bytes := func(v int32) []byte {
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, uint32(v))
		return b
	}

	var body []byte
	// key "a" -> 8-bit int 123
	body = append(body, 0x00)
	body = append(body, lenStr("a")...)
	body = append(body, 0xC0, 123)
	// key "b" -> 16-bit int 12345
	body = append(body, 0x00)
	body = append(body, lenStr("b")...)
	body = append(body, 0xC1)
	body = append(body, int16Bytes(12345)...)
	// key "c" -> 32-bit int 1234567
	body = append(body, 0x00)
	body = append(body, lenStr("c")...)
	body = append(body, 0xC2)
	body = append(body, int32Bytes(1234567)...)

	entries, err := parse(build(body))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := map[string]string{"a": "123", "b": "12345", "c": "1234567"}
	if len(entries) != len(want) {
		t.Fatalf("got %d entries, want %d: %+v", len(entries), len(want), entries)
	}
	for _, e := range entries {
		if want[e.Key] != e.Value {
			t.Errorf("key %q: got %q, want %q", e.Key, e.Value, want[e.Key])
		}
	}
}

func TestParseWithMetadataAndDBSections(t *testing.T) {
	var body []byte
	// metadata: redis-ver = 6.0.16
	body = append(body, 0xFA)
	body = append(body, lenStr("redis-ver")...)
	body = append(body, lenStr("6.0.16")...)
	// database section
	body = append(body, 0xFE, 0x00) // db index 0
	body = append(body, 0xFB, 0x01, 0x00) // ht sizes: 1 key, 0 expires
	body = append(body, 0x00)
	body = append(body, lenStr("foo")...)
	body = append(body, lenStr("bar")...)

	entries, err := parse(build(body))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(entries) != 1 || entries[0].Key != "foo" || entries[0].Value != "bar" {
		t.Fatalf("got %+v, want foo=bar", entries)
	}
}

func TestLoadMissingFileReturnsNil(t *testing.T) {
	entries, err := Load("/nonexistent/path/dump.rdb")
	if err != nil {
		t.Fatalf("expected nil error for missing file, got %v", err)
	}
	if entries != nil {
		t.Fatalf("expected nil entries, got %+v", entries)
	}
}
