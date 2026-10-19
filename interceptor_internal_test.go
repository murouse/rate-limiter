package ratelimiter

import (
	"testing"

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
