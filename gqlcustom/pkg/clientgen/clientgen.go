package clientgen

import (
	"bytes"
	"fmt"
	"gqlcustom/pkg/schemagql"
	"gqlcustom/pkg/templater"
	"gqlcustom/pkg/typegql"
	"gqlcustom/pkg/util"
	"gqlcustom/pkg/writer"
	"sort"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
)

type ClientConfig struct {
	Bindings typegql.TypeMap `json:"bindings"`
}

// Generator orchestrates the SDK generation process
type Generator struct {
	config *Config
	schema *ast.Schema
	// typeMapper *TypeMapper
	writer *writer.Writer

	clientConfig *ClientConfig
	templates    *templater.Template
}

// New creates a new Generator from the given configuration
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

// Generate generates the SDK
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

type EnumDef struct {
	Name        string
	Description string
	EnumValues  []EnumValueDef
}

type EnumValueDef struct {
	Name        string
	Description string
}

type EnumDefMap map[string]EnumDef

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
