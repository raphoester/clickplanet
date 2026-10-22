package tools

import (
	"google.golang.org/protobuf/reflect/protoreflect"
)

// A JSON Schema, as MCP takes a tool's input. It is what protojson reads: the field's JSON name, an enum by its
// value's name, a timestamp in RFC 3339 and a duration in seconds, so Claude's arguments are forwarded as they are.
type Schema map[string]any

// A message that holds itself is described this deep, then as any object.
const maxDepth = 6

func SchemaOf(message protoreflect.MessageDescriptor) Schema {
	return objectOf(message, 0)
}

func objectOf(message protoreflect.MessageDescriptor, depth int) Schema {
	properties := map[string]any{}
	fields := message.Fields()
	for i := range fields.Len() {
		field := fields.Get(i)
		schema := fieldOf(field, depth)
		if description := commentOf(field); description != "" {
			schema["description"] = description
		}
		properties[field.JSONName()] = schema
	}

	return Schema{"type": "object", "properties": properties, "additionalProperties": false}
}

func fieldOf(field protoreflect.FieldDescriptor, depth int) Schema {
	switch {
	case field.IsMap():
		return Schema{"type": "object", "additionalProperties": singleOf(field.MapValue(), depth)}
	case field.IsList():
		return Schema{"type": "array", "items": singleOf(field, depth)}
	}
	return singleOf(field, depth)
}

func singleOf(field protoreflect.FieldDescriptor, depth int) Schema {
	switch field.Kind() {
	case protoreflect.BoolKind:
		return Schema{"type": "boolean"}
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return Schema{"type": "integer"}
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind, protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return Schema{"type": "integer", "minimum": 0}
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return Schema{"type": "number"}
	case protoreflect.StringKind:
		return Schema{"type": "string"}
	case protoreflect.BytesKind:
		return Schema{"type": "string", "contentEncoding": "base64"}
	case protoreflect.EnumKind:
		values := field.Enum().Values()
		names := make([]string, 0, values.Len())
		for i := range values.Len() {
			names = append(names, string(values.Get(i).Name()))
		}
		return Schema{"type": "string", "enum": names}
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return messageOf(field.Message(), depth+1)
	}
	return Schema{}
}

func messageOf(message protoreflect.MessageDescriptor, depth int) Schema {
	switch message.FullName() {
	case "google.protobuf.Timestamp":
		return Schema{"type": "string", "format": "date-time"}
	case "google.protobuf.Duration":
		return Schema{"type": "string", "pattern": `^-?[0-9]+(\.[0-9]+)?s$`, "examples": []string{"3600s"}}
	}
	if depth > maxDepth {
		return Schema{"type": "object"}
	}
	return objectOf(message, depth)
}
