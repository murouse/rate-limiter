package ratelimiter

import (
	"sync"

	"github.com/murouse/rate-limiter/internal/cache"
	"github.com/murouse/rate-limiter/internal/extender"
)

// RateLimiter implements a fixed-window rate limiting middleware for gRPC.
// The limiter enforces fixed-window semantics.
type RateLimiter struct {
	rateKeyExtender      RateKeyExtender
	cache                Cache
	namespace            string
	globalLimitRules     []Rule
	rateKeyFormatter     rateKeyFormatterFunc
	exceedErrorFormatter exceedErrorFormatterFunc
	logger               Logger

	methodRules     map[string][]Rule
	methodRulesOnce sync.Once
}

func New(opts ...Option) *RateLimiter {
	rl := &RateLimiter{
		cache:                cache.NewInMemoryCache(),
		rateKeyExtender:      extender.NewDefaultRateKeyExtender(),
		namespace:            "default",
		globalLimitRules:     nil,
		rateKeyFormatter:     defaultRateKeyFormatter,
		exceedErrorFormatter: defaultExceedErrorFormatter,
		logger:               &noopLogger{},
	}

	for _, opt := range opts {
		opt(rl)
	}

	return rl
}
