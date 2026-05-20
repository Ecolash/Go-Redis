package resp_test

import (
	"bytes"
	"testing"

	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func TestFileEncoding(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want []byte
	}{
		{
			name: "empty payload",
			in:   []byte{},
			want: []byte("$0\r\n"),
		},
		{
			name: "ascii payload",
			in:   []byte("hello"),
			want: []byte("$5\r\nhello"),
		},
		{
			name: "binary payload with embedded CRLF and NUL",
			in:   []byte{0x52, 0x45, 0x44, 0x49, 0x53, 0x00, 0x0D, 0x0A, 0xFF},
			want: append([]byte("$9\r\n"), 0x52, 0x45, 0x44, 0x49, 0x53, 0x00, 0x0D, 0x0A, 0xFF),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resp.File(tt.in)
			if !bytes.Equal(got, tt.want) {
				t.Errorf("got %q (% x), want %q (% x)", got, got, tt.want, tt.want)
			}
		})
	}
}

func TestFileHasNoTrailingCRLF(t *testing.T) {
	got := resp.File([]byte("x"))
	if bytes.HasSuffix(got, []byte("x\r\n")) {
		t.Errorf("File output must not have a trailing CRLF after payload, got %q", got)
	}
}
