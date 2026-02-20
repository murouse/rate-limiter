package ratelimiter

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

// Cache defines storage behavior for fixed-window rate limiting.
//
// Implementations MUST provide atomic increment semantics.
//
// Increment atomically increments the counter for the given key and
// MUST ensure that:
//
//  1. The TTL is set only if the key did not previously exist
//     (i.e. on the first increment).
//  2. The TTL MUST NOT be extended or modified on subsequent increments.
//
// This behavior guarantees fixed-window semantics.
//
// If TTL is extended on every increment, the algorithm becomes a sliding window,
// which is NOT the intended behavior of this interface.
//
// Implementations should ensure atomicity (e.g. Redis Lua script).
type Cache interface {
	Increment(ctx context.Context, key string, ttl time.Duration) (int64, error)
}

// RateKeyExtractor extracts a rate limit key from an incoming gRPC request.
//
// Implementations may derive the key from:
//
//   - authenticated user ID
//   - tenant ID
//   - IP address
//   - request metadata
//
// The extracted key is used as a part of the rate limiting storage key.
type RateKeyExtractor interface {
	ExtractRateKey(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo) (string, error)
}

type Logger interface {
	Debugf(msg string, args ...any)
	Infof(msg string, args ...any)
	Warnf(msg string, args ...any)
	Errorf(msg string, args ...any)
}
