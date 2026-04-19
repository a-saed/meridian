package cache_test

import (
	"testing"

	"meridian/internal/cache"
)

func TestSetAndGet(t *testing.T) {
	c, err := cache.New(10)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	c.Set("key1", []byte("hello"))

	val, ok := c.Get("key1")
	if !ok {
		t.Fatal("expected hit on key1")
	}
	if string(val) != "hello" {
		t.Errorf("want 'hello', got %s", val)
	}
}

func TestMiss(t *testing.T) {
	c, _ := cache.New(10)
	_, ok := c.Get("missing")
	if ok {
		t.Fatal("expected miss")
	}
}

func TestEviction(t *testing.T) {
	c, _ := cache.New(2) // capacity 2

	c.Set("a", []byte("1"))
	c.Set("b", []byte("2"))
	c.Set("c", []byte("3")) // evicts "a" (LRU)

	_, ok := c.Get("a")
	if ok {
		t.Fatal("expected 'a' to be evicted")
	}
	_, ok = c.Get("c")
	if !ok {
		t.Fatal("expected 'c' to be present")
	}
}
