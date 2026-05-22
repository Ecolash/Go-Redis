package aof_test

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func TestNewWriter(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, dir string)
		wantErr bool
		verify  func(t *testing.T, dir string)
	}{
		{
			name: "creates file from manifest",
			setup: func(t *testing.T, dir string) {
				writeManifest(t, dir, DefaultAppendDirName, DefaultAppendFilename, DefaultAppendFilename+".1.incr.aof")
			},
			verify: func(t *testing.T, dir string) {
				w, err := NewWriter(dir, DefaultAppendDirName, DefaultAppendFilename)
				if err != nil {
					t.Fatalf("NewWriter error: %v", err)
				}
				if err := w.Append("PING\r\n"); err != nil {
					t.Fatalf("Append error: %v", err)
				}
				if err := w.Close(); err != nil {
					t.Fatalf("Close error: %v", err)
				}
				data, err := os.ReadFile(filepath.Join(dir, DefaultAppendDirName, DefaultAppendFilename+".1.incr.aof"))
				if err != nil {
					t.Fatalf("read incr file: %v", err)
				}
				if string(data) != "PING\r\n" {
					t.Fatalf("content = %q", string(data))
				}
			},
		},
		{
			name:    "missing manifest",
			setup:   func(t *testing.T, dir string) {},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.setup != nil {
				tc.setup(t, dir)
			}
			w, err := NewWriter(dir, DefaultAppendDirName, DefaultAppendFilename)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("NewWriter error: %v", err)
			}
			if err := w.Close(); err != nil {
				t.Fatalf("Close error: %v", err)
			}
			if tc.verify != nil {
				tc.verify(t, dir)
			}
		})
	}
}

func TestWriterAppend(t *testing.T) {
	tests := []struct {
		name   string
		writes []string
		want   string
	}{
		{
			name:   "single append",
			writes: []string{"*1\r\n"},
			want:   "*1\r\n",
		},
		{
			name:   "multiple appends",
			writes: []string{"*1\r\n", "$4\r\nPING\r\n"},
			want:   "*1\r\n$4\r\nPING\r\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeManifest(t, dir, DefaultAppendDirName, DefaultAppendFilename, DefaultAppendFilename+".1.incr.aof")
			w, err := NewWriter(dir, DefaultAppendDirName, DefaultAppendFilename)
			if err != nil {
				t.Fatalf("NewWriter error: %v", err)
			}
			for _, chunk := range tc.writes {
				if err := w.Append(chunk); err != nil {
					t.Fatalf("Append error: %v", err)
				}
			}
			if err := w.Close(); err != nil {
				t.Fatalf("Close error: %v", err)
			}
			data, err := os.ReadFile(filepath.Join(dir, DefaultAppendDirName, DefaultAppendFilename+".1.incr.aof"))
			if err != nil {
				t.Fatalf("read incr file: %v", err)
			}
			if string(data) != tc.want {
				t.Fatalf("content = %q", string(data))
			}
		})
	}
}

func TestWriterClose(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "close prevents further writes"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeManifest(t, dir, DefaultAppendDirName, DefaultAppendFilename, DefaultAppendFilename+".1.incr.aof")
			w, err := NewWriter(dir, DefaultAppendDirName, DefaultAppendFilename)
			if err != nil {
				t.Fatalf("NewWriter error: %v", err)
			}
			if err := w.Close(); err != nil {
				t.Fatalf("Close error: %v", err)
			}
			if err := w.Append("PING\r\n"); err == nil {
				t.Fatalf("expected append error after close")
			}
		})
	}
}

func TestReplay(t *testing.T) {
	applyErr := errors.New("apply fail")
	tests := []struct {
		name    string
		setup   func(t *testing.T, dir string)
		apply   func([]byte) error
		wantErr error
		want    []string
	}{
		{
			name: "reads incremental file",
			setup: func(t *testing.T, dir string) {
				writeManifest(t, dir, DefaultAppendDirName, DefaultAppendFilename, DefaultAppendFilename+".1.incr.aof")
				content := resp.Array([]string{"SET", "foo", "bar"}) + resp.Array([]string{"INCR", "counter"})
				writeIncrFile(t, dir, DefaultAppendDirName, DefaultAppendFilename+".1.incr.aof", content)
			},
			apply: func(cmd []byte) error { return nil },
			want: []string{
				resp.Array([]string{"SET", "foo", "bar"}),
				resp.Array([]string{"INCR", "counter"}),
			},
		},
		{
			name: "apply returns error",
			setup: func(t *testing.T, dir string) {
				writeManifest(t, dir, DefaultAppendDirName, DefaultAppendFilename, DefaultAppendFilename+".1.incr.aof")
				content := resp.Array([]string{"PING"})
				writeIncrFile(t, dir, DefaultAppendDirName, DefaultAppendFilename+".1.incr.aof", content)
			},
			apply: func(cmd []byte) error { return applyErr },
			wantErr: applyErr,
		},
		{
			name:    "missing manifest",
			setup:   func(t *testing.T, dir string) {},
			apply:   func(cmd []byte) error { return nil },
			wantErr: errors.New("missing"),
		},
		{
			name: "missing incremental file",
			setup: func(t *testing.T, dir string) {
				writeManifest(t, dir, DefaultAppendDirName, DefaultAppendFilename, DefaultAppendFilename+".1.incr.aof")
			},
			apply:   func(cmd []byte) error { return nil },
			wantErr: errors.New("missing"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.setup != nil {
				tc.setup(t, dir)
			}
			var got []string
			err := Replay(dir, DefaultAppendDirName, DefaultAppendFilename, func(cmd []byte) error {
				got = append(got, string(cmd))
				if tc.apply != nil {
					return tc.apply(cmd)
				}
				return nil
			})
			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error")
				}
				if tc.wantErr == applyErr && !errors.Is(err, applyErr) {
					t.Fatalf("expected apply error, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Replay error: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("expected %d commands, got %d", len(tc.want), len(got))
			}
			for i, want := range tc.want {
				if got[i] != want {
					t.Fatalf("command %d = %q", i, got[i])
				}
			}
		})
	}
}

func TestActiveIncrFile(t *testing.T) {
	tests := []struct {
		name      string
		manifest  string
		want      string
		wantErr   bool
		writeFile bool
	}{
		{
			name:      "finds incremental entry",
			manifest:  "file base.aof seq 1 type b\nfile incr.aof seq 2 type i\n",
			want:      "incr.aof",
			writeFile: true,
		},
		{
			name:      "no incremental entry",
			manifest:  "file base.aof seq 1 type b\n",
			wantErr:   true,
			writeFile: true,
		},
		{
			name:      "missing manifest",
			wantErr:   true,
			writeFile: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			aofDir := filepath.Join(dir, DefaultAppendDirName)
			if err := os.MkdirAll(aofDir, 0o755); err != nil {
				t.Fatalf("mkdir aof dir: %v", err)
			}
			if tc.writeFile {
				if err := os.WriteFile(filepath.Join(aofDir, DefaultAppendFilename+".manifest"), []byte(tc.manifest), 0o644); err != nil {
					t.Fatalf("write manifest: %v", err)
				}
			}
			got, err := activeIncrFile(aofDir, DefaultAppendFilename)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("activeIncrFile error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("name = %q", got)
			}
		})
	}
}

func TestReadCommand(t *testing.T) {
	valid := resp.Array([]string{"SET", "foo", "bar"})
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "valid command",
			input: valid,
			want:  valid,
		},
		{
			name:    "invalid array header",
			input:   "+OK\r\n",
			wantErr: true,
		},
		{
			name:    "invalid array length",
			input:   "*x\r\n",
			wantErr: true,
		},
		{
			name:    "invalid bulk header",
			input:   "*1\r\n+OK\r\n",
			wantErr: true,
		},
		{
			name:    "invalid bulk length",
			input:   "*1\r\n$z\r\n",
			wantErr: true,
		},
		{
			name:    "truncated bulk body",
			input:   "*1\r\n$3\r\nab\r\n",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := bufio.NewReader(strings.NewReader(tc.input))
			got, err := readCommand(r)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("readCommand error: %v", err)
			}
			if string(got) != tc.want {
				t.Fatalf("command = %q", string(got))
			}
		})
	}
}

func writeManifest(t *testing.T, dir, appendDirName, appendFilename, incrName string) {
	t.Helper()
	aofDir := filepath.Join(dir, appendDirName)
	if err := os.MkdirAll(aofDir, 0o755); err != nil {
		t.Fatalf("mkdir aof dir: %v", err)
	}
	manifest := "file " + incrName + " seq 1 type i\n"
	if err := os.WriteFile(filepath.Join(aofDir, appendFilename+".manifest"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

func writeIncrFile(t *testing.T, dir, appendDirName, incrName, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, appendDirName, incrName), []byte(content), 0o644); err != nil {
		t.Fatalf("write incr file: %v", err)
	}
}