package aof

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type Writer struct {
	mu sync.Mutex
	f  *os.File
}

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

func Replay(dir, appendDirName, appendFilename string, apply func([]byte) error) error {
	aofDir := filepath.Join(dir, appendDirName)
	incrName, err := activeIncrFile(aofDir, appendFilename)
	if err != nil {
		return err
	}

	f, err := os.Open(filepath.Join(aofDir, incrName))
	if err != nil {
		return err
	}
	defer f.Close()

	r := bufio.NewReader(f)
	for {
		cmd, err := readCommand(r)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := apply(cmd); err != nil {
			return err
		}
	}
}

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

func readCommand(r *bufio.Reader) ([]byte, error) {
	header, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if len(header) < 3 || header[0] != '*' || !strings.HasSuffix(header, "\r\n") {
		return nil, fmt.Errorf("aof: invalid array header %q", header)
	}
	count, err := strconv.Atoi(strings.TrimSuffix(header[1:], "\r\n"))
	if err != nil {
		return nil, fmt.Errorf("aof: invalid array length %q: %w", header, err)
	}

	var raw strings.Builder
	raw.WriteString(header)
	for i := 0; i < count; i++ {
		lenLine, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		if len(lenLine) < 3 || lenLine[0] != '$' || !strings.HasSuffix(lenLine, "\r\n") {
			return nil, fmt.Errorf("aof: invalid bulk string header %q", lenLine)
		}
		raw.WriteString(lenLine)
		bulkLen, err := strconv.Atoi(strings.TrimSuffix(lenLine[1:], "\r\n"))
		if err != nil {
			return nil, fmt.Errorf("aof: invalid bulk length %q: %w", lenLine, err)
		}

		body := make([]byte, bulkLen+2)
		if _, err := io.ReadFull(r, body); err != nil {
			return nil, err
		}
		raw.Write(body)
	}

	return []byte(raw.String()), nil
}
