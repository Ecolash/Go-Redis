package rdb

import "encoding/hex"

// emptyHex is a hex-encoded snapshot of an empty Redis RDB file. Used to
// satisfy a replica's FULLRESYNC handshake before any data exists.
const emptyHex = "524544495330303131fa0972656469732d76657205372e322e30fa0a72656469732d62697473c040fa056374696d65c26d08bc65fa08757365642d6d656dc2b0c41000fa08616f662d62617365c000fff06e3bfec0ff5aa2"

var empty []byte

func init() {
	b, err := hex.DecodeString(emptyHex)
	if err != nil {
		panic("rdb: invalid emptyHex constant: " + err.Error())
	}
	empty = b
}

// Empty returns the bytes of an empty RDB file snapshot.
func Empty() []byte {
	return empty
}
