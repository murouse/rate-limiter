package ratelimiter

import (
	"fmt"
)

// Option configures RateLimiter.
type Option func(*RateLimiter)

// WithCache overrides the default storage backend.
//
// The cache MUST implement fixed-window semantics.
func WithCache(cache Cache) Option {
	return func(rl *RateLimiter) {
		rl.cache = cache
	}
}

// WithRateKeyExtractor sets a custom rate key extractor.
func WithRateKeyExtractor(extractor RateKeyExtractor) Option {
	return func(rl *RateLimiter) {
		rl.extractor = extractor
	}
}

// WithNamespace sets a namespace prefix for all storage keys.
//
// Useful when sharing the same storage across multiple services.
func WithNamespace(namespace string) Option {
	return func(rl *RateLimiter) {
		rl.namespace = namespace
	}
}

// WithGlobalLimitRules sets rules applied to all RPC methods.
func WithGlobalLimitRules(rules []Rule) Option {
	return func(rl *RateLimiter) {
		rl.globalLimitRules = rules
	}
}

// WithKeyFormatterFunc overrides the storage key formatting logic.
//
// Advanced usage only.
func WithKeyFormatterFunc(keyFormatter keyFormatterFunc) Option {
	return func(rl *RateLimiter) {
		rl.keyFormatter = keyFormatter
	}
}

func WithLogger(logger Logger) Option {
	return func(rl *RateLimiter) {
		rl.logger = logger
	}
}

// keyFormatterFunc builds a unique storage key.
//
// Parameters:
//   - namespace
//   - rateKey (e.g. user ID)
//   - fullMethod (gRPC method name)
//   - ruleName
type keyFormatterFunc func(namespace, rateKey, fullMethod, ruleName string) string

// defaultKeyFormatter builds a colon-separated storage key.
//
// Example:
//
//	rate-limiter:default:user123:/svc.Method:per_minute
func defaultKeyFormatter(namespace, rateKey, fullMethod, ruleName string) string {
	return fmt.Sprintf("rate-limiter:%s:%s:%s:%s", namespace, rateKey, fullMethod, ruleName)
}
