package cache

//func NewRedisCache(client *redis.Client) *RedisCache {
//	script := redis.NewScript(`
//        local current = redis.call("INCR", KEYS[1])
//        if current == 1 then
//            redis.call("PEXPIRE", KEYS[1], ARGV[1])
//        end
//        return current
//    `)
//
//	return &RedisCache{
//		client: client,
//		script: script,
//	}
//}
//
//func (r *RedisCache) Increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
//	res, err := r.script.Run(
//		ctx,
//		r.client,
//		[]string{key},
//		ttl.Milliseconds(),
//	).Result()
//
//	if err != nil {
//		return 0, err
//	}
//
//	count, ok := res.(int64)
//	if !ok {
//		return 0, fmt.Errorf("unexpected type %T", res)
//	}
//
//	return count, nil
//}
