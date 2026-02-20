package ratelimiter

import (
	"sync"

	"github.com/murouse/rate-limiter/internal/cache"
	"github.com/murouse/rate-limiter/internal/extractor"
)

// RateLimiter implements a fixed-window rate limiting middleware for gRPC.
// The limiter enforces fixed-window semantics.
type RateLimiter struct {
	extractor            RateKeyExtractor
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
		extractor:            extractor.NewDefaultExtractor(),
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
