package store_test

import (
	"testing"

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
	s.Set("foo", "bar")
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
	s.Set("key", "first")
	s.Set("key", "second")
	val, _ := s.Get("key")
	if val != "second" {
		t.Errorf("expected second, got %q", val)
	}
}
