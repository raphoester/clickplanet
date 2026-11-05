package cpproto

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Overlay is base with every value top holds written over it. A message top holds is overlaid field by field, but a
// well-known type or a message of lists only is one value; a list replaces base's when it holds an item.
func Overlay[M proto.Message](base, top M) M {
	overlaid := proto.CloneOf(base)
	write(overlaid.ProtoReflect(), proto.CloneOf(top).ProtoReflect())
	return overlaid
}

func write(base, top protoreflect.Message) {
	top.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		if isGroup(field) {
			write(base.Mutable(field).Message(), value.Message())
		} else {
			base.Set(field, value)
		}
		return true
	})
}

func isGroup(field protoreflect.FieldDescriptor) bool {
	message := field.Message()
	if message == nil || field.IsList() || field.IsMap() || message.ParentFile().Package() == "google.protobuf" {
		return false
	}

	fields := message.Fields()
	for i := range fields.Len() {
		if !fields.Get(i).IsList() {
			return true
		}
	}
	return false
}
