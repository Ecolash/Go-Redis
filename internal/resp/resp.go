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

// Entry is a single stream entry for use with StreamEntries.
type Entry struct {
	ID     string
	Fields []string
}

// StreamResult encodes XREAD's outer array: *1 → [*2 → [key, entries]].
func StreamResult(key string, entries []Entry) string {
	return "*1\r\n*2\r\n" + BulkString(key) + StreamEntries(entries)
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
