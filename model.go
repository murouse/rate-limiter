package ratelimiter

import (
	"time"

	"github.com/samber/lo"

	pb "github.com/murouse/rate-limiter/pkg/api/murouse/rate_limiter/v1"
)

// Rule describes a single fixed-window rate limiting rule.
type Rule struct {
	Name   string
	Limit  int
	Window time.Duration
}

// RateLimitRulesToModel converts protobuf Rule definitions
// into internal Rule models used by the rate limiter.
func RateLimitRulesToModel(rs []*pb.Rule) []Rule {
	return lo.Map(rs, func(r *pb.Rule, _ int) Rule {
		return Rule{
			Name:   r.Name,
			Limit:  int(r.Limit),
			Window: r.Window.AsDuration(),
		}
	})
}
