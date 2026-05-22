package aof

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Writer appends RESP-encoded write commands to the active incremental AOF
// file. It is safe for concurrent use across connections.
type Writer struct {
	mu sync.Mutex
	f  *os.File
}

// NewWriter reads the manifest under dir/appendDirName to find the active
// incremental file (the entry with type "i") and opens it for appending.
func NewWriter(dir, appendDirName, appendFilename string) (*Writer, error) {
	aofDir := filepath.Join(dir, appendDirName)
	incrName, err := activeIncrFile(aofDir, appendFilename)
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(aofDir, incrName), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	return &Writer{f: f}, nil
}

// Append writes the given RESP-encoded command bytes to the AOF file.
func (w *Writer) Append(data string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	_, err := w.f.WriteString(data)
	return err
}

func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.f.Close()
}

// activeIncrFile parses the manifest and returns the file name of the entry
// whose type is "i" (the active incremental file).
func activeIncrFile(aofDir, appendFilename string) (string, error) {
	data, err := os.ReadFile(filepath.Join(aofDir, appendFilename+".manifest"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		var name, typ string
		for i := 0; i+1 < len(fields); i += 2 {
			switch fields[i] {
			case "file":
				name = fields[i+1]
			case "type":
				typ = fields[i+1]
			}
		}
		if typ == "i" && name != "" {
			return name, nil
		}
	}
	return "", fmt.Errorf("aof: no incremental file entry in manifest")
}
