// Package tools is what the connector offers Claude: one tool per method of the admin services, read from their
// protos. A method's comment is its tool's description, its request is its input, and NO_SIDE_EFFECTS is what
// tells a read from a change. So a method added to an admin.proto is a tool after `make proto`, with no code here.
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

	// Whoever holds the token is not the operator who connected the app, or the token is not one at all.
	ErrNoToken       = errors.New("no live access token")
	ErrNotAnOperator = errors.New("not an operator")
)

const serviceName = "AdminService"

type Tool struct {
	Name        string
	Title       string
	Description string
	InputSchema Schema
	ReadOnly    bool
	// The Connect procedure it calls: /planet.v1.AdminService/BanPlayer.
	Procedure string
}

// What every tool's description is read against: the services, in the order their files come.
type Catalog struct {
	tools []Tool
}

// The admin services of a descriptor set, but those of a package left out and the methods hidden by their full name
// (planet.v1.AdminService.GetReplay).
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
				if slices.Contains(hidden, method.FullName()) || method.IsStreamingClient() || method.IsStreamingServer() {
					continue
				}

				tool := toolOf(method)
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

func toolOf(method protoreflect.MethodDescriptor) Tool {
	service := method.Parent().(protoreflect.ServiceDescriptor) //nolint:forcetypeassert // a method's parent is its service.
	words := wordsOf(string(method.Name()))
	options, _ := method.Options().(*descriptorpb.MethodOptions)

	return Tool{
		Name:        contextOf(service) + "_" + strings.ToLower(strings.Join(words, "_")),
		Title:       titleOf(words),
		Description: commentOf(method),
		InputSchema: SchemaOf(method.Input()),
		ReadOnly:    options.GetIdempotencyLevel() == descriptorpb.MethodOptions_NO_SIDE_EFFECTS,
		Procedure:   "/" + string(service.FullName()) + "/" + string(method.Name()),
	}
}

// planet.v1 is planet: the bounded context, which names the module a tool reaches.
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

// A leading comment, its lines joined: a description is prose, and the proto's line breaks are the editor's.
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

// What a call answered: the response as JSON, or the reason the service refused it, which Claude reads to try again.
type Result struct {
	Text    string
	Refused bool
}
