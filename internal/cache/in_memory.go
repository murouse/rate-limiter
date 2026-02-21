package cache

import (
	"context"
	"sync"
	"time"
)

// InMemoryCache is an in-memory Cache implementation.
//
// It provides atomic fixed-window semantics using a mutex.
// Intended primarily for testing or single-instance deployments.
type InMemoryCache struct {
	mu     sync.Mutex
	counts map[string]int64
	ttl    map[string]time.Time
}

// NewInMemoryCache creates a new in-memory cache instance.
func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{
		counts: make(map[string]int64),
		ttl:    make(map[string]time.Time),
	}
}

// Increment atomically increments the counter for the given key.
//
// If the key is new or expired, the counter starts from 1
// and the TTL is set to now + window.
// Otherwise, the counter is incremented without modifying TTL.
func (c *InMemoryCache) Increment(_ context.Context, key string, window time.Duration) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	// Проверяем истечение TTL
	if expireAt, ok := c.ttl[key]; ok {
		if now.After(expireAt) {
			delete(c.counts, key)
			delete(c.ttl, key)
		}
	}

	// Если ключ новый (или был удалён после expiration)
	if _, exists := c.counts[key]; !exists {
		c.counts[key] = 1
		if window > 0 {
			c.ttl[key] = now.Add(window)
		}

		return 1, nil
	}

	// Иначе просто увеличиваем счётчик
	c.counts[key]++

	return c.counts[key], nil
}
