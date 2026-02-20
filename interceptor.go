package ratelimiter

import (
	"context"
	"fmt"
	"strings"

	"github.com/samber/lo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	ratelimiterpb "github.com/murouse/rate-limiter/github.com/murouse/rate-limiter"
)

func (rl *RateLimiter) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Извлекаем rate key
		rateKey, err := rl.extractor.ExtractRateKey(ctx, req, info)
		if err != nil {
			rl.logger.Errorf("cannot extract rate key for method %q: %v", info.FullMethod, err)
			return nil, status.Errorf(codes.Internal, "cannot extract rate key: %v", err)
		}
		rl.logger.Debugf("extracted rate key %q for method %q", rateKey, info.FullMethod)

		methodRules := rl.getMethodRules()[info.FullMethod]
		rl.logger.Debugf("found %d rate limit rules for method %q", len(methodRules), info.FullMethod)

		exceededRules, err := rl.allow(ctx, rateKey, info.FullMethod, methodRules)
		if err != nil {
			rl.logger.Errorf("error checking rate limits for key %q, method %q: %v", rateKey, info.FullMethod, err)
			return nil, status.Errorf(codes.Internal, "rate limiter allow: %v", err)
		}
		if len(exceededRules) > 0 {
			msg := strings.Join(lo.Map(exceededRules, func(exceededRule Rule, _ int) string {
				return exceededRule.Name
			}), ", ")

			return nil, status.Errorf(codes.ResourceExhausted, "rate limit exceeded: %s", msg)
		}

		return handler(ctx, req)
	}
}

func (rl *RateLimiter) getMethodRules() map[string][]Rule {
	rl.methodRulesOnce.Do(rl.loadMethodRules)
	return rl.methodRules
}

func (rl *RateLimiter) loadMethodRules() {
	files := protoregistry.GlobalFiles
	rulesMap := make(map[string][]Rule)

	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		for i := 0; i < fd.Services().Len(); i++ {
			service := fd.Services().Get(i)

			for j := 0; j < service.Methods().Len(); j++ {
				method := service.Methods().Get(j)
				fullMethodName := fmt.Sprintf("/%s/%s", service.FullName(), method.Name())

				options := method.Options().(*descriptorpb.MethodOptions)
				if options == nil || !proto.HasExtension(options, ratelimiterpb.E_Rules) {
					continue
				}

				extension := proto.GetExtension(options, ratelimiterpb.E_Rules)
				if rulesSlice, ok := extension.([]*ratelimiterpb.Rule); ok {
					rulesMap[fullMethodName] = RateLimitRulesToModel(rulesSlice)
				}
			}
		}
		return true
	})

	rl.methodRules = rulesMap
}
