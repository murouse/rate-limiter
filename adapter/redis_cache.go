package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCacheAdapter implements the Cache interface using Redis.
//
// It relies on a Lua script to guarantee atomic increment
// and fixed-window TTL semantics.
type RedisCacheAdapter struct {
	client redis.Scripter
}

// NewRedisCache creates a Redis-backed Cache implementation.
//
// The provided client must support script execution (redis.Scripter).
func NewRedisCache(client redis.Scripter) *RedisCacheAdapter {
	return &RedisCacheAdapter{client: client}
}

// Increment atomically increments the counter for the given key.
//
// TTL is set only when the key is created (first increment)
// and is not extended on subsequent calls, ensuring fixed-window behavior.
func (c *RedisCacheAdapter) Increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	script := redis.NewScript(`
       local current = redis.call("INCR", KEYS[1])
       if current == 1 then
           redis.call("PEXPIRE", KEYS[1], ARGV[1])
       end
       return current
   `)

	res, err := script.Run(
		ctx,
		c.client,
		[]string{key},
		ttl.Milliseconds(),
	).Result()
	if err != nil {
		return 0, err
	}

	count, ok := res.(int64)
	if !ok {
		return 0, fmt.Errorf("unexpected type %T", res)
	}

	return count, nil
}
