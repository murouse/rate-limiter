package ratelimiter

import (
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
	logger               Logger

	methodRules     map[string][]Rule
	methodRulesOnce sync.Once
}

func New(opts ...Option) *RateLimiter {
	rl := &RateLimiter{
		cache:                cache.NewInMemoryCache(),
		namespace:            "default",
		globalLimitRules:     nil,
		rateKeyExtender:      defaultRateKeyExtender,
		rateKeyFormatter:     defaultRateKeyFormatter,
		exceedErrorFormatter: defaultExceedErrorFormatter,
		logger:               &noopLogger{},
	}

	for _, opt := range opts {
		opt(rl)
	}

	return rl
}
