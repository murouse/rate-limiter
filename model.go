package ratelimiter

import (
	"fmt"
	"time"

	"github.com/samber/lo"

	pb "github.com/murouse/rate-limiter/pkg/api/murouse/rate_limiter/v1"
)

// Rule describes a single fixed-window rate limiting rule.
type Rule struct {
	Name   string
	Limit  int
	Window time.Duration
	// Shared — one counter for the whole method, regardless of rate key attributes and extension
	Shared bool
}

// RateLimitRulesToModel converts protobuf Rule definitions
// into internal Rule models used by the rate limiter.
func RateLimitRulesToModel(rs []*pb.Rule) []Rule {
	return lo.Map(rs, func(r *pb.Rule, _ int) Rule {
		return Rule{
			Name:   r.Name,
			Limit:  int(r.Limit),
			Window: r.Window.AsDuration(),
			Shared: r.Shared,
		}.withName()
	})
}

// withName fills an empty Name from the rule itself: "<limit>_per_<window>", plus "_shared" for a shared rule.
// Rules with the same limit, window and scope get the same name — and the same counter, which is what they are
func (r Rule) withName() Rule {
	if r.Name != "" {
		return r
	}

	r.Name = fmt.Sprintf("%d_per_%s", r.Limit, formatWindow(r.Window))
	if r.Shared {
		r.Name += "_shared"
	}

	return r
}

// formatWindow — the largest whole unit: 1d, 12h, 1m, 30s (time.Duration.String gives "24h0m0s")
func formatWindow(w time.Duration) string {
	const day = 24 * time.Hour

	switch {
	case w > 0 && w%day == 0:
		return fmt.Sprintf("%dd", w/day)
	case w > 0 && w%time.Hour == 0:
		return fmt.Sprintf("%dh", w/time.Hour)
	case w > 0 && w%time.Minute == 0:
		return fmt.Sprintf("%dm", w/time.Minute)
	case w > 0 && w%time.Second == 0:
		return fmt.Sprintf("%ds", w/time.Second)
	default:
		return w.String()
	}
}
