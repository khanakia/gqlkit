// Package builder provides the runtime foundation for generated GraphQL operation
// builders. It contains FieldSelection (a recursive tree that assembles the
// requested fields of a GraphQL selection set) and BaseBuilder (the shared logic
// every generated query/mutation builder delegates to for argument tracking,
// query string construction, and execution).
package builder

import (
	"context"
	"sort"
	"strings"
)

// GraphQLClient is the interface that the generated SDK expects consumers to
// implement (or satisfy via graphqlclient.Client). It sends a single GraphQL
// request and unmarshals the JSON response into the supplied target.
type GraphQLClient interface {
	Execute(ctx context.Context, query string, variables map[string]interface{}, response any) error
}

// FieldSelection tracks selected fields for a GraphQL query.
// Scalar fields are stored in `fields`; nested object selections are stored in
// `children`, forming a recursive tree that mirrors the GraphQL selection set.
type FieldSelection struct {
	fields   []string                    // leaf/scalar field names
	children map[string]*FieldSelection  // nested object selections keyed by field name
}

// NewFieldSelection creates a new FieldSelection
func NewFieldSelection() *FieldSelection {
	return &FieldSelection{
		fields:   make([]string, 0),
		children: make(map[string]*FieldSelection),
	}
}

// AddField adds a scalar field to the selection
func (fs *FieldSelection) AddField(name string) {
	fs.fields = append(fs.fields, name)
}

// AddChild adds a nested field selection
func (fs *FieldSelection) AddChild(name string, child *FieldSelection) {
	fs.children[name] = child
}

// Build recursively renders the field selection into a GraphQL selection set
// string. Each level is indented by `indent` levels of two spaces. Child keys
// are sorted alphabetically to ensure deterministic output across runs.
func (fs *FieldSelection) Build(indent int) string {
	if len(fs.fields) == 0 && len(fs.children) == 0 {
		return ""
	}

	var sb strings.Builder
	prefix := strings.Repeat("  ", indent)

	for _, field := range fs.fields {
		sb.WriteString(prefix + field + "\n")
	}

	// Sort children keys for deterministic output
	childKeys := make([]string, 0, len(fs.children))
	for k := range fs.children {
		childKeys = append(childKeys, k)
	}
	sort.Strings(childKeys)

	for _, name := range childKeys {
		child := fs.children[name]
		childStr := child.Build(indent + 1)
		if childStr != "" {
			sb.WriteString(prefix + name + " {\n")
			sb.WriteString(childStr)
			sb.WriteString(prefix + "}\n")
		}
	}

	return sb.String()
}

// BaseBuilder provides the shared state and logic for every generated operation
// builder. It tracks the GraphQL client, the operation type ("query" or
// "mutation"), the operation and field names, arguments with their GraphQL type
// annotations, and the field selection tree. Generated per-operation builders
// embed *BaseBuilder and add typed setter methods.
type BaseBuilder struct {
	client    GraphQLClient              // underlying HTTP transport
	opType    string                     // "query" or "mutation"
	opName    string                     // PascalCase operation name used in the query string
	fieldName string                     // the root field name inside the operation
	args      map[string]interface{}     // argument values keyed by GraphQL arg name
	argTypes  map[string]string          // GraphQL type strings keyed by arg name (e.g. "Int!", "[ID!]")
	selection *FieldSelection            // selected fields for the response
}

// NewBaseBuilder creates a new BaseBuilder
func NewBaseBuilder(client GraphQLClient, opType, opName, fieldName string) *BaseBuilder {
	return &BaseBuilder{
		client:    client,
		opType:    opType,
		opName:    opName,
		fieldName: fieldName,
		args:      make(map[string]interface{}),
		argTypes:  make(map[string]string),
		selection: NewFieldSelection(),
	}
}

// SetArg records an argument value and its corresponding GraphQL type string.
// Generated builders call this from their typed setter methods (e.g. SetLimit).
func (b *BaseBuilder) SetArg(name string, value interface{}, graphqlType string) {
	b.args[name] = value
	b.argTypes[name] = graphqlType
}

// GetSelection returns the field selection
func (b *BaseBuilder) GetSelection() *FieldSelection {
	return b.selection
}

// GetClient returns the client
func (b *BaseBuilder) GetClient() GraphQLClient {
	return b.client
}

// BuildQuery assembles the full GraphQL query/mutation string from the
// operation metadata, registered arguments, and field selection tree.
// Variable declarations are sorted alphabetically for deterministic output.
func (b *BaseBuilder) BuildQuery() string {
	var sb strings.Builder

	sb.WriteString(b.opType + " " + b.opName)

	if len(b.args) > 0 {
		sb.WriteString("(")
		vars := make([]string, 0, len(b.args))
		for name, gqlType := range b.argTypes {
			vars = append(vars, "$"+name+": "+gqlType)
		}
		sort.Strings(vars)
		sb.WriteString(strings.Join(vars, ", "))
		sb.WriteString(")")
	}

	sb.WriteString(" {\n")
	sb.WriteString("  " + b.fieldName)

	if len(b.args) > 0 {
		sb.WriteString("(")
		args := make([]string, 0, len(b.args))
		for name := range b.args {
			args = append(args, name+": $"+name)
		}
		sort.Strings(args)
		sb.WriteString(strings.Join(args, ", "))
		sb.WriteString(")")
	}

	selectionStr := b.selection.Build(2)
	if selectionStr != "" {
		sb.WriteString(" {\n")
		sb.WriteString(selectionStr)
		sb.WriteString("  }")
	}

	sb.WriteString("\n}")
	return sb.String()
}

// GetVariables returns the variables map
func (b *BaseBuilder) GetVariables() map[string]interface{} {
	return b.args
}

// ExecuteRaw builds the query, sends it via the GraphQL client, and returns the
// top-level response data as an untyped map. This is used internally by generated
// builders before unmarshalling into the concrete response struct.
func (b *BaseBuilder) ExecuteRaw(ctx context.Context) (map[string]interface{}, error) {
	query := b.BuildQuery()
	variables := b.GetVariables()

	var response map[string]interface{}
	if err := b.client.Execute(ctx, query, variables, &response); err != nil {
		return nil, err
	}

	return response, nil
}
