package ratelimiter

import (
	"context"
	"fmt"
	"time"

	"github.com/murouse/rate-limiter/internal/cache"
	"github.com/murouse/rate-limiter/internal/extractor"
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

// RateLimiter implements a fixed-window rate limiting middleware for gRPC.
//
// It supports:
//
//   - Per-method limits defined via protobuf options
//   - Global limits applied to all methods
//   - Pluggable storage backend (Cache)
//   - Custom key extraction
//   - Custom key formatting
//
// The limiter enforces fixed-window semantics.
type RateLimiter struct {
	extractor        RateKeyExtractor
	cache            Cache
	namespace        string
	globalLimitRules []RateLimitRule
	keyFormatter     keyFormatterFunc
}

// New creates a new RateLimiter with default configuration.
//
// Defaults:
//   - In-memory cache
//   - Default key extractor
//   - Namespace = "default"
//   - No global limits
//   - Default key formatter
//
// Options can override any of these.
func New(opts ...Option) *RateLimiter {
	rl := &RateLimiter{
		cache:            cache.New(),
		extractor:        extractor.New(),
		namespace:        "default",
		globalLimitRules: nil,
		keyFormatter:     defaultKeyFormatter,
	}

	for _, opt := range opts {
		opt(rl)
	}

	return rl
}

// allow evaluates global and method-specific rules for a given request.
//
// Returns:
//
//   - true if all rules pass
//   - false if any rule is exceeded
//   - error if storage fails
//
// Evaluation order:
//  1. Global rules
//  2. Method rules
//
// Stops on first violation.
func (rl *RateLimiter) allow(ctx context.Context, rateKey, fullMethod string, methodRules []RateLimitRule) (bool, error) {
	// итерируемся по глобальным правилам
	for _, globalRule := range rl.globalLimitRules {
		ok, err := rl.checkRule(ctx, rateKey, "global", "global", globalRule)
		if err != nil {
			return false, fmt.Errorf("failed to check global rule: %w", err)
		}
		if !ok {
			return false, nil
		}
	}

	// итерируемся по правилам метода
	for _, methodRule := range methodRules {
		ok, err := rl.checkRule(ctx, rateKey, fullMethod, methodRule.Name, methodRule)
		if err != nil {
			return false, fmt.Errorf("failed to check method rule: %w", err)
		}
		if !ok {
			return false, nil
		}
	}

	return true, nil
}

// checkRule evaluates a single rate limit rule using fixed-window semantics.
//
// A unique storage key is built using namespace, rate key,
// method name and rule name.
//
// Returns false if the rule limit is exceeded.
func (rl *RateLimiter) checkRule(ctx context.Context, rateKey, fullMethod, ruleName string, rule RateLimitRule) (bool, error) {
	fullKey := rl.keyFormatter(rl.namespace, rateKey, fullMethod, ruleName)

	count, err := rl.cache.Increment(ctx, fullKey, rule.Window)
	if err != nil {
		return false, fmt.Errorf("increment: %w", err)
	}

	if count > int64(rule.Limit) {
		return false, nil
	}

	return true, nil
}
