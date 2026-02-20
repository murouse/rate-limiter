package ratelimiter

import (
	"context"
	"fmt"

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
		// Извлекаем атрибуты
		var attrs map[string]string
		if msg, ok := req.(proto.Message); ok {
			attrs = extractRateKeyAttrs(msg)
		}

		// Извлекаем дополнительный кастомный rate key (например идентификатор пользователя из контекста)
		rateKeyExtension, err := rl.rateKeyExtender.ExtendRateKey(ctx, req, info)
		if err != nil {
			rl.logger.Errorf("cannot extend rate key for method %q: %v", info.FullMethod, err)
			return nil, status.Errorf(codes.Internal, "cannot extend rate key: %v", err)
		}
		rl.logger.Debugf("rate key extension %q for method %q", rateKeyExtension, info.FullMethod)

		methodRules := rl.getMethodRules()[info.FullMethod]
		rl.logger.Debugf("found %d rate limit rules for method %q", len(methodRules), info.FullMethod)

		exceededRules, err := rl.allow(ctx, rateKeyExtension, info.FullMethod, attrs, methodRules)
		if err != nil {
			rl.logger.Errorf("error checking rate limits for key %q, method %q: %v", rateKeyExtension, info.FullMethod, err)
			return nil, status.Errorf(codes.Internal, "rate limiter allow: %v", err)
		}
		if len(exceededRules) > 0 {
			return nil, rl.exceedErrorFormatter(exceededRules)
		}

		return handler(ctx, req)
	}
}

func (rl *RateLimiter) allow(ctx context.Context, rateKeyExtension, fullMethod string, attrs map[string]string, methodRules []Rule) ([]Rule, error) {
	var exceededRules []Rule

	for _, globalRule := range rl.globalLimitRules {
		fullRateKey := rl.rateKeyFormatter(rl.namespace, rateKeyExtension, fullMethod, globalRule.Name, attrs)

		ok, err := rl.checkRule(ctx, fullRateKey, globalRule)
		if err != nil {
			return nil, fmt.Errorf("failed to check global rule: %w", err)
		}
		if !ok {
			exceededRules = append(exceededRules, globalRule)
		}
	}

	for _, methodRule := range methodRules {
		fullRateKey := rl.rateKeyFormatter(rl.namespace, rateKeyExtension, fullMethod, methodRule.Name, attrs)

		ok, err := rl.checkRule(ctx, fullRateKey, methodRule)
		if err != nil {
			return nil, fmt.Errorf("failed to check method rule: %w", err)
		}
		if !ok {
			exceededRules = append(exceededRules, methodRule)
		}
	}

	return exceededRules, nil
}

func (rl *RateLimiter) checkRule(ctx context.Context, fullRateKey string, rule Rule) (bool, error) {
	count, err := rl.cache.Increment(ctx, fullRateKey, rule.Window)
	if err != nil {
		rl.logger.Errorf("increment failed for key %q: %v", fullRateKey, err)
		return false, fmt.Errorf("increment: %w", err)
	}

	if count > int64(rule.Limit) {
		return false, nil
	}

	return true, nil
}

func extractRateKeyAttrs(msg proto.Message) map[string]string {
	attrs := make(map[string]string)
	if msg == nil {
		return attrs
	}

	var walk func(ref protoreflect.Message, prefix string)
	walk = func(ref protoreflect.Message, prefix string) {
		desc := ref.Descriptor()

		for i := 0; i < desc.Fields().Len(); i++ {
			field := desc.Fields().Get(i)
			opts, ok := field.Options().(*descriptorpb.FieldOptions)
			if !ok || opts == nil {
				continue
			}

			fieldName := string(field.Name())
			fullName := fieldName
			if prefix != "" {
				fullName = prefix + "." + fieldName
			}

			// Проверяем опцию rate_key
			if proto.HasExtension(opts, ratelimiterpb.E_RateKey) {
				alias := proto.GetExtension(opts, ratelimiterpb.E_RateKey).(string)
				if ref.Has(field) {
					val := ref.Get(field)
					attrs[alias] = formatProtoValue(val)
				}
			}

			// Рекурсивно для вложенных сообщений
			if field.Kind() == protoreflect.MessageKind && ref.Has(field) {
				if field.IsList() {
					list := ref.Get(field).List()
					for j := 0; j < list.Len(); j++ {
						if m, ok := list.Get(j).Message().Interface().(proto.Message); ok {
							walk(m.ProtoReflect(), fullName)
						}
					}
				} else {
					if m, ok := ref.Get(field).Message().Interface().(proto.Message); ok {
						walk(m.ProtoReflect(), fullName)
					}
				}
			}
		}
	}

	walk(msg.ProtoReflect(), "")
	return attrs
}

// formatProtoValue преобразует protoreflect.Value в строку
func formatProtoValue(val protoreflect.Value) string {
	switch v := val.Interface().(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
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
