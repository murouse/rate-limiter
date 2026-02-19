package ratelimiter

import (
	"context"

	"github.com/samber/lo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	ratelimiterpb "github.com/murouse/rate-limiter/gitlab.com/murouse/rate-limiter"
)

// UnaryServerInterceptor returns a gRPC unary interceptor that
// enforces fixed-window rate limiting.
//
// Behavior:
//
//   - Extracts rate key using RateKeyExtractor
//   - Reads rate limit rules from protobuf method options
//   - Applies global and method rules
//   - Returns ResourceExhausted if limit exceeded
//
// If no rules are defined for a method, the request proceeds normally.
func (rl *RateLimiter) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Извлекаем key
		rateKey, err := rl.extractor.ExtractRateKey(ctx, req, info)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "cannot extract rate key: %v", err)
		}

		// Получаем правила лимита из proto-опций метода
		protoRules := getRateLimitRules(info.FullMethod)

		// Конвертируем в локальные структуры
		rules := lo.Map(protoRules, func(r *ratelimiterpb.RateLimitRule, _ int) RateLimitRule {
			return RateLimitRule{
				Name:   r.Name,
				Limit:  int(r.Limit),
				Window: r.Window.AsDuration(),
			}
		})

		// Проверяем все правила лимита
		allowed, err := rl.allow(ctx, rateKey, info.FullMethod, rules)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "rate limiter allow: %v", err)
		}
		if !allowed {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}

		// Вызываем оригинальный обработчик
		return handler(ctx, req)
	}
}

// getRateLimitRules returns the protobuf-defined rate limit rules
// for a given gRPC full method name.
//
// It scans all registered proto files in protoregistry.GlobalFiles
// to find the method descriptor matching methodFullName and reads
// the (rate_limits) extension from its MethodOptions.
//
// Returns an empty slice if no rules are defined or the method is not found.
//
// The returned rules are raw protobuf definitions and should be
// converted to internal RateLimitRule structures before evaluation.
//
// Fixed-window semantics are enforced later by the Cache implementation.
func getRateLimitRules(methodFullName string) []*ratelimiterpb.RateLimitRule {
	files := protoregistry.GlobalFiles
	var rules []*ratelimiterpb.RateLimitRule

	// Проходим по всем файлам proto
	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		for i := 0; i < fd.Services().Len(); i++ {
			svc := fd.Services().Get(i)
			for j := 0; j < svc.Methods().Len(); j++ {
				m := svc.Methods().Get(j)
				fullName := string(svc.FullName()) + "/" + string(m.Name())
				if fullName == methodFullName {
					opts := m.Options().(*descriptorpb.MethodOptions)
					if opts != nil && proto.HasExtension(opts, ratelimiterpb.E_RateLimits) {
						ext := proto.GetExtension(opts, ratelimiterpb.E_RateLimits)
						if rulesSlice, ok := ext.([]*ratelimiterpb.RateLimitRule); ok {
							rules = append(rules, rulesSlice...)
						}
					}
					return false // нашли нужный метод, дальше не ищем
				}
			}
		}
		return true
	})

	return rules
}
