package resp

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseArray parses a RESP array of bulk strings and returns the elements.
// Input example: "*2\r\n$4\r\nECHO\r\n$3\r\nhey\r\n"
func ParseArray(data []byte) ([]string, error) {
	s := string(data)
	lines := strings.Split(s, "\r\n")
	if len(lines) == 0 || lines[0] == "" {
		return nil, fmt.Errorf("empty input")
	}
	if lines[0][0] != '*' {
		return nil, fmt.Errorf("expected array, got %q", lines[0])
	}
	count, err := strconv.Atoi(lines[0][1:])
	if err != nil {
		return nil, fmt.Errorf("invalid array length: %w", err)
	}

	result := make([]string, 0, count)
	i := 1
	for range count {
		if i >= len(lines) {
			return nil, fmt.Errorf("unexpected end of input")
		}
		if lines[i] == "" || lines[i][0] != '$' {
			return nil, fmt.Errorf("expected bulk string, got %q", lines[i])
		}
		i++ // skip the $N line
		if i >= len(lines) {
			return nil, fmt.Errorf("unexpected end of input after length")
		}
		result = append(result, lines[i])
		i++
	}
	return result, nil
}

func Error(msg string) string {
	return "-" + msg + "\r\n"
}

func BulkString(s string) string {
	return fmt.Sprintf("$%d\r\n%s\r\n", len(s), s)
}

func Integer(n int) string {
	return fmt.Sprintf(":%d\r\n", n)
}

func Array(strs []string) string {
	result := fmt.Sprintf("*%d\r\n", len(strs))
	for _, s := range strs {
		result += BulkString(s)
	}
	return result
}

// RawArray concatenates already-encoded RESP values into an array
func RawArray(parts []string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "*%d\r\n", len(parts))
	for _, p := range parts {
		sb.WriteString(p)
	}
	return sb.String()
}

// Entry is a single stream entry for use with StreamEntries.
type Entry struct {
	ID     string
	Fields []string
}

// XReadResult is a single stream result for use with StreamResults.
type XReadResult struct {
	Key     string
	Entries []Entry
}

// StreamResults encodes XREAD's response: *N → [[key, entries], ...].
func StreamResults(results []XReadResult) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "*%d\r\n", len(results))
	for _, r := range results {
		sb.WriteString("*2\r\n")
		sb.WriteString(BulkString(r.Key))
		sb.WriteString(StreamEntries(r.Entries))
	}
	return sb.String()
}

// StreamResult encodes a single-stream XREAD response.
func StreamResult(key string, entries []Entry) string {
	return StreamResults([]XReadResult{{Key: key, Entries: entries}})
}

// StreamEntries encodes a slice of stream entries as a RESP array of arrays.
// Each entry encodes as *2[id, *N[fields...]].
func StreamEntries(entries []Entry) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "*%d\r\n", len(entries))
	for _, e := range entries {
		sb.WriteString("*2\r\n")
		sb.WriteString(BulkString(e.ID))
		fmt.Fprintf(&sb, "*%d\r\n", len(e.Fields))
		for _, f := range e.Fields {
			sb.WriteString(BulkString(f))
		}
	}
	return sb.String()
}
