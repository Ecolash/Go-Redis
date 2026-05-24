package client

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RESPValue is a parsed RESP response.
type RESPValue struct {
	Type   byte         // '+', '-', ':', '$', '*'
	Value  string       // for simple/bulk/error/integer
	Items  []*RESPValue // for arrays
	IsNull bool
}

// Message is a Pub/Sub push message.
type Message struct {
	Kind    string // "message" or "pmessage"
	Pattern string // only for pmessage
	Channel string
	Payload string
}

// Client is a thread-safe Redis TCP client.
type Client struct {
	mu      sync.Mutex
	conn    net.Conn
	reader  *bufio.Reader
	latency time.Duration
}

// New dials host:port, optionally authenticates, and returns a Client.
func New(host string, port int, password string) (*Client, error) {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	c := &Client{conn: conn, reader: bufio.NewReader(conn)}
	if password != "" {
		resp, err := c.Do("AUTH", password)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("auth: %w", err)
		}
		if resp.Type == '-' {
			conn.Close()
			return nil, fmt.Errorf("auth: %s", resp.Value)
		}
	}
	return c, nil
}

// Do sends a command and returns the parsed RESP response.
func (c *Client) Do(args ...string) (*RESPValue, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	start := time.Now()
	if err := c.send(args...); err != nil {
		return nil, err
	}
	val, err := c.readValue()
	if err != nil {
		return nil, err
	}
	c.latency = time.Since(start)
	return val, nil
}

// Latency returns the round-trip time of the last Do call.
func (c *Client) Latency() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.latency
}

// Close closes the underlying TCP connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// Subscribe enters Pub/Sub mode and returns a channel of incoming messages.
// Call cancel() to unsubscribe and stop the goroutine.
func (c *Client) Subscribe(channels []string) (<-chan Message, func(), error) {
	c.mu.Lock()
	args := append([]string{"SUBSCRIBE"}, channels...)
	if err := c.send(args...); err != nil {
		c.mu.Unlock()
		return nil, nil, err
	}
	// drain the subscription confirmation responses
	for range channels {
		c.readValue() //nolint:errcheck
	}
	c.mu.Unlock()

	ch := make(chan Message, 32)
	done := make(chan struct{})

	go func() {
		defer close(ch)
		for {
			select {
			case <-done:
				return
			default:
			}
			c.mu.Lock()
			val, err := c.readValue()
			c.mu.Unlock()
			if err != nil {
				return
			}
			if val.Type != '*' || len(val.Items) < 3 {
				continue
			}
			kind := val.Items[0].Value
			if kind == "message" {
				ch <- Message{Kind: "message", Channel: val.Items[1].Value, Payload: val.Items[2].Value}
			} else if kind == "pmessage" && len(val.Items) == 4 {
				ch <- Message{Kind: "pmessage", Pattern: val.Items[1].Value, Channel: val.Items[2].Value, Payload: val.Items[3].Value}
			}
		}
	}()

	cancel := func() {
		close(done)
		c.Do("UNSUBSCRIBE") //nolint:errcheck
	}
	return ch, cancel, nil
}

func (c *Client) send(args ...string) error {
	var sb strings.Builder
	fmt.Fprintf(&sb, "*%d\r\n", len(args))
	for _, a := range args {
		fmt.Fprintf(&sb, "$%d\r\n%s\r\n", len(a), a)
	}
	_, err := fmt.Fprint(c.conn, sb.String())
	return err
}

func (c *Client) readValue() (*RESPValue, error) {
	line, err := c.reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimRight(line, "\r\n")
	if len(line) == 0 {
		return nil, fmt.Errorf("empty response line")
	}
	typ := line[0]
	data := line[1:]

	switch typ {
	case '+':
		return &RESPValue{Type: '+', Value: data}, nil
	case '-':
		return &RESPValue{Type: '-', Value: data}, nil
	case ':':
		return &RESPValue{Type: ':', Value: data}, nil
	case '$':
		n, err := strconv.Atoi(data)
		if err != nil {
			return nil, fmt.Errorf("invalid bulk length: %w", err)
		}
		if n == -1 {
			return &RESPValue{Type: '$', IsNull: true}, nil
		}
		buf := make([]byte, n+2)
		if _, err := c.reader.Read(buf); err != nil {
			return nil, err
		}
		return &RESPValue{Type: '$', Value: string(buf[:n])}, nil
	case '*':
		count, err := strconv.Atoi(data)
		if err != nil {
			return nil, fmt.Errorf("invalid array count: %w", err)
		}
		if count == -1 {
			return &RESPValue{Type: '*', IsNull: true}, nil
		}
		items := make([]*RESPValue, count)
		for i := range items {
			v, err := c.readValue()
			if err != nil {
				return nil, err
			}
			items[i] = v
		}
		return &RESPValue{Type: '*', Items: items}, nil
	default:
		return nil, fmt.Errorf("unknown RESP type %q in %q", typ, line)
	}
}
