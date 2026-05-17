package store_test

import (
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
