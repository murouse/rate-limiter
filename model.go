package ratelimiter

import (
	"time"

	ratelimiterpb "github.com/murouse/rate-limiter/github.com/murouse/rate-limiter"
	"github.com/samber/lo"
)

// Rule describes a single fixed-window rate limiting rule.
//
// Fields:
//
//   - Name: logical rule identifier (used in storage key)
//   - Limit: maximum number of requests allowed
//   - Window: duration of the fixed window
//
// Example:
//
//	10 requests per minute:
//	  Name   = "per_minute"
//	  Limit  = 10
//	  Window = time.Minute
//
// Multiple rules may be attached to a single RPC method.
type Rule struct {
	Name   string
	Limit  int
	Window time.Duration
}

func RateLimitRulesToModel(rs []*ratelimiterpb.Rule) []Rule {
	return lo.Map(rs, func(r *ratelimiterpb.Rule, _ int) Rule {
		return Rule{
			Name:   r.Name,
			Limit:  int(r.Limit),
			Window: r.Window.AsDuration(),
		}
	})
}
