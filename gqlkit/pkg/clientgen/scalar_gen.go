package clientgen

import (
	"gqlkit/pkg/typegql"

	"github.com/vektah/gqlparser/v2/ast"
)

// SchemaScalar holds the metadata for a single GraphQL scalar type, including
// the resolved Go type string and whether it is one of the five built-in
// GraphQL scalars (String, Int, Float, Boolean, ID).
type SchemaScalar struct {
	Name        string
	Description string
	// GoType       string
	IsBuiltIn bool
	GoType    string
	// TypeMapEntry typegql.TypeMapEntry
}

// SchemaScalarMap maps GraphQL scalar names to their SchemaScalar metadata.
type SchemaScalarMap map[string]SchemaScalar

// builtInScalars identifies the five standard GraphQL scalar types. These are
// mapped to Go primitives by the built-in type map and do not need custom
// type alias generation.
var builtInScalars = map[string]bool{
	"String":  true,
	"Int":     true,
	"Float":   true,
	"Boolean": true,
	"ID":      true,
}

// ScalarData bundles the scalar map and required imports for template rendering.
type ScalarData struct {
	SchemaScalarMap SchemaScalarMap
	Imports         []string
}

// buildSchemaScalarMap iterates over all scalar types in the schema, resolves
// each one's Go type via the bindings map (falling back to `any`), and collects
// the necessary Go import paths. Returns the scalar map and import list.
func buildSchemaScalarMap(schema *ast.Schema, bindings typegql.TypeMap) ScalarData {
	schemaScalarMap := make(SchemaScalarMap)
	imports := make([]string, 0)

	for _, scalar := range schema.Types {
		if scalar.Kind != ast.Scalar {
			continue
		}
		typeMapEntry, ok := bindings[scalar.Name]
		if !ok {
			typeMapEntry = typegql.AnyType()
		}

		schemaScalarMap[scalar.Name] = SchemaScalar{
			Name:        scalar.Name,
			Description: scalar.Description,
			GoType:      typeMapEntry.GoType,
			IsBuiltIn:   builtInScalars[scalar.Name],
		}

		if typeMapEntry.GoImport != "" {
			imports = append(imports, typeMapEntry.GoImport)
		}
	}
	return ScalarData{
		SchemaScalarMap: schemaScalarMap,
		Imports:         imports,
	}
}
