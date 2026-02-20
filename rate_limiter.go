package ratelimiter

import (
	"context"
	"fmt"
	"sync"

	"github.com/murouse/rate-limiter/internal/cache"
	"github.com/murouse/rate-limiter/internal/extractor"
)

// RateLimiter implements a fixed-window rate limiting middleware for gRPC.
// The limiter enforces fixed-window semantics.
type RateLimiter struct {
	extractor        RateKeyExtractor
	cache            Cache
	namespace        string
	globalLimitRules []Rule
	keyFormatter     keyFormatterFunc
	logger           Logger

	methodRules     map[string][]Rule
	methodRulesOnce sync.Once
}

func New(opts ...Option) *RateLimiter {
	rl := &RateLimiter{
		cache:            cache.NewInMemoryCache(),
		extractor:        extractor.NewDefaultExtractor(),
		namespace:        "default",
		globalLimitRules: nil,
		keyFormatter:     defaultKeyFormatter,
		logger:           &noopLogger{},
	}

	for _, opt := range opts {
		opt(rl)
	}

	return rl
}

func (rl *RateLimiter) allow(ctx context.Context, rateKey, fullMethod string, methodRules []Rule) ([]Rule, error) {
	var exceededRules []Rule

	for _, globalRule := range rl.globalLimitRules {
		ok, err := rl.checkRule(ctx, rateKey, "global", "global", globalRule)
		if err != nil {
			return nil, fmt.Errorf("failed to check global rule: %w", err)
		}
		if !ok {
			exceededRules = append(exceededRules, globalRule)
		}
	}

	for _, methodRule := range methodRules {
		ok, err := rl.checkRule(ctx, rateKey, fullMethod, methodRule.Name, methodRule)
		if err != nil {
			return nil, fmt.Errorf("failed to check method rule: %w", err)
		}
		if !ok {
			exceededRules = append(exceededRules, methodRule)
		}
	}

	return exceededRules, nil
}

func (rl *RateLimiter) checkRule(ctx context.Context, rateKey, fullMethod, ruleName string, rule Rule) (bool, error) {
	fullKey := rl.keyFormatter(rl.namespace, rateKey, fullMethod, ruleName)

	count, err := rl.cache.Increment(ctx, fullKey, rule.Window)
	if err != nil {
		rl.logger.Errorf("increment failed for key %q: %v", fullKey, err)
		return false, fmt.Errorf("increment: %w", err)
	}

	if count > int64(rule.Limit) {
		return false, nil
	}

	return true, nil
}
