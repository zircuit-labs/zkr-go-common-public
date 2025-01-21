package cache

import (
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

// Memory is an in-memory cache implementation using an expirable LRU cache from HashiCorp.
// It provides thread-safe methods to store and retrieve data with optional expiration.
type (
	Memory struct {
		cache *expirable.LRU[string, []byte]
	}
)

// NewMemory creates and returns a new Memory instance with the specified maximum size and TTL.
func NewMemory(maxSize int, ttl time.Duration) *Memory {
	return &Memory{
		cache: expirable.NewLRU[string, []byte](maxSize, nil, ttl),
	}
}

// Get retrieves the content associated with the given key from the cache.
// If the key is not found or the item has expired, it returns nil and false.
func (m *Memory) Get(key string) ([]byte, bool) {
	return m.cache.Get(key)
}

// Set stores the content in the cache with the specified key and duration.
// The item will expire after the given duration.
func (m *Memory) Set(key string, content []byte) {
	m.cache.Add(key, content)
}
