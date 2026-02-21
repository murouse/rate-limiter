package ratelimiter

import (
	"time"

	"github.com/samber/lo"

	ratelimiterpb "github.com/murouse/rate-limiter/github.com/murouse/rate-limiter"
)

// Rule describes a single fixed-window rate limiting rule.
type Rule struct {
	Name   string
	Limit  int
	Window time.Duration
}

// RateLimitRulesToModel converts protobuf Rule definitions
// into internal Rule models used by the rate limiter.
func RateLimitRulesToModel(rs []*ratelimiterpb.Rule) []Rule {
	return lo.Map(rs, func(r *ratelimiterpb.Rule, _ int) Rule {
		return Rule{
			Name:   r.Name,
			Limit:  int(r.Limit),
			Window: r.Window.AsDuration(),
		}
	})
}
