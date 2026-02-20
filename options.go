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

// WithCache overrides the default storage backend.
//
// The cache MUST implement fixed-window semantics.
func WithCache(cache Cache) Option {
	return func(rl *RateLimiter) {
		rl.cache = cache
	}
}

// WithRateKeyExtender sets a custom rate key rateKeyExtender.
func WithRateKeyExtender(rateKeyExtender rateKeyExtenderFunc) Option {
	return func(rl *RateLimiter) {
		rl.rateKeyExtender = rateKeyExtender
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

// WithRateKeyFormatter overrides the storage key formatting logic.
//
// Advanced usage only.
func WithRateKeyFormatter(rateKeyFormatter rateKeyFormatterFunc) Option {
	return func(rl *RateLimiter) {
		rl.rateKeyFormatter = rateKeyFormatter
	}
}

func WithLogger(logger Logger) Option {
	return func(rl *RateLimiter) {
		rl.logger = logger
	}
}

func WithExceedErrorFormatter(exceedErrorFormatter exceedErrorFormatterFunc) Option {
	return func(rl *RateLimiter) {
		rl.exceedErrorFormatter = exceedErrorFormatter
	}
}

// rateKeyFormatterFunc builds a unique storage key.
//
// Parameters:
//   - namespace
//   - rateKey (e.g. user ID)
//   - fullMethod (gRPC method name)
//   - ruleName
type rateKeyFormatterFunc func(namespace, rateKeyExtension, fullMethod, ruleName string, attrs map[string]string) string

// defaultRateKeyFormatter builds a colon-separated storage key.
//
// Example:
//
//	rate-limiter:default:user123:/svc.Method:per_minute
func defaultRateKeyFormatter(namespace, rateKeyExtension, fullMethod, ruleName string, attrs map[string]string) string {
	// Сортируем ключи attrs для детерминированного порядка
	keys := lo.Keys(attrs)
	slices.Sort(keys)

	// Строим часть ключа из attrs: key1=val1,key2=val2,...
	var attrParts []string
	for _, k := range keys {
		attrParts = append(attrParts, fmt.Sprintf("%s=%s", k, attrs[k]))
	}
	attrStr := strings.Join(attrParts, ",")

	// Собираем финальный ключ
	if attrStr != "" {
		return fmt.Sprintf("rate-limiter:%s:%s:%s:%s:%s", namespace, rateKeyExtension, fullMethod, ruleName, attrStr)
	}
	return fmt.Sprintf("rate-limiter:%s:%s:%s:%s", namespace, rateKeyExtension, fullMethod, ruleName)
}

type exceedErrorFormatterFunc func(exceededRules []Rule) error

func defaultExceedErrorFormatter(exceededRules []Rule) error {
	msg := strings.Join(lo.Map(exceededRules, func(exceededRule Rule, _ int) string {
		return exceededRule.Name
	}), ", ")

	return status.Errorf(codes.ResourceExhausted, "rate limit exceeded: %s", msg)
}

type rateKeyExtenderFunc func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo) (string, error)

func defaultRateKeyExtender(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo) (string, error) {
	return "rate-key-extension", nil
}
