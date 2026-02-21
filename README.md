# gRPC Fixed-Window Rate Limiter

Lightweight, protobuf-driven **fixed-window** rate limiter middleware for gRPC.

* ✅ Fixed-window semantics (no sliding window surprises)
* ✅ Rules defined directly in `.proto`
* ✅ Global + per-method limits
* ✅ Redis or in-memory backend
* ✅ Pluggable key strategy and logger

---

## Installation

```bash
go get github.com/murouse/rate-limiter
```

---

## How It Works

The limiter:

1. Extracts `rate_key` attributes from request protobuf messages
2. Builds a deterministic storage key
3. Applies:

    * Global rules
    * Per-method rules (defined in proto)
4. Uses atomic increment with TTL set **only on first request**

> TTL is **not extended** on subsequent increments → strict fixed-window behavior.

---

# Define Rules in Proto

```proto
syntax = "proto3";

package rate_limiter;

option go_package = "github.com/murouse/rate-limiter;rate_limiter";

import "google/protobuf/descriptor.proto";
import "google/protobuf/duration.proto";

message Rule {
  string name = 1;
  int32 limit = 2;
  google.protobuf.Duration window = 3;
}

extend google.protobuf.MethodOptions {
  repeated Rule rules = 51234;
}

extend google.protobuf.FieldOptions {
  string rate_key = 51235;
}
```

---

## Example: Per-Method Rule

```proto
service AuthService {
  rpc SendCode(SendCodeRequest) returns (SendCodeResponse) {
    option (rate_limiter.rules) = {
      name: "per_minute"
      limit: 6
      window: { seconds: 60 }
    };
  }
}

message SendCodeRequest {
  string phone = 1 [(rate_limiter.rate_key) = "phone"];
}
```

### What happens

For `SendCode`:

* Limit: **6 requests per 60 seconds**
* Key will include:

    * namespace
    * custom rate key (e.g. user ID)
    * method name
    * rule name
    * `phone` field value

---

# Usage

## Basic Setup

```go
rateLimiter := ratelimiter.New(
    ratelimiter.WithNamespace("hookah-culture"),

    ratelimiter.WithCache(
        ratelimiteradapter.NewRedisCache(redisClient),
    ),

    ratelimiter.WithGlobalLimitRules([]ratelimiter.Rule{
        {
            Name:   "global",
            Limit:  5,
            Window: time.Minute,
        },
    }),

    ratelimiter.WithRateKeyExtender(
        func(ctx context.Context, _ interface{}, _ *grpc.UnaryServerInfo) (string, error) {
            user, ok := actor.FromContext(ctx)
            if !ok {
                return "", nil
            }
            return strconv.FormatInt(user.ID, 10), nil
        },
    ),
)
```

Then attach to gRPC server:

```go
grpc.NewServer(
    grpc.UnaryInterceptor(rateLimiter.UnaryServerInterceptor()),
)
```

---

# Storage Backends

## Redis (Recommended)

Uses Lua script for atomic `INCR` + `PEXPIRE`.

```go
ratelimiter.WithCache(
    ratelimiteradapter.NewRedisCache(redisClient),
)
```

Guarantees:

* Atomic increment
* TTL set only on first increment

---

## In-Memory

Suitable for:

* Testing
* Single-instance services

```go
ratelimiter.WithCache(
    cache.NewInMemoryCache(),
)
```

---

# Key Strategy

Default storage key format:

```
rate-limiter:<namespace>:<rateKeyExtension>:<fullMethod>:<ruleName>:<sorted_attrs>
```

Example:

```
rate-limiter:hookah-culture:42:/auth.AuthService/SendCode:per_minute:phone=79998887766
```

You can override formatting:

```go
ratelimiter.WithRateKeyFormatter(customFormatter)
```

---

# Global Rules

Apply to **all** methods:

```go
ratelimiter.WithGlobalLimitRules([]ratelimiter.Rule{
    {
        Name:   "global",
        Limit:  100,
        Window: time.Minute,
    },
})
```

---

# Custom Rate Key

By default, a static value is used.

Override to inject:

* User ID
* API key
* Tenant ID
* Any context-based identity

```go
ratelimiter.WithRateKeyExtender(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo) (string, error) {
    return "custom-key", nil
})
```

---

# Error Behavior

When a rule is exceeded:

* gRPC status: `ResourceExhausted`
* Message: `rate limit exceeded: rule_name`

You can customize:

```go
ratelimiter.WithExceedErrorFormatter(customFormatter)
```

---

# Design Guarantees

* Deterministic key construction
* Atomic counter increment
* TTL never extended
* No sliding-window side effects
* Protobuf-driven configuration
* Zero reflection at runtime for rules (cached once)

---

# When To Use

Good fit for:

* Auth flows (OTP, login)
* Public APIs
* Multi-tenant systems
* Internal service protection

---

# License

MIT