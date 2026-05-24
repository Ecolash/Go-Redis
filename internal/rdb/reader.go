package rdb

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"
)

const emptyHex = "524544495330303131fa0972656469732d76657205372e322e30fa0a72656469732d62697473c040fa056374696d65c26d08bc65fa08757365642d6d656dc2b0c41000fa08616f662d62617365c000fff06e3bfec0ff5aa2"

var empty []byte

func init() {
	b, err := hex.DecodeString(emptyHex)
	if err != nil {
		panic("rdb: invalid emptyHex constant: " + err.Error())
	}
	empty = b
}

func Empty() []byte {
	return empty
}

type Entry struct {
	Key       string
	Value     string
	ExpiresAt time.Time
}

func Load(path string) ([]Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return parse(data)
}

func parse(data []byte) ([]Entry, error) {
	if len(data) < 9 || string(data[:5]) != "REDIS" {
		return nil, fmt.Errorf("rdb: invalid magic header")
	}
	r := &rdbReader{data: data, pos: 9}
	return r.readAll()
}

type rdbReader struct {
	data []byte
	pos  int
}

func (r *rdbReader) readAll() ([]Entry, error) {
	var entries []Entry
	for r.pos < len(r.data) {
		b := r.data[r.pos]
		r.pos++
		switch b {
		case 0xFA: // metadata subsection — skip name and value
			if _, err := r.readString(); err != nil {
				return nil, err
			}
			if _, err := r.readString(); err != nil {
				return nil, err
			}
		case 0xFE: // database subsection — skip index
			if _, err := r.readSize(); err != nil {
				return nil, err
			}
		case 0xFB: // hash table sizes — skip both
			if _, err := r.readSize(); err != nil {
				return nil, err
			}
			if _, err := r.readSize(); err != nil {
				return nil, err
			}
		case 0xFC: // key with millisecond expiry
			if r.pos+8 > len(r.data) {
				return nil, io.ErrUnexpectedEOF
			}
			expMs := binary.LittleEndian.Uint64(r.data[r.pos : r.pos+8])
			r.pos += 8
			e, err := r.readKeyValue()
			if err != nil {
				return nil, err
			}
			e.ExpiresAt = time.UnixMilli(int64(expMs))
			entries = append(entries, e)
		case 0xFD: // key with second expiry
			if r.pos+4 > len(r.data) {
				return nil, io.ErrUnexpectedEOF
			}
			expS := binary.LittleEndian.Uint32(r.data[r.pos : r.pos+4])
			r.pos += 4
			e, err := r.readKeyValue()
			if err != nil {
				return nil, err
			}
			e.ExpiresAt = time.Unix(int64(expS), 0)
			entries = append(entries, e)
		case 0xFF: // end of file
			return entries, nil
		default: // treat as value-type byte (0x00 = string, others unsupported)
			if b != 0x00 {
				return nil, fmt.Errorf("rdb: unsupported value type 0x%02x", b)
			}
			key, err := r.readString()
			if err != nil {
				return nil, err
			}
			val, err := r.readString()
			if err != nil {
				return nil, err
			}
			entries = append(entries, Entry{Key: key, Value: val})
		}
	}
	return entries, nil
}

func (r *rdbReader) readKeyValue() (Entry, error) {
	if r.pos >= len(r.data) {
		return Entry{}, io.ErrUnexpectedEOF
	}
	typ := r.data[r.pos]
	r.pos++
	if typ != 0x00 {
		return Entry{}, fmt.Errorf("rdb: unsupported value type 0x%02x", typ)
	}
	key, err := r.readString()
	if err != nil {
		return Entry{}, err
	}
	val, err := r.readString()
	if err != nil {
		return Entry{}, err
	}
	return Entry{Key: key, Value: val}, nil
}

func (r *rdbReader) readSize() (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.ErrUnexpectedEOF
	}
	b := r.data[r.pos]
	r.pos++
	switch b >> 6 {
	case 0: // remaining 6 bits
		return int(b & 0x3F), nil
	case 1: // next 14 bits (big-endian)
		if r.pos >= len(r.data) {
			return 0, io.ErrUnexpectedEOF
		}
		next := r.data[r.pos]
		r.pos++
		return int(b&0x3F)<<8 | int(next), nil
	case 2: // next 4 bytes (big-endian)
		if r.pos+4 > len(r.data) {
			return 0, io.ErrUnexpectedEOF
		}
		n := binary.BigEndian.Uint32(r.data[r.pos : r.pos+4])
		r.pos += 4
		return int(n), nil
	case 3: // special encoding — signal with negative value encoding the sub-type
		return -(int(b&0x3F) + 1), nil
	}
	return 0, fmt.Errorf("rdb: unreachable size encoding")
}

// readString decodes a string-encoded value.
func (r *rdbReader) readString() (string, error) {
	if r.pos >= len(r.data) {
		return "", io.ErrUnexpectedEOF
	}
	// Peek at first byte to detect special (0b11) encoding before readSize.
	if r.data[r.pos]>>6 == 3 {
		sub := r.data[r.pos] & 0x3F
		r.pos++
		switch sub {
		case 0: // 8-bit integer
			if r.pos >= len(r.data) {
				return "", io.ErrUnexpectedEOF
			}
			n := int8(r.data[r.pos])
			r.pos++
			return fmt.Sprintf("%d", n), nil
		case 1: // 16-bit integer (little-endian)
			if r.pos+2 > len(r.data) {
				return "", io.ErrUnexpectedEOF
			}
			n := binary.LittleEndian.Uint16(r.data[r.pos : r.pos+2])
			r.pos += 2
			return fmt.Sprintf("%d", int16(n)), nil
		case 2: // 32-bit integer (little-endian)
			if r.pos+4 > len(r.data) {
				return "", io.ErrUnexpectedEOF
			}
			n := binary.LittleEndian.Uint32(r.data[r.pos : r.pos+4])
			r.pos += 4
			return fmt.Sprintf("%d", int32(n)), nil
		default:
			return "", fmt.Errorf("rdb: unsupported special string encoding %d", sub)
		}
	}
	n, err := r.readSize()
	if err != nil {
		return "", err
	}
	if r.pos+n > len(r.data) {
		return "", io.ErrUnexpectedEOF
	}
	s := string(r.data[r.pos : r.pos+n])
	r.pos += n
	return s, nil
}
