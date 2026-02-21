package ratelimiter

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/samber/lo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Option configures RateLimiter.
type Option func(*RateLimiter)

// WithCache sets a custom storage backend.
//
// The provided cache must implement atomic fixed-window semantics.
func WithCache(cache Cache) Option {
	return func(rl *RateLimiter) {
		rl.cache = cache
	}
}

// WithRateKeyExtender overrides the default rate key extension logic.
//
// It allows adding custom identifiers (e.g. user ID) to the rate key.
func WithRateKeyExtender(rateKeyExtender rateKeyExtenderFunc) Option {
	return func(rl *RateLimiter) {
		rl.rateKeyExtender = rateKeyExtender
	}
}

// WithNamespace sets a namespace prefix for all generated storage keys.
//
// Useful when sharing the same cache across multiple services.
func WithNamespace(namespace string) Option {
	return func(rl *RateLimiter) {
		rl.namespace = namespace
	}
}

// WithGlobalLimitRules configures rules that apply to all RPC methods.
func WithGlobalLimitRules(rules []Rule) Option {
	return func(rl *RateLimiter) {
		rl.globalLimitRules = rules
	}
}

// WithRateKeyFormatter overrides the storage key formatting logic.
//
// Intended for advanced customization of key structure.
func WithRateKeyFormatter(rateKeyFormatter rateKeyFormatterFunc) Option {
	return func(rl *RateLimiter) {
		rl.rateKeyFormatter = rateKeyFormatter
	}
}

// WithLogger sets a custom logger implementation.
func WithLogger(logger Logger) Option {
	return func(rl *RateLimiter) {
		rl.logger = logger
	}
}

// WithExceedErrorFormatter overrides the error returned
// when one or more rate limit rules are exceeded.
func WithExceedErrorFormatter(exceedErrorFormatter exceedErrorFormatterFunc) Option {
	return func(rl *RateLimiter) {
		rl.exceedErrorFormatter = exceedErrorFormatter
	}
}

type rateKeyFormatterFunc func(namespace, rateKeyExtension, fullMethod, ruleName string, attrs map[string]string) string

// defaultRateKeyFormatter builds a deterministic storage key
// composed of namespace, rate key extension, full method name,
// rule name, and sorted rate key attributes.
func defaultRateKeyFormatter(namespace, rateKeyExtension, fullMethod, ruleName string, attrs map[string]string) string {
	// Сортируем ключи attrs для детерминированного порядка
	keys := lo.Keys(attrs)
	slices.Sort(keys)

	// Строим часть ключа из attrs: key1=val1,key2=val2,...
	attrParts := lo.Map(keys, func(k string, _ int) string {
		return fmt.Sprintf("%s=%s", k, attrs[k])
	})
	attrStr := strings.Join(attrParts, ",")

	// Собираем финальный ключ
	if attrStr != "" {
		return fmt.Sprintf("rate-limiter:%s:%s:%s:%s:%s", namespace, rateKeyExtension, fullMethod, ruleName, attrStr)
	}

	return fmt.Sprintf("rate-limiter:%s:%s:%s:%s", namespace, rateKeyExtension, fullMethod, ruleName)
}

type exceedErrorFormatterFunc func(exceededRules []Rule) error

// defaultExceedErrorFormatter returns a ResourceExhausted gRPC error
// containing the names of exceeded rules.
func defaultExceedErrorFormatter(exceededRules []Rule) error {
	msg := strings.Join(lo.Map(exceededRules, func(exceededRule Rule, _ int) string {
		return exceededRule.Name
	}), ", ")

	return status.Errorf(codes.ResourceExhausted, "rate limit exceeded: %s", msg)
}

type rateKeyExtenderFunc func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo) (string, error)

// defaultRateKeyExtender returns a static rate key extension.
//
// Applications should override this to provide meaningful identifiers
// such as user ID or API key.
func defaultRateKeyExtender(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo) (string, error) {
	return "rate-key-extension", nil
}
