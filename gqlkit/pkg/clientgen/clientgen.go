// Package clientgen is the main orchestrator for generating a type-safe Go
// client SDK from a parsed GraphQL schema. It coordinates schema parsing,
// type mapping, template execution, and file writing to produce a complete
// Go package with types, enums, scalars, input types, field selectors, and
// per-operation query/mutation builders.
package clientgen

import (
	"bytes"
	"fmt"
	"gqlkit/pkg/schemagql"
	"gqlkit/pkg/templater"
	"gqlkit/pkg/typegql"
	"gqlkit/pkg/util"
	"gqlkit/pkg/writer"
	"sort"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
)

// ClientConfig holds user-specified scalar-to-Go-type bindings loaded from the
// JSON config file. These bindings override the default built-in type mappings.
type ClientConfig struct {
	Bindings typegql.TypeMap `json:"bindings"`
}

// Generator orchestrates the entire Go SDK generation process. It holds the
// parsed schema, resolved type bindings, compiled templates, and the file
// writer. Call New() to create one, then Generate() to produce all output files.
type Generator struct {
	config       *Config            // CLI/caller-supplied generation settings
	schema       *ast.Schema        // parsed and validated GraphQL schema
	writer       *writer.Writer     // writes formatted Go files to the output directory
	clientConfig *ClientConfig      // user-specified scalar bindings from config.jsonc
	templates    *templater.Template // compiled Go code generation templates
}

// New creates a new Generator by validating the config, loading the client
// config file, parsing the GraphQL schema, merging built-in and user type
// bindings, and compiling the embedded Go templates.
func New(config *Config) (*Generator, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	clientConfig, err := loadClientConfig(config.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load client config: %w", err)
	}

	if clientConfig == nil {
		return nil, fmt.Errorf("client config is nil")
	}

	// Parse schema
	// schema, err := parseSchemaFile(config.SchemaPath)
	schema, err := schemagql.GetSchema(schemagql.StringList{config.SchemaPath})
	if err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}

	typeMapEntryMap := typegql.Merge(typegql.BuiltInTypes(), clientConfig.Bindings)
	typeMapEntryMap = typegql.Build(typeMapEntryMap)
	clientConfig.Bindings = typeMapEntryMap

	templates := templater.MustParse(templater.NewTemplate("templates").
		ParseFS(templater.TemplateDir(), "template/*.tmpl"))

	return &Generator{
		config:       config,
		schema:       schema,
		clientConfig: clientConfig,
		// typeMapper: NewTypeMapper(),
		writer:    writer.NewWriter(config.OutputDir),
		templates: templates,
	}, nil
}

// GetSchema returns the parsed schema
func (g *Generator) GetSchema() *ast.Schema {
	return g.schema
}

// Generate runs the full code generation pipeline: scalars, enums, types,
// input types, the builder base, field selectors, and query/mutation operation
// builders. Each step writes one or more files to the configured output directory.
func (g *Generator) Generate() error {
	fmt.Printf("Generating SDK from %s\n", g.config.SchemaPath)
	fmt.Printf("Output directory: %s\n", g.config.OutputDir)
	fmt.Printf("Package name: %s\n", g.config.PackageName)

	// Ensure output directory exists
	if err := g.writer.EnsureDir(); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	util.DumpStructToFile(g.schema, "schema.json")

	// Generate go.mod
	// if err := g.writer.WriteGoMod(g.config.ModulePath, g.config.PackageName); err != nil {
	// 	return fmt.Errorf("failed to write go.mod: %w", err)
	// }
	// fmt.Println("Generated: go.mod")

	// Generate scalars
	if err := g.generateScalars(); err != nil {
		return fmt.Errorf("failed to generate scalars: %w", err)
	}
	fmt.Println("Generated: scalars.go")

	// Generate enums
	if err := g.generateEnums(); err != nil {
		return fmt.Errorf("failed to generate enums: %w", err)
	}
	fmt.Println("Generated: enums.go")

	// Generate types
	if err := g.generateTypes(); err != nil {
		return fmt.Errorf("failed to generate types: %w", err)
	}
	fmt.Println("Generated: types.go")

	// Generate inputs
	if err := g.generateInputTypes(); err != nil {
		return fmt.Errorf("failed to generate input types: %w", err)
	}
	fmt.Println("Generated: inputs.go")

	// Generate builder files
	if err := g.generateBuilderFiles(); err != nil {
		return fmt.Errorf("failed to generate builder files: %w", err)
	}
	fmt.Println("Generated: builder.go")

	// Generate field selection files (one per type in fields/)
	if err := g.generateFieldSelectionFiles(); err != nil {
		return fmt.Errorf("failed to generate field selection files: %w", err)
	}
	fmt.Println("Generated: fields/field_*.go")

	// Generate query and mutation builders (queries/, mutations/)
	if err := g.generateOperationFiles(); err != nil {
		return fmt.Errorf("failed to generate operation files: %w", err)
	}
	fmt.Println("Generated: queries/ and mutations/")

	// // Generate inputs
	// if err := g.generateInputs(); err != nil {
	// 	return fmt.Errorf("failed to generate inputs: %w", err)
	// }
	// fmt.Println("  Generated: inputs.go")

	// Generate client
	// if err := g.generateClient(); err != nil {
	// 	return fmt.Errorf("failed to generate client: %w", err)
	// }
	// fmt.Println("  Generated: client.go")

	// Generate builder files (separate files for each query/mutation)
	// if err := g.generateBuilderFiles(); err != nil {
	// 	return fmt.Errorf("failed to generate builder files: %w", err)
	// }

	fmt.Printf("SDK generated successfully in %s\n", g.config.OutputDir)
	return nil
}

// generateBuilderFiles renders the builder/builder.go file that re-exports the
// base builder types from the builder package into the generated SDK.
func (g *Generator) generateBuilderFiles() error {
	b := bytes.NewBuffer(nil)
	err := g.templates.ExecuteTemplate(b, "builder", map[string]interface{}{
		"Config":      g.config,
		"PackageName": "builder",
	})
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}
	content := b.String()
	return g.writer.WriteFile("builder/builder.go", content)
}

// generateScalars builds scalar type aliases (e.g. type DateTime = time.Time)
// and writes them to scalars/scalars.go.
func (g *Generator) generateScalars() error {
	scalarData := buildSchemaScalarMap(g.schema, g.clientConfig.Bindings)
	b := bytes.NewBuffer(nil)
	err := g.templates.ExecuteTemplate(b, "scalar", map[string]interface{}{
		"Config":  g.config,
		"Scalars": scalarData.SchemaScalarMap,
		"Imports": scalarData.Imports,
		"Package": "scalars",
	})
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}
	content := b.String()

	return g.writer.WriteFile("scalars/scalars.go", content)
}

// EnumDef represents a single GraphQL enum type with its values, ready for
// template rendering.
type EnumDef struct {
	Name        string
	Description string
	EnumValues  []EnumValueDef
}

// EnumValueDef represents a single value within a GraphQL enum.
type EnumValueDef struct {
	Name        string
	Description string
}

// EnumDefMap maps enum names to their definitions.
type EnumDefMap map[string]EnumDef

// generateEnums collects all non-built-in enum types from the schema,
// sorts them alphabetically, and writes enums/enums.go via the enums template.
func (g *Generator) generateEnums() error {

	enumDefMap := make(EnumDefMap)
	for _, def := range g.schema.Types {
		if def.BuiltIn || strings.HasPrefix(def.Name, "__") {
			continue
		}

		if def.Kind == ast.Enum {
			enumDef := EnumDef{
				Name:        def.Name,
				Description: def.Description,
			}
			// goutil.PrintToJSON(def)
			// fmt.Println(def.Name)
			for _, val := range def.EnumValues {
				enumDef.EnumValues = append(enumDef.EnumValues, EnumValueDef{
					Name:        val.Name,
					Description: val.Description,
				})
			}
			enumDefMap[def.Name] = enumDef
		}
	}

	// Convert map to sorted slice for deterministic output
	enumList := make([]EnumDef, 0, len(enumDefMap))
	for _, enumDef := range enumDefMap {
		enumList = append(enumList, enumDef)
	}
	sort.Slice(enumList, func(i, j int) bool {
		return enumList[i].Name < enumList[j].Name
	})

	b := bytes.NewBuffer(nil)
	err := g.templates.ExecuteTemplate(b, "enums", map[string]interface{}{
		"Config":  g.config,
		"Enums":   enumList,
		"Package": "enums",
	})
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}
	content := b.String()

	return g.writer.WriteFile("enums/enums.go", content)
}

// extractLocalPackageName extracts the local package name from an import path
// e.g., "testsdk/api" -> "api", "mypackage" -> "mypackage"
// func extractLocalPackageName(importPath string) string {
// 	if idx := strings.LastIndex(importPath, "/"); idx != -1 {
// 		return importPath[idx+1:]
// 	}
// 	return importPath
// }
