package ratelimiter

import (
	"log/slog"
	"sync"

	"github.com/murouse/rate-limiter/internal/cache"
)

// RateLimiter implements a fixed-window rate limiting middleware for gRPC.
// The limiter enforces fixed-window semantics.
type RateLimiter struct {
	cache                Cache
	namespace            string
	globalLimitRules     []Rule
	rateKeyExtender      rateKeyExtenderFunc
	rateKeyFormatter     rateKeyFormatterFunc
	exceedErrorFormatter exceedErrorFormatterFunc
	logger               *slog.Logger

	methodRules     map[string][]Rule
	methodRulesOnce sync.Once
}

// New creates a new RateLimiter with default configuration.
//
// By default, it uses an in-memory cache, no-op logger,
// default namespace, and standard key formatting behavior.
func New(opts ...Option) *RateLimiter {
	rl := &RateLimiter{
		cache:                cache.NewInMemoryCache(),
		namespace:            "default",
		globalLimitRules:     nil,
		rateKeyExtender:      defaultRateKeyExtender,
		rateKeyFormatter:     defaultRateKeyFormatter,
		exceedErrorFormatter: defaultExceedErrorFormatter,
		logger:               slog.New(slog.DiscardHandler),
	}

	for _, opt := range opts {
		opt(rl)
	}

	return rl
}
