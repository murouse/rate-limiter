package ratelimiter

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// Запрос с map-полем, у которого есть опции (например, buf.validate), не должен ронять интерцептор:
// обход вызывал Message() у map и паниковал «cannot convert map to message»
func TestExtractRateKeyAttrsSkipsMaps(t *testing.T) {
	file, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Name:    proto.String("maptest.proto"),
		Package: proto.String("maptest"),
		Syntax:  proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{{
			Name: proto.String("Request"),
			Field: []*descriptorpb.FieldDescriptorProto{{
				Name:     proto.String("context"),
				Number:   proto.Int32(1),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				TypeName: proto.String(".maptest.Request.ContextEntry"),
				JsonName: proto.String("context"),
				Options:  &descriptorpb.FieldOptions{Deprecated: proto.Bool(true)},
			}},
			NestedType: []*descriptorpb.DescriptorProto{{
				Name: proto.String("ContextEntry"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: proto.String("key"), Number: proto.Int32(1), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), JsonName: proto.String("key")},
					{Name: proto.String("value"), Number: proto.Int32(2), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), JsonName: proto.String("value")},
				},
				Options: &descriptorpb.MessageOptions{MapEntry: proto.Bool(true)},
			}},
		}},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	desc := file.Messages().ByName("Request")
	msg := dynamicpb.NewMessage(desc)
	field := desc.Fields().ByName("context")
	m := msg.Mutable(field).Map()
	m.Set(protoreflect.ValueOfString("version").MapKey(), protoreflect.ValueOfString("abc123"))

	if attrs := extractRateKeyAttrs(msg); len(attrs) != 0 {
		t.Fatalf("want no attrs, got %v", attrs)
	}
}

type failingCache struct{}

func (failingCache) Increment(context.Context, string, time.Duration) (int64, error) {
	return 0, errors.New("i/o timeout")
}

// Хранилище не ответило — Unavailable (запрос можно повторить), а не Internal; с WithFailOpen — запрос проходит
func TestCacheFailure(t *testing.T) {
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}
	rules := WithGlobalLimitRules([]Rule{{Name: "global", Limit: 1, Window: time.Minute}})
	handler := func(context.Context, any) (any, error) { return "ok", nil }

	_, err := New(WithCache(failingCache{}), rules).UnaryServerInterceptor()(context.Background(), nil, info, handler)
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("want Unavailable, got %v", err)
	}

	res, err := New(WithCache(failingCache{}), rules, WithFailOpen()).
		UnaryServerInterceptor()(context.Background(), nil, info, handler)
	if err != nil || res != "ok" {
		t.Fatalf("fail-open: want handler result, got %v, %v", res, err)
	}
}

// Общее правило (Shared) считает все вызовы метода вместе: разные rate_key и пользователи — один счетчик,
// а обычное правило — свой счетчик на каждый ключ
func TestSharedRule(t *testing.T) {
	ctx := context.Background()
	rl := New()
	rules := []Rule{
		{Name: "per_key", Limit: 1, Window: time.Minute},
		{Name: "total", Limit: 2, Window: time.Minute, Shared: true},
	}
	allow := func(user, phone string) []Rule {
		exceeded, err := rl.allow(ctx, user, "/test.Service/Method", map[string]string{"phone": phone}, rules)
		if err != nil {
			t.Fatal(err)
		}
		return exceeded
	}

	if ex := allow("u1", "1"); len(ex) != 0 {
		t.Fatalf("first call: exceeded %v", ex)
	}
	if ex := allow("u2", "2"); len(ex) != 0 {
		t.Fatalf("another key: exceeded %v", ex)
	}
	if ex := allow("u3", "3"); len(ex) != 1 || ex[0].Name != "total" {
		t.Fatalf("third key: want only total exceeded, got %v", ex)
	}
	if ex := allow("u1", "1"); len(ex) != 2 {
		t.Fatalf("same key again: want per_key and total exceeded, got %v", ex)
	}
}

// Имя правила без name — из самого правила: лимит, окно и общее ли оно
func TestRuleName(t *testing.T) {
	for _, tc := range []struct {
		rule Rule
		want string
	}{
		{Rule{Limit: 2, Window: time.Minute}, "2_per_1m"},
		{Rule{Limit: 10, Window: 24 * time.Hour}, "10_per_1d"},
		{Rule{Limit: 30, Window: time.Hour}, "30_per_1h"},
		{Rule{Limit: 5, Window: 90 * time.Second}, "5_per_90s"},
		{Rule{Limit: 5000, Window: 24 * time.Hour, Shared: true}, "5000_per_1d_shared"},
		{Rule{Name: "custom", Limit: 1, Window: time.Minute}, "custom"},
	} {
		if got := tc.rule.withName().Name; got != tc.want {
			t.Errorf("%+v: want %q, got %q", tc.rule, tc.want, got)
		}
	}

	if got := New(WithGlobalLimitRules([]Rule{{Limit: 1, Window: time.Second}})).globalLimitRules[0].Name; got != "1_per_1s" {
		t.Errorf("global rule: want 1_per_1s, got %q", got)
	}
}
