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
