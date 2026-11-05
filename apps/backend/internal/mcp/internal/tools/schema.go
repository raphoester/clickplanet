package tools

import (
	"fmt"

	"google.golang.org/protobuf/reflect/protoreflect"
)

type Schema map[string]any

// A message that holds itself is described this deep, then as any object.
const maxDepth = 6

func SchemaOf(message protoreflect.MessageDescriptor) (Schema, error) {
	return objectOf(message, 0)
}

func objectOf(message protoreflect.MessageDescriptor, depth int) (Schema, error) {
	properties := map[string]any{}
	fields := message.Fields()
	for i := range fields.Len() {
		field := fields.Get(i)
		schema, err := fieldOf(field, depth)
		if err != nil {
			return nil, err
		}
		if description := commentOf(field); description != "" {
			schema["description"] = description
		}
		properties[field.JSONName()] = schema
	}

	return Schema{"type": "object", "properties": properties, "additionalProperties": false}, nil
}

func fieldOf(field protoreflect.FieldDescriptor, depth int) (Schema, error) {
	switch {
	case field.IsMap():
		value, err := singleOf(field.MapValue(), depth)
		return Schema{"type": "object", "additionalProperties": value}, err
	case field.IsList():
		item, err := singleOf(field, depth)
		return Schema{"type": "array", "items": item}, err
	}
	return singleOf(field, depth)
}

func singleOf(field protoreflect.FieldDescriptor, depth int) (Schema, error) {
	switch field.Kind() {
	case protoreflect.BoolKind:
		return Schema{"type": "boolean"}, nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return Schema{"type": "integer"}, nil
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind, protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return Schema{"type": "integer", "minimum": 0}, nil
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return Schema{"type": "number"}, nil
	case protoreflect.StringKind:
		return Schema{"type": "string"}, nil
	case protoreflect.BytesKind:
		return Schema{"type": "string", "contentEncoding": "base64"}, nil
	case protoreflect.EnumKind:
		values := field.Enum().Values()
		names := make([]string, 0, values.Len())
		for i := range values.Len() {
			names = append(names, string(values.Get(i).Name()))
		}
		return Schema{"type": "string", "enum": names}, nil
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return messageOf(field.Message(), depth+1)
	}
	return nil, fmt.Errorf("%s: a field of kind %s has no schema", field.FullName(), field.Kind())
}

func messageOf(message protoreflect.MessageDescriptor, depth int) (Schema, error) {
	switch message.FullName() {
	case "google.protobuf.Timestamp":
		return Schema{"type": "string", "format": "date-time"}, nil
	case "google.protobuf.Duration":
		return Schema{"type": "string", "pattern": `^-?[0-9]+(\.[0-9]+)?s$`, "examples": []string{"3600s"}}, nil
	}
	if depth > maxDepth {
		return Schema{"type": "object"}, nil
	}
	return objectOf(message, depth)
}
