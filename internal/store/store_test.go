package store_test

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/codecrafters-io/redis-starter-go/internal/store"
)

func TestGet(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(s *store.Store)
		key    string
		wantOK bool
		want   string
	}{
		{
			name:   "missing key",
			setup:  func(s *store.Store) {},
			key:    "missing",
			wantOK: false,
		},
		{
			name:   "after set",
			setup:  func(s *store.Store) { s.Set("foo", "bar", 0) },
			key:    "foo",
			wantOK: true,
			want:   "bar",
		},
		{
			name: "overwritten key returns latest value",
			setup: func(s *store.Store) {
				s.Set("key", "first", 0)
				s.Set("key", "second", 0)
			},
			key:    "key",
			wantOK: true,
			want:   "second",
		},
		{
			name:   "before TTL expires",
			setup:  func(s *store.Store) { s.Set("foo", "bar", 200*time.Millisecond) },
			key:    "foo",
			wantOK: true,
			want:   "bar",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := store.New()
			tt.setup(s)
			val, ok := s.Get(tt.key)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && val != tt.want {
				t.Errorf("val = %q, want %q", val, tt.want)
			}
		})
	}
}

func TestGetAfterTTLExpires(t *testing.T) {
	s := store.New()
	s.Set("foo", "bar", 20*time.Millisecond)
	if _, ok := s.Get("foo"); !ok {
		t.Fatal("expected key to exist before TTL expires")
	}
	time.Sleep(30 * time.Millisecond)
	if _, ok := s.Get("foo"); ok {
		t.Error("expected key to be expired and missing")
	}
}

func TestRPush(t *testing.T) {
	tests := []struct {
		name      string
		pushes    [][]string
		wantCount int
		wantList  []string
	}{
		{
			name:      "creates list and returns 1",
			pushes:    [][]string{{"foo"}},
			wantCount: 1,
			wantList:  []string{"foo"},
		},
		{
			name:      "appends to existing list",
			pushes:    [][]string{{"foo"}, {"bar"}},
			wantCount: 2,
			wantList:  []string{"foo", "bar"},
		},
		{
			name:      "multiple values in one call appended in order",
			pushes:    [][]string{{"a", "b", "c"}},
			wantCount: 3,
			wantList:  []string{"a", "b", "c"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := store.New()
			var n int
			for _, vals := range tt.pushes {
				n = s.RPush("mylist", vals...)
			}
			if n != tt.wantCount {
				t.Errorf("count = %d, want %d", n, tt.wantCount)
			}
			got, _ := s.LRange("mylist", 0, -1)
			if len(got) != len(tt.wantList) {
				t.Fatalf("list len = %d, want %d; got %v", len(got), len(tt.wantList), got)
			}
			for i, v := range tt.wantList {
				if got[i] != v {
					t.Errorf("index %d: got %q, want %q", i, got[i], v)
				}
			}
		})
	}
}

func TestLPush(t *testing.T) {
	tests := []struct {
		name      string
		pushes    [][]string
		wantCount int
		wantList  []string
	}{
		{
			name:      "creates list and returns 1",
			pushes:    [][]string{{"foo"}},
			wantCount: 1,
			wantList:  []string{"foo"},
		},
		{
			name:      "prepends to existing list",
			pushes:    [][]string{{"foo"}, {"bar"}},
			wantCount: 2,
			wantList:  []string{"bar", "foo"},
		},
		{
			name:      "multiple values pushed left to right resulting in reverse order",
			pushes:    [][]string{{"a", "b", "c"}},
			wantCount: 3,
			wantList:  []string{"c", "b", "a"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := store.New()
			var n int
			for _, vals := range tt.pushes {
				n = s.LPush("mylist", vals...)
			}
			if n != tt.wantCount {
				t.Errorf("count = %d, want %d", n, tt.wantCount)
			}
			got, ok := s.LRange("mylist", 0, -1)
			if !ok {
				t.Fatal("expected list to exist")
			}
			if len(got) != len(tt.wantList) {
				t.Fatalf("list len = %d, want %d; got %v", len(got), len(tt.wantList), got)
			}
			for i, v := range tt.wantList {
				if got[i] != v {
					t.Errorf("index %d: got %q, want %q", i, got[i], v)
				}
			}
		})
	}
}

func TestLLen(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		want   int
	}{
		{
			name:   "missing key returns 0",
			values: nil,
			want:   0,
		},
		{
			name:   "empty list returns 0",
			values: []string{},
			want:   0,
		},
		{
			name:   "non-empty list returns correct count",
			values: []string{"a", "b", "c"},
			want:   3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := store.New()
			if tt.values != nil {
				s.RPush("mylist", tt.values...)
			}
			if got := s.LLen("mylist"); got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}	
}

func TestLPop(t *testing.T) {
	tests := []struct {
		name     string
		setup    []string
		wantVal  string
		wantOK   bool
		wantList []string
	}{
		{
			name:   "missing key returns not ok",
			wantOK: false,
		},
		{
			name:     "removes and returns first element",
			setup:    []string{"a", "b", "c"},
			wantVal:  "a",
			wantOK:   true,
			wantList: []string{"b", "c"},
		},
		{
			name:     "single element list becomes empty",
			setup:    []string{"only"},
			wantVal:  "only",
			wantOK:   true,
			wantList: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := store.New()
			if tt.setup != nil {
				s.RPush("mylist", tt.setup...)
			}
			val, ok := s.LPop("mylist")
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && val != tt.wantVal {
				t.Errorf("val = %q, want %q", val, tt.wantVal)
			}
			if tt.wantList != nil {
				got, _ := s.LRange("mylist", 0, -1)
				if len(got) != len(tt.wantList) {
					t.Fatalf("list len = %d, want %d; got %v", len(got), len(tt.wantList), got)
				}
				for i, v := range tt.wantList {
					if got[i] != v {
						t.Errorf("index %d: got %q, want %q", i, got[i], v)
					}
				}
			}
		})
	}
}

func TestRPop(t *testing.T) {
	tests := []struct {
		name     string
		setup    []string
		wantVal  string
		wantOK   bool
		wantList []string
	}{
		{
			name:   "missing key returns not ok",
			wantOK: false,
		},
		{
			name:     "removes and returns last element",
			setup:    []string{"a", "b", "c"},
			wantVal:  "c",
			wantOK:   true,
			wantList: []string{"a", "b"},
		},
		{
			name:     "single element list becomes empty",
			setup:    []string{"only"},
			wantVal:  "only",
			wantOK:   true,
			wantList: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := store.New()
			if tt.setup != nil {
				s.RPush("mylist", tt.setup...)
			}
			val, ok := s.RPop("mylist")
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && val != tt.wantVal {
				t.Errorf("val = %q, want %q", val, tt.wantVal)
			}
			if tt.wantList != nil {
				got, _ := s.LRange("mylist", 0, -1)
				if len(got) != len(tt.wantList) {
					t.Fatalf("list len = %d, want %d; got %v", len(got), len(tt.wantList), got)
				}
				for i, v := range tt.wantList {
					if got[i] != v {
						t.Errorf("index %d: got %q, want %q", i, got[i], v)
					}
				}
			}
		})
	}
}

func TestLPopCount(t *testing.T) {
	tests := []struct {
		name     string
		setup    []string
		count    int
		wantVals []string
		wantOK   bool
		wantList []string
	}{
		{
			name:   "missing key returns not ok",
			count:  2,
			wantOK: false,
		},
		{
			name:     "pops requested count from front",
			setup:    []string{"a", "b", "c", "d"},
			count:    2,
			wantVals: []string{"a", "b"},
			wantOK:   true,
			wantList: []string{"c", "d"},
		},
		{
			name:     "count exceeds length returns all elements",
			setup:    []string{"a", "b"},
			count:    10,
			wantVals: []string{"a", "b"},
			wantOK:   true,
			wantList: []string{},
		},
		{
			name:     "count zero returns empty slice",
			setup:    []string{"a", "b"},
			count:    0,
			wantVals: []string{},
			wantOK:   true,
			wantList: []string{"a", "b"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := store.New()
			if tt.setup != nil {
				s.RPush("mylist", tt.setup...)
			}
			vals, ok := s.LPopCount("mylist", tt.count)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok {
				if len(vals) != len(tt.wantVals) {
					t.Fatalf("vals len = %d, want %d; got %v", len(vals), len(tt.wantVals), vals)
				}
				for i, v := range tt.wantVals {
					if vals[i] != v {
						t.Errorf("index %d: got %q, want %q", i, vals[i], v)
					}
				}
				got, _ := s.LRange("mylist", 0, -1)
				if len(got) != len(tt.wantList) {
					t.Fatalf("remaining list len = %d, want %d; got %v", len(got), len(tt.wantList), got)
				}
				for i, v := range tt.wantList {
					if got[i] != v {
						t.Errorf("remaining[%d]: got %q, want %q", i, got[i], v)
					}
				}
			}
		})
	}
}

func TestRPopCount(t *testing.T) {
	tests := []struct {
		name     string
		setup    []string
		count    int
		wantVals []string
		wantOK   bool
		wantList []string
	}{
		{
			name:   "missing key returns not ok",
			count:  2,
			wantOK: false,
		},
		{
			name:     "pops requested count from back",
			setup:    []string{"a", "b", "c", "d"},
			count:    2,
			wantVals: []string{"d", "c"},
			wantOK:   true,
			wantList: []string{"a", "b"},
		},
		{
			name:     "count exceeds length returns all elements",
			setup:    []string{"a", "b"},
			count:    10,
			wantVals: []string{"b", "a"},
			wantOK:   true,
			wantList: []string{},
		},
		{
			name:     "count zero returns empty slice",
			setup:    []string{"a", "b"},
			count:    0,
			wantVals: []string{},
			wantOK:   true,
			wantList: []string{"a", "b"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := store.New()
			if tt.setup != nil {
				s.RPush("mylist", tt.setup...)
			}
			vals, ok := s.RPopCount("mylist", tt.count)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok {
				if len(vals) != len(tt.wantVals) {
					t.Fatalf("vals len = %d, want %d; got %v", len(vals), len(tt.wantVals), vals)
				}
				for i, v := range tt.wantVals {
					if vals[i] != v {
						t.Errorf("index %d: got %q, want %q", i, vals[i], v)
					}
				}
				got, _ := s.LRange("mylist", 0, -1)
				if len(got) != len(tt.wantList) {
					t.Fatalf("remaining list len = %d, want %d; got %v", len(got), len(tt.wantList), got)
				}
				for i, v := range tt.wantList {
					if got[i] != v {
						t.Errorf("remaining[%d]: got %q, want %q", i, got[i], v)
					}
				}
			}
		})
	}
}

func TestBLPopWait(t *testing.T) {
	t.Run("returns immediately when list has elements", func(t *testing.T) {
		s := store.New()
		s.RPush("mylist", "a", "b", "c")
		ch, cancel := s.BLPopWait([]string{"mylist"})
		defer cancel()
		select {
		case got := <-ch:
			if got.Key != "mylist" || got.Val != "a" {
				t.Errorf("got {%q %q}, want {mylist a}", got.Key, got.Val)
			}
		default:
			t.Fatal("expected immediate result, channel was empty")
		}
	})

	t.Run("blocks until element is pushed", func(t *testing.T) {
		s := store.New()
		ch, cancel := s.BLPopWait([]string{"mylist"})
		defer cancel()
		go func() {
			time.Sleep(10 * time.Millisecond)
			s.RPush("mylist", "hello")
		}()
		select {
		case got := <-ch:
			if got.Key != "mylist" || got.Val != "hello" {
				t.Errorf("got {%q %q}, want {mylist hello}", got.Key, got.Val)
			}
		case <-time.After(200 * time.Millisecond):
			t.Fatal("timed out waiting for pushed element")
		}
	})

	t.Run("multiple keys returns first key with element", func(t *testing.T) {
		s := store.New()
		ch, cancel := s.BLPopWait([]string{"k1", "k2"})
		defer cancel()
		go func() {
			time.Sleep(10 * time.Millisecond)
			s.RPush("k2", "from-k2")
		}()
		select {
		case got := <-ch:
			if got.Key != "k2" || got.Val != "from-k2" {
				t.Errorf("got {%q %q}, want {k2 from-k2}", got.Key, got.Val)
			}
		case <-time.After(200 * time.Millisecond):
			t.Fatal("timed out waiting for pushed element")
		}
	})

	t.Run("cancel removes waiter so push does not deliver", func(t *testing.T) {
		s := store.New()
		ch, cancel := s.BLPopWait([]string{"mylist"})
		cancel()
		s.RPush("mylist", "ignored")
		select {
		case got := <-ch:
			t.Errorf("expected no delivery after cancel, got {%q %q}", got.Key, got.Val)
		default:
		}
	})

	t.Run("multiple clients each receive one element", func(t *testing.T) {
		s := store.New()
		ch1, cancel1 := s.BLPopWait([]string{"mylist"})
		ch2, cancel2 := s.BLPopWait([]string{"mylist"})
		defer cancel1()
		defer cancel2()
		s.RPush("mylist", "first", "second")
		got1 := <-ch1
		got2 := <-ch2
		if got1.Val != "first" {
			t.Errorf("client1 got %q, want first", got1.Val)
		}
		if got2.Val != "second" {
			t.Errorf("client2 got %q, want second", got2.Val)
		}
	})
}

func TestType(t *testing.T) {
	tests := []struct {
		name  string
		setup func(s *store.Store)
		key   string
		want  string
	}{
		{
			name:  "missing key returns none",
			setup: func(s *store.Store) {},
			key:   "missing",
			want:  "none",
		},
		{
			name:  "string key returns string",
			setup: func(s *store.Store) { s.Set("k", "v", 0) },
			key:   "k",
			want:  "string",
		},
		{
			name:  "list key returns list",
			setup: func(s *store.Store) { s.RPush("k", "v") },
			key:   "k",
			want:  "list",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := store.New()
			tt.setup(s)
			if got := s.Type(tt.key); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestXAdd(t *testing.T) {
	tests := []struct {
		name   string
		id     string
		fields []string
		wantID string
	}{
		{
			name:   "returns the entry ID",
			id:     "1526919030474-0",
			fields: []string{"temperature", "36", "humidity", "95"},
			wantID: "1526919030474-0",
		},
		{
			name:   "short ID",
			id:     "0-1",
			fields: []string{"foo", "bar"},
			wantID: "0-1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := store.New()
			got, err := s.XAdd("mystream", tt.id, tt.fields)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantID {
				t.Errorf("got %q, want %q", got, tt.wantID)
			}
		})
	}
}

func TestXAddCreatesStreamType(t *testing.T) {
	s := store.New()
	if _, err := s.XAdd("mystream", "0-1", []string{"foo", "bar"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := s.Type("mystream"); got != "stream" {
		t.Errorf("Type = %q, want \"stream\"", got)
	}
}

func TestXAddAppendsMultipleEntries(t *testing.T) {
	s := store.New()
	if _, err := s.XAdd("mystream", "0-1", []string{"foo", "bar"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	id, err := s.XAdd("mystream", "0-2", []string{"baz", "qux"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "0-2" {
		t.Errorf("second XAdd returned %q, want \"0-2\"", id)
	}
	if got := s.Type("mystream"); got != "stream" {
		t.Errorf("Type after second XAdd = %q, want \"stream\"", got)
	}
}

func TestXAddIDValidation(t *testing.T) {
	const errZero = "ERR The ID specified in XADD must be greater than 0-0"
	const errSmall = "ERR The ID specified in XADD is equal or smaller than the target stream top item"

	tests := []struct {
		name    string
		setup   []string // IDs to pre-populate the stream
		id      string
		wantErr string // empty means success expected
	}{
		{
			name:    "0-0 on empty stream",
			id:      "0-0",
			wantErr: errZero,
		},
		{
			name:    "0-0 on non-empty stream",
			setup:   []string{"1-1"},
			id:      "0-0",
			wantErr: errZero,
		},
		{
			name:    "equal to last ID",
			setup:   []string{"1-1"},
			id:      "1-1",
			wantErr: errSmall,
		},
		{
			name:    "ms less than last ms",
			setup:   []string{"1-1"},
			id:      "0-2",
			wantErr: errSmall,
		},
		{
			name:    "ms equal, seq less than last seq",
			setup:   []string{"1-5"},
			id:      "1-4",
			wantErr: errSmall,
		},
		{
			name:  "0-1 on empty stream is valid",
			id:    "0-1",
		},
		{
			name:  "ms greater than last ms",
			setup: []string{"1-1"},
			id:    "2-0",
		},
		{
			name:  "ms equal, seq greater than last seq",
			setup: []string{"1-1"},
			id:    "1-2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := store.New()
			for _, sid := range tt.setup {
				if _, err := s.XAdd("mystream", sid, []string{"k", "v"}); err != nil {
					t.Fatalf("setup XAdd %q failed: %v", sid, err)
				}
			}
			_, err := s.XAdd("mystream", tt.id, []string{"foo", "bar"})
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Errorf("error = %q, want %q", err.Error(), tt.wantErr)
				}
			}
		})
	}
}

func TestXAddPartialAutoID(t *testing.T) {
	tests := []struct {
		name    string
		setup   []string // IDs to pre-populate
		id      string
		wantSeq string
		wantErr bool
	}{
		{
			name:    "ms-* on empty stream gives seq 0",
			id:      "5-*",
			wantSeq: "0",
		},
		{
			name:    "0-* on empty stream gives seq 1 (0-0 is invalid)",
			id:      "0-*",
			wantSeq: "1",
		},
		{
			name:    "ms-* with larger ms gives seq 0",
			setup:   []string{"3-2"},
			id:      "5-*",
			wantSeq: "0",
		},
		{
			name:    "ms-* with equal ms increments seq",
			setup:   []string{"5-4"},
			id:      "5-*",
			wantSeq: "5",
		},
		{
			name:    "ms-* with ms smaller than last returns error",
			setup:   []string{"5-1"},
			id:      "3-*",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := store.New()
			for _, sid := range tt.setup {
				if _, err := s.XAdd("mystream", sid, []string{"k", "v"}); err != nil {
					t.Fatalf("setup XAdd %q failed: %v", sid, err)
				}
			}
			id, err := s.XAdd("mystream", tt.id, []string{"foo", "bar"})
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got id %q", id)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			parts := strings.SplitN(id, "-", 2)
			if len(parts) != 2 || parts[1] != tt.wantSeq {
				t.Errorf("got id %q, want seq %q", id, tt.wantSeq)
			}
		})
	}
}

func TestXAddAutoID(t *testing.T) {
	s := store.New()
	before := time.Now().UnixMilli()
	id, err := s.XAdd("mystream", "*", []string{"field", "value"})
	after := time.Now().UnixMilli()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	parts := strings.SplitN(id, "-", 2)
	if len(parts) != 2 {
		t.Fatalf("expected ms-seq format, got %q", id)
	}
	ms, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		t.Fatalf("ms part %q is not a number: %v", parts[0], err)
	}
	if ms < before || ms > after {
		t.Errorf("ms %d not in range [%d, %d]", ms, before, after)
	}
	if parts[1] != "0" {
		t.Errorf("expected seq 0, got %q", parts[1])
	}
}

func TestXRange(t *testing.T) {
	type entry struct {
		id     string
		fields []string
	}
	tests := []struct {
		name    string
		entries []entry
		start   string
		end     string
		wantIDs []string
	}{
		{
			name:    "missing key returns empty slice",
			start:   "0-1",
			end:     "9-9",
			wantIDs: nil,
		},
		{
			name: "full range returns all entries",
			entries: []entry{
				{"1-0", []string{"k", "v"}},
				{"2-0", []string{"k", "v"}},
				{"3-0", []string{"k", "v"}},
			},
			start:   "1-0",
			end:     "3-0",
			wantIDs: []string{"1-0", "2-0", "3-0"},
		},
		{
			name: "partial start ID defaults seq to 0",
			entries: []entry{
				{"2-0", []string{"k", "v"}},
				{"2-5", []string{"k", "v"}},
				{"3-0", []string{"k", "v"}},
			},
			start:   "2",
			end:     "3-0",
			wantIDs: []string{"2-0", "2-5", "3-0"},
		},
		{
			name: "partial end ID defaults seq to max",
			entries: []entry{
				{"1-0", []string{"k", "v"}},
				{"2-0", []string{"k", "v"}},
				{"2-999", []string{"k", "v"}},
			},
			start:   "1-0",
			end:     "2",
			wantIDs: []string{"1-0", "2-0", "2-999"},
		},
		{
			name: "range excludes entries before start",
			entries: []entry{
				{"1-0", []string{"k", "v"}},
				{"2-0", []string{"k", "v"}},
				{"3-0", []string{"k", "v"}},
			},
			start:   "2-0",
			end:     "9-9",
			wantIDs: []string{"2-0", "3-0"},
		},
		{
			name: "range excludes entries after end",
			entries: []entry{
				{"1-0", []string{"k", "v"}},
				{"2-0", []string{"k", "v"}},
				{"3-0", []string{"k", "v"}},
			},
			start:   "0-1",
			end:     "2-0",
			wantIDs: []string{"1-0", "2-0"},
		},
		{
			name: "boundaries are inclusive",
			entries: []entry{
				{"5-0", []string{"k", "v"}},
				{"6-0", []string{"k", "v"}},
				{"7-0", []string{"k", "v"}},
			},
			start:   "6-0",
			end:     "6-0",
			wantIDs: []string{"6-0"},
		},
		{
			name: "start after end returns empty",
			entries: []entry{
				{"1-0", []string{"k", "v"}},
				{"2-0", []string{"k", "v"}},
			},
			start:   "3-0",
			end:     "1-0",
			wantIDs: nil,
		},
		{
			name: "- as start returns all entries from beginning",
			entries: []entry{
				{"1-0", []string{"k", "v"}},
				{"2-0", []string{"k", "v"}},
				{"3-0", []string{"k", "v"}},
			},
			start:   "-",
			end:     "2-0",
			wantIDs: []string{"1-0", "2-0"},
		},
		{
			name: "+ as end returns all entries to the end",
			entries: []entry{
				{"1-0", []string{"k", "v"}},
				{"2-0", []string{"k", "v"}},
				{"3-0", []string{"k", "v"}},
			},
			start:   "2-0",
			end:     "+",
			wantIDs: []string{"2-0", "3-0"},
		},
		{
			name: "- and + return all entries",
			entries: []entry{
				{"1-0", []string{"k", "v"}},
				{"2-0", []string{"k", "v"}},
				{"3-0", []string{"k", "v"}},
			},
			start:   "-",
			end:     "+",
			wantIDs: []string{"1-0", "2-0", "3-0"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := store.New()
			for _, e := range tt.entries {
				if _, err := s.XAdd("mystream", e.id, e.fields); err != nil {
					t.Fatalf("XAdd %q failed: %v", e.id, err)
				}
			}
			got, err := s.XRange("mystream", tt.start, tt.end)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.wantIDs) {
				t.Fatalf("len = %d, want %d; got %v", len(got), len(tt.wantIDs), got)
			}
			for i, wantID := range tt.wantIDs {
				if got[i].ID != wantID {
					t.Errorf("index %d: got ID %q, want %q", i, got[i].ID, wantID)
				}
			}
		})
	}
}

func TestXRangeReturnsFields(t *testing.T) {
	s := store.New()
	if _, err := s.XAdd("s", "1-0", []string{"temperature", "36", "humidity", "95"}); err != nil {
		t.Fatalf("XAdd failed: %v", err)
	}
	got, err := s.XRange("s", "1-0", "1-0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got))
	}
	wantFields := []string{"temperature", "36", "humidity", "95"}
	if len(got[0].Fields) != len(wantFields) {
		t.Fatalf("fields len = %d, want %d", len(got[0].Fields), len(wantFields))
	}
	for i, f := range wantFields {
		if got[0].Fields[i] != f {
			t.Errorf("field[%d]: got %q, want %q", i, got[0].Fields[i], f)
		}
	}
}

func TestTypeExpiredKeyReturnsNone(t *testing.T) {
	s := store.New()
	s.Set("k", "v", 20*time.Millisecond)
	time.Sleep(30 * time.Millisecond)
	if got := s.Type("k"); got != "none" {
		t.Errorf("got %q, want \"none\"", got)
	}
}

func TestLRange(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		start  int
		stop   int
		wantOK bool
		want   []string
	}{
		{
			name:   "elements in range",
			values: []string{"a", "b", "c", "d", "e"},
			start:  1,
			stop:   3,
			wantOK: true,
			want:   []string{"b", "c", "d"},
		},
		{
			name:   "negative indices",
			values: []string{"a", "b", "c", "d", "e"},
			start:  -3,
			stop:   -1,
			wantOK: true,
			want:   []string{"c", "d", "e"},
		},
		{
			name:   "out of bounds indices clamp to list",
			values: []string{"a", "b", "c"},
			start:  -10,
			stop:   10,
			wantOK: true,
			want:   []string{"a", "b", "c"},
		},
		{
			name:   "start greater than stop returns empty",
			values: []string{"a", "b", "c"},
			start:  2,
			stop:   1,
			wantOK: true,
			want:   []string{},
		},
		{
			name:   "missing key returns not ok",
			start:  0,
			stop:   -1,
			wantOK: false,
			want:   nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := store.New()
			if tt.values != nil {
				s.RPush("mylist", tt.values...)
			}
			vals, ok := s.LRange("mylist", tt.start, tt.stop)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if len(vals) != len(tt.want) {
				t.Fatalf("len = %d, want %d; got %v", len(vals), len(tt.want), vals)
			}
			for i, v := range tt.want {
				if vals[i] != v {
					t.Errorf("index %d: got %q, want %q", i, vals[i], v)
				}
			}
		})
	}
}

func TestXRead(t *testing.T) {
	type entry struct {
		id     string
		fields []string
	}
	tests := []struct {
		name    string
		entries []entry
		afterID string
		wantIDs []string
	}{
		{
			name:    "missing key returns empty slice",
			afterID: "0-0",
			wantIDs: nil,
		},
		{
			name: "returns all entries strictly after given ID",
			entries: []entry{
				{"1-0", []string{"k", "v"}},
				{"2-0", []string{"k", "v"}},
				{"3-0", []string{"k", "v"}},
			},
			afterID: "1-0",
			wantIDs: []string{"2-0", "3-0"},
		},
		{
			name: "ID 0-0 returns all entries",
			entries: []entry{
				{"1-0", []string{"k", "v"}},
				{"2-0", []string{"k", "v"}},
			},
			afterID: "0-0",
			wantIDs: []string{"1-0", "2-0"},
		},
		{
			name: "ID equal to last entry returns empty",
			entries: []entry{
				{"1-0", []string{"k", "v"}},
				{"2-0", []string{"k", "v"}},
			},
			afterID: "2-0",
			wantIDs: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := store.New()
			for _, e := range tt.entries {
				if _, err := s.XAdd("mystream", e.id, e.fields); err != nil {
					t.Fatalf("XAdd %q failed: %v", e.id, err)
				}
			}
			got, err := s.XRead("mystream", tt.afterID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.wantIDs) {
				t.Fatalf("len = %d, want %d; got %v", len(got), len(tt.wantIDs), got)
			}
			for i, wantID := range tt.wantIDs {
				if got[i].ID != wantID {
					t.Errorf("index %d: got ID %q, want %q", i, got[i].ID, wantID)
				}
			}
		})
	}
}
