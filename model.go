package ratelimiter

import "time"

// RateLimitRule describes a single fixed-window rate limiting rule.
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
type RateLimitRule struct {
	Name   string
	Limit  int
	Window time.Duration
}
