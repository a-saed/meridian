package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	lru "github.com/hashicorp/golang-lru/v2"
)

// Cache is a thread-safe LRU cache for rendered map images (PNG bytes).
type Cache struct {
	lru *lru.Cache[string, []byte]
}

// New returns a Cache with the given maximum number of entries.
func New(size int) (*Cache, error) {
	l, err := lru.New[string, []byte](size)
	if err != nil {
		return nil, fmt.Errorf("cache: init: %w", err)
	}
	return &Cache{lru: l}, nil
}

// Get retrieves a cached image by key. Returns (nil, false) on miss.
func (c *Cache) Get(key string) ([]byte, bool) {
	return c.lru.Get(key)
}

// Set stores image bytes under key, evicting the LRU entry if at capacity.
func (c *Cache) Set(key string, val []byte) {
	c.lru.Add(key, val)
}

// Key generates a stable cache key from WMS request parameters.
func Key(params map[string]string) string {
	h := sha256.New()
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(h, "%s=%s&", k, params[k])
	}
	return hex.EncodeToString(h.Sum(nil))
}
