package builder

import (
	"context"
	"sort"
	"strings"
)

type GraphQLClient interface {
	Execute(ctx context.Context, query string, variables map[string]interface{}, response any) error
}

// FieldSelection tracks selected fields for a GraphQL query
type FieldSelection struct {
	fields   []string
	children map[string]*FieldSelection
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

// Build builds the GraphQL field selection string
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

// BaseBuilder provides common builder functionality
type BaseBuilder struct {
	client    GraphQLClient
	opType    string
	opName    string
	fieldName string
	args      map[string]interface{}
	argTypes  map[string]string
	selection *FieldSelection
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

// SetArg sets an argument for the operation
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

// BuildQuery builds the GraphQL query string
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

// ExecuteRaw executes and returns raw map response
func (b *BaseBuilder) ExecuteRaw(ctx context.Context) (map[string]interface{}, error) {
	query := b.BuildQuery()
	variables := b.GetVariables()

	var response map[string]interface{}
	if err := b.client.Execute(ctx, query, variables, &response); err != nil {
		return nil, err
	}

	return response, nil
}
