package aof

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func TestReplayReadsIncrementalFile(t *testing.T) {
	dir := t.TempDir()
	aofDir := filepath.Join(dir, DefaultAppendDirName)
	if err := os.MkdirAll(aofDir, 0o755); err != nil {
		t.Fatalf("mkdir aof dir: %v", err)
	}

	incrName := DefaultAppendFilename + ".1.incr.aof"
	manifest := "file " + incrName + " seq 1 type i\n"
	if err := os.WriteFile(filepath.Join(aofDir, DefaultAppendFilename+".manifest"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	content := resp.Array([]string{"SET", "foo", "bar"}) + resp.Array([]string{"INCR", "counter"})
	if err := os.WriteFile(filepath.Join(aofDir, incrName), []byte(content), 0o644); err != nil {
		t.Fatalf("write incr file: %v", err)
	}

	var got [][]byte
	if err := Replay(dir, DefaultAppendDirName, DefaultAppendFilename, func(cmd []byte) error {
		got = append(got, append([]byte(nil), cmd...))
		return nil
	}); err != nil {
		t.Fatalf("Replay returned error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(got))
	}
	if string(got[0]) != resp.Array([]string{"SET", "foo", "bar"}) {
		t.Fatalf("first command = %q", string(got[0]))
	}
	if string(got[1]) != resp.Array([]string{"INCR", "counter"}) {
		t.Fatalf("second command = %q", string(got[1]))
	}
}