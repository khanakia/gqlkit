// Package clientgents is the main orchestrator for generating a type-safe
// TypeScript client SDK from a parsed GraphQL schema. It mirrors the structure
// of the Go clientgen package but targets TypeScript output: interfaces instead
// of structs, TS enums, type aliases for scalars, and class-based builders that
// extend the gqlkit-ts BaseBuilder.
package clientgents

import (
	"embed"
	"fmt"
	"github.com/khanakia/gqlkit/gqlkit/pkg/schemagql"
	"sort"
	"strings"
	"text/template"

	"github.com/vektah/gqlparser/v2/ast"
)

// templateDir embeds all .tmpl files under the template/ subdirectory. These
// are the TypeScript code generation templates.
//
//go:embed template/*
var templateDir embed.FS

// Generator orchestrates the entire TypeScript SDK generation process. It holds
// the parsed schema, TS type map, compiled templates, and the file writer.
type Generator struct {
	config       *Config
	schema       *ast.Schema
	writer       *TSWriter
	clientConfig *ClientConfig
	tsTypeMap    TSTypeMap
	templates    *template.Template
}

// New creates a new Generator by validating config, loading the client config,
// parsing the schema, building the TS type map, and compiling templates.
func New(config *Config) (*Generator, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	clientConfig, err := loadClientConfig(config.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load client config: %w", err)
	}

	schema, err := schemagql.GetSchema(schemagql.StringList{config.SchemaPath})
	if err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}

	// Build TS type map: built-in types + config overrides
	tsTypeMap := BuiltInTSTypes()
	if clientConfig.Bindings != nil {
		tsTypeMap.Merge(clientConfig.Bindings)
	}

	funcMap := template.FuncMap{
		"lower":     strings.ToLower,
		"upper":     strings.ToUpper,
		"joinComma": func(s []string) string { return strings.Join(s, ", ") },
	}

	templates, err := template.New("").Funcs(funcMap).ParseFS(templateDir, "template/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return &Generator{
		config:       config,
		schema:       schema,
		clientConfig: clientConfig,
		tsTypeMap:    tsTypeMap,
		writer:       NewTSWriter(config.OutputDir),
		templates:    templates,
	}, nil
}

// Generate runs the full TypeScript code generation pipeline: builder re-exports,
// scalars, enums, types, inputs, field selectors, and operation builders.
func (g *Generator) Generate() error {
	fmt.Printf("Generating TypeScript SDK from %s\n", g.config.SchemaPath)
	fmt.Printf("Output directory: %s\n", g.config.OutputDir)

	if err := g.writer.EnsureDir(); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 1. builder/index.ts
	if err := g.generateBuilderIndex(); err != nil {
		return fmt.Errorf("failed to generate builder/index.ts: %w", err)
	}
	fmt.Println("Generated: builder/index.ts")

	// 2. scalars/index.ts
	if err := g.generateScalars(); err != nil {
		return fmt.Errorf("failed to generate scalars: %w", err)
	}
	fmt.Println("Generated: scalars/index.ts")

	// 3. enums/index.ts
	if err := g.generateEnums(); err != nil {
		return fmt.Errorf("failed to generate enums: %w", err)
	}
	fmt.Println("Generated: enums/index.ts")

	// 4. types/index.ts
	if err := g.generateTypes(); err != nil {
		return fmt.Errorf("failed to generate types: %w", err)
	}
	fmt.Println("Generated: types/index.ts")

	// 5. inputs/index.ts
	if err := g.generateInputTypes(); err != nil {
		return fmt.Errorf("failed to generate inputs: %w", err)
	}
	fmt.Println("Generated: inputs/index.ts")

	// 6. fields/*.ts + fields/index.ts
	if err := g.generateFieldSelectionFiles(); err != nil {
		return fmt.Errorf("failed to generate field selectors: %w", err)
	}
	fmt.Println("Generated: fields/*.ts")

	// 7. queries/*.ts + queries/root.ts + queries/index.ts
	// 8. mutations/*.ts + mutations/root.ts + mutations/index.ts
	if err := g.generateOperationFiles(); err != nil {
		return fmt.Errorf("failed to generate operations: %w", err)
	}
	fmt.Println("Generated: queries/ and mutations/")

	fmt.Printf("TypeScript SDK generated successfully in %s\n", g.config.OutputDir)
	return nil
}

// generateBuilderIndex generates builder/index.ts which re-exports the
// FieldSelection, BaseBuilder, and GraphQLClient types from the gqlkit-ts npm package.
func (g *Generator) generateBuilderIndex() error {
	content := `// Re-export builder classes from gqlkit-ts
export { FieldSelection, BaseBuilder } from "gqlkit-ts";
export { GraphQLClient } from "gqlkit-ts";
`
	return g.writer.WriteFile("builder/index.ts", content)
}

// getSortedObjectTypeNames returns schema object type names (excluding builtin and roots), sorted
func (g *Generator) getSortedObjectTypeNames() []string {
	var names []string
	for name, def := range g.schema.Types {
		if def.BuiltIn || strings.HasPrefix(name, "__") {
			continue
		}
		if def.Kind != ast.Object && def.Kind != ast.Interface {
			continue
		}
		if name == "Query" || name == "Mutation" || name == "Subscription" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// getBaseTypeName returns the named type after unwrapping List and NonNull
func getBaseTypeName(t *ast.Type) string {
	if t == nil {
		return ""
	}
	if t.Elem != nil {
		return getBaseTypeName(t.Elem)
	}
	return t.NamedType
}

// isObjectType returns true if the type is an Object or Interface
func isObjectType(schema *ast.Schema, t *ast.Type) bool {
	name := getBaseTypeName(t)
	if name == "" {
		return false
	}
	def := schema.Types[name]
	if def == nil {
		return false
	}
	return def.Kind == ast.Object || def.Kind == ast.Interface
}

// formatGraphQLType reconstructs the GraphQL type notation string from an AST
// type (e.g. "[ChatbotOrder!]" or "Int!"). Used in generated setArg calls.
func formatGraphQLType(t *ast.Type) string {
	if t == nil {
		return ""
	}

	var result string
	if t.Elem != nil {
		result = "[" + formatGraphQLType(t.Elem) + "]"
	} else {
		result = t.NamedType
	}

	if t.NonNull {
		result += "!"
	}

	return result
}
