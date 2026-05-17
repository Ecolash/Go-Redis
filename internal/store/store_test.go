package store_test

import (
	"testing"
	"time"

	"github.com/codecrafters-io/redis-starter-go/internal/store"
)

func TestGetReturnsMissingForUnknownKey(t *testing.T) {
	s := store.New()
	val, ok := s.Get("missing")
	if ok {
		t.Errorf("expected key to be missing, got %q", val)
	}
}

func TestSetThenGetReturnsValue(t *testing.T) {
	s := store.New()
	s.Set("foo", "bar", 0)
	val, ok := s.Get("foo")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if val != "bar" {
		t.Errorf("expected bar, got %q", val)
	}
}

func TestSetOverwritesExistingKey(t *testing.T) {
	s := store.New()
	s.Set("key", "first", 0)
	s.Set("key", "second", 0)
	val, _ := s.Get("key")
	if val != "second" {
		t.Errorf("expected second, got %q", val)
	}
}

func TestGetReturnsValueBeforeTTLExpires(t *testing.T) {
	s := store.New()
	s.Set("foo", "bar", 200 * time.Millisecond)
	val, ok := s.Get("foo")
	if !ok {
		t.Fatal("expected key to exist before TTL expires")
	}
	if val != "bar" {
		t.Errorf("expected bar, got %q", val)
	}
}

func TestGetReturnsMissingAfterTTLExpires(t *testing.T) {
	s := store.New()
	s.Set("foo", "bar", 20 * time.Millisecond)
	_, ok := s.Get("foo") // Access before expiration to ensure it exists
	if !ok {
		t.Fatal("expected key to exist before TTL expires")
	}
	time.Sleep(30 * time.Millisecond)
	_, ok = s.Get("foo")
	if ok {
		t.Error("expected key to be expired and missing")
	}
}

func TestRPushCreatesListAndReturnsOne(t *testing.T) {
	s := store.New()
	n := s.RPush("mylist", "foo")
	if n != 1 {
		t.Errorf("expected 1, got %d", n)
	}
}

func TestRPushAppendsToExistingListAndReturnsCount(t *testing.T) {
	s := store.New()
	s.RPush("mylist", "foo")
	n := s.RPush("mylist", "bar")
	if n != 2 {
		t.Errorf("expected 2, got %d", n)
	}
}

func TestRPushWithMultipleValuesReturnsTotal(t *testing.T) {
	s := store.New()
	n := s.RPush("mylist", "a", "b", "c")
	if n != 3 {
		t.Errorf("expected 3, got %d", n)
	}
}
