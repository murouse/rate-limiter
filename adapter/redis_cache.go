package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCacheAdapter struct {
	client redis.Scripter
}

func NewRedisCache(client redis.Scripter) *RedisCacheAdapter {
	return &RedisCacheAdapter{client: client}
}

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
