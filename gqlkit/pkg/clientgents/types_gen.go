package clientgents

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
)

// TSTypeDef holds the metadata for a single TypeScript interface generated from
// a GraphQL object or interface type.
type TSTypeDef struct {
	Name   string
	Fields []TSFieldDef
}

// TSFieldDef holds the metadata for a single field within a TS interface.
type TSFieldDef struct {
	Name     string // camelCase field name
	TypeStr  string // full "field?: Type;" or "field: Type;" line
	Optional bool
	TSType   string
}

// TSTypesData is the template data passed to ts_types.tmpl to generate types/index.ts.
type TSTypesData struct {
	EnumImports   []string
	ScalarImports []string
	Types         []TSTypeDef
}

// generateTypes collects all non-built-in object and interface types from the
// schema, converts them to TSTypeDef structs, and writes types/index.ts.
func (g *Generator) generateTypes() error {
	var types []TSTypeDef

	for _, def := range g.schema.Types {
		if def.BuiltIn || strings.HasPrefix(def.Name, "__") {
			continue
		}
		if def.Kind == ast.Object && (def.Name == "Query" || def.Name == "Mutation" || def.Name == "Subscription") {
			continue
		}
		if def.Kind != ast.Object && def.Kind != ast.Interface {
			continue
		}

		typeDef := TSTypeDef{
			Name: def.Name,
		}

		for _, field := range def.Fields {
			if strings.HasPrefix(field.Name, "__") {
				continue
			}
			fieldName := field.Name
			tsType := g.fieldTSType(field.Type)
			optional := !field.Type.NonNull

			typeDef.Fields = append(typeDef.Fields, TSFieldDef{
				Name:     fieldName,
				Optional: optional,
				TSType:   tsType,
			})
		}

		types = append(types, typeDef)
	}

	sort.Slice(types, func(i, j int) bool {
		return types[i].Name < types[j].Name
	})

	data := TSTypesData{
		EnumImports:   g.collectTypeEnumImports(),
		ScalarImports: g.collectTypeScalarImports(),
		Types:         types,
	}

	var buf bytes.Buffer
	if err := g.templates.ExecuteTemplate(&buf, "ts_types.tmpl", data); err != nil {
		return fmt.Errorf("failed to execute types template: %w", err)
	}

	return g.writer.WriteFile("types/index.ts", buf.String())
}
