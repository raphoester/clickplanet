package tools

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

var (
	ErrUnknownTool = errors.New("no such tool")

	ErrNoToken       = errors.New("no live access token")
	ErrNotAnOperator = errors.New("not an operator")
)

const serviceName = "AdminService"

var (
	LeftOut = []protoreflect.FullName{"auth.v1"}
	Hidden  = []protoreflect.FullName{
		"planet.v1.AdminService.GetReplay",
		"planet.v1.AdminService.GetLedger",
		"planet.v1.AdminService.GetTerritories",
		"clips.v1.AdminService.KeepSnapshot",
		"clips.v1.AdminService.MakeCast",
		"clips.v1.AdminService.KeepAppearance",
		"publishing.v1.AdminService.CreateMediaUpload",
		"publishing.v1.AdminService.SchedulePost",
	}
)

type Tool struct {
	Name        string
	Title       string
	Description string
	InputSchema Schema
	ReadOnly    bool
	Procedure   string
}

type Catalog struct {
	tools []Tool
}

func CatalogOf(set []byte, leftOut []protoreflect.FullName, hidden []protoreflect.FullName) (*Catalog, error) {
	var files descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(set, &files); err != nil {
		return nil, fmt.Errorf("failed to read the descriptor set: %w", err)
	}
	registry, err := protodesc.NewFiles(&files)
	if err != nil {
		return nil, fmt.Errorf("failed to build the descriptors: %w", err)
	}

	catalog := &Catalog{}
	names := map[string]protoreflect.FullName{}
	var failed error
	registry.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		if slices.Contains(leftOut, file.Package()) {
			return true
		}
		for i := range file.Services().Len() {
			service := file.Services().Get(i)
			if service.Name() != serviceName {
				continue
			}
			for j := range service.Methods().Len() {
				method := service.Methods().Get(j)
				if slices.Contains(hidden, method.FullName()) || deprecated(method) ||
					method.IsStreamingClient() || method.IsStreamingServer() {
					continue
				}

				tool, err := toolOf(method)
				if err != nil {
					failed = err
					return false
				}
				if other, taken := names[tool.Name]; taken {
					failed = fmt.Errorf("%s and %s are both the tool %s", other, method.FullName(), tool.Name)
					return false
				}
				names[tool.Name] = method.FullName()
				catalog.tools = append(catalog.tools, tool)
			}
		}
		return true
	})
	if failed != nil {
		return nil, failed
	}

	return catalog, nil
}

func deprecated(method protoreflect.MethodDescriptor) bool {
	options, _ := method.Options().(*descriptorpb.MethodOptions)
	return options.GetDeprecated()
}

func toolOf(method protoreflect.MethodDescriptor) (Tool, error) {
	service := method.Parent().(protoreflect.ServiceDescriptor) //nolint:forcetypeassert // a method's parent is its service.
	words := wordsOf(string(method.Name()))
	options, _ := method.Options().(*descriptorpb.MethodOptions)

	input, err := SchemaOf(method.Input())
	if err != nil {
		return Tool{}, fmt.Errorf("failed to describe the input of %s: %w", method.FullName(), err)
	}

	return Tool{
		Name:        contextOf(service) + "_" + strings.ToLower(strings.Join(words, "_")),
		Title:       titleOf(words),
		Description: commentOf(method),
		InputSchema: input,
		ReadOnly:    options.GetIdempotencyLevel() == descriptorpb.MethodOptions_NO_SIDE_EFFECTS,
		Procedure:   "/" + string(service.FullName()) + "/" + string(method.Name()),
	}, nil
}

func contextOf(service protoreflect.ServiceDescriptor) string {
	context, _, _ := strings.Cut(string(service.ParentFile().Package()), ".")
	return context
}

func wordsOf(name string) []string {
	var words []string
	start := 0
	for i, r := range name {
		if i > 0 && unicode.IsUpper(r) {
			words = append(words, name[start:i])
			start = i
		}
	}
	return append(words, name[start:])
}

func titleOf(words []string) string {
	title := slices.Clone(words)
	for i := 1; i < len(title); i++ {
		title[i] = strings.ToLower(title[i])
	}
	return strings.Join(title, " ")
}

func commentOf(descriptor protoreflect.Descriptor) string {
	location := descriptor.ParentFile().SourceLocations().ByDescriptor(descriptor)
	return strings.Join(strings.Fields(location.LeadingComments), " ")
}

func (c *Catalog) Tools() []Tool {
	return slices.Clone(c.tools)
}

func (c *Catalog) Tool(name string) (Tool, error) {
	for _, tool := range c.tools {
		if tool.Name == name {
			return tool, nil
		}
	}
	return Tool{}, fmt.Errorf("%w: %q", ErrUnknownTool, name)
}

type Result struct {
	Text    string
	Refused bool
}
