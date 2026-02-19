package cache

import (
	"context"
	"sync"
	"time"
)

// InMemoryCache implements fixed-window semantics.
// TTL is set only on the first increment of a key
// and is NOT extended on subsequent increments.
type InMemoryCache struct {
	mu     sync.Mutex
	counts map[string]int64
	ttl    map[string]time.Time
}

func New() *InMemoryCache {
	return &InMemoryCache{
		counts: make(map[string]int64),
		ttl:    make(map[string]time.Time),
	}
}

// Increment atomically increments the counter for key.
//
// Semantics:
//   - If the key is expired or does not exist → counter starts from 1
//     and TTL is set to now + ttl.
//   - If the key exists and is not expired → counter increments
//     and TTL is NOT modified.
//
// This guarantees fixed-window behavior.
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
