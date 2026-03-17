package clientgen

import (
	"bytes"
	"fmt"
	"gqlkit/pkg/util"
	"sort"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
)

// TypeDef holds the metadata for a single Go struct generated from a GraphQL
// object or interface type. Interfaces produce marker types with an Is<Name>()
// method; objects produce full structs with fields.
type TypeDef struct {
	Name        string
	Description string
	Fields      []FieldDef
	IsInterface bool
}

// FieldDef holds the metadata for a single field within a generated Go struct.
type FieldDef struct {
	Name        string
	Description string
	GoType      string
	JSONTag     string
	OmitEmpty   bool
}

// TypeDefList is a sortable slice of TypeDef used for deterministic output.
type TypeDefList []TypeDef

// graphQLToGoType converts a GraphQL AST type to its Go equivalent string.
// It handles lists ([]T), non-null vs nullable (*T for pointers), and
// delegates named type resolution to namedTypeToGo.
func (g *Generator) graphQLToGoType(t *ast.Type) string {
	if t == nil {
		return "interface{}"
	}

	goType := g.resolveType(t)

	// If nullable (not NonNull), make it a pointer (except for slices)
	if !t.NonNull && !strings.HasPrefix(goType, "[]") {
		goType = "*" + goType
	}

	return goType
}

// resolveType resolves the base Go type from a GraphQL type
func (g *Generator) resolveType(t *ast.Type) string {
	if t.Elem != nil {
		// It's a list type
		elemType := g.graphQLToGoType(t.Elem)
		return "[]" + elemType
	}

	// Named type
	return g.namedTypeToGo(t.NamedType)
}

// namedTypeToGo converts a named GraphQL type to its Go representation.
// It checks custom bindings first, then falls back to schema kind-based
// resolution (scalars. prefix, enums. prefix, or plain type name).
func (g *Generator) namedTypeToGo(name string) string {

	// Check if there's a custom binding
	if entry, ok := g.clientConfig.Bindings[name]; ok {
		return entry.GoType
		// // If GoPackage is set by typegql.Build, construct type as package.TypeName
		// if entry.PkgName != "" {
		// 	// Extract type name from Model (which is set to t.Obj().Name() by typegql.Build)
		// 	typeName := entry.Model
		// 	if typeName == "" {
		// 		typeName = name
		// 	}
		// 	// return entry.GoPackage + "." + typeName
		// 	return entry.GoType
		// }
		// // Use GoType if available (for built-in types processed by typegql)
		// if entry.GoType != "" {
		// 	return entry.GoType
		// }
		// // Fallback to Model
		// return entry.Model
	}

	def := g.schema.Types[name]

	switch def.Kind {
	case ast.Scalar:
		return "scalars." + def.Name
	case ast.Enum:
		return "enums." + def.Name
	case ast.Object:
		return def.Name
	case ast.InputObject:
		return def.Name
		// case ast.Interface:
		// 	return def.Name
		// // TODO: Handle union types
		// case ast.Union:
		// 	return def.Name
	}

	return "interface{}"

	// Built-in scalars
	// switch name {
	// case "String":
	// 	return "string"
	// case "Int":
	// 	return "int"
	// case "Float":
	// 	return "float64"
	// case "Boolean":
	// 	return "bool"
	// case "ID":
	// 	return "string"
	// }

	// // User-defined types (keep the name)
	// return name
}

// generateTypes collects all non-built-in object and interface types from the
// schema (excluding Query, Mutation, Subscription), converts them to TypeDef
// structs with Go field metadata, and writes types/types.go via the types template.
func (g *Generator) generateTypes() error {
	typeDefMap := make(map[string]TypeDef)

	for _, def := range g.schema.Types {
		if def.BuiltIn || strings.HasPrefix(def.Name, "__") {
			continue
		}

		// Skip Query, Mutation, Subscription root types
		if def.Kind == ast.Object && (def.Name == "Query" || def.Name == "Mutation" || def.Name == "Subscription") {
			continue
		}

		if def.Kind == ast.Object || def.Kind == ast.Interface {
			typeDef := TypeDef{
				Name:        def.Name,
				Description: def.Description,
				IsInterface: def.Kind == ast.Interface,
			}

			// Interfaces only need the Is<Name>() method, not fields
			if def.Kind == ast.Interface {
				typeDefMap[def.Name] = typeDef
				continue
			}

			for _, field := range def.Fields {
				goType := g.graphQLToGoType(field.Type)
				omitempty := !field.Type.NonNull
				jsonName := field.Name // GraphQL field names are already camelCase

				var jsonTag string
				if omitempty {
					jsonTag = fmt.Sprintf("`json:\"%s,omitempty\"`", jsonName)
				} else {
					jsonTag = fmt.Sprintf("`json:\"%s\"`", jsonName)
				}

				fieldDef := FieldDef{
					Name:        util.ToPascalCase(field.Name),
					Description: field.Description,
					GoType:      goType,
					JSONTag:     jsonTag,
					OmitEmpty:   omitempty,
				}

				typeDef.Fields = append(typeDef.Fields, fieldDef)
			}

			typeDefMap[def.Name] = typeDef
		}
	}

	// Convert map to sorted slice for deterministic output
	typeList := make([]TypeDef, 0, len(typeDefMap))
	for _, typeDef := range typeDefMap {
		typeList = append(typeList, typeDef)
	}
	sort.Slice(typeList, func(i, j int) bool {
		return typeList[i].Name < typeList[j].Name
	})

	// Collect imports
	imports := g.collectTypeImports()

	// goutil.PrintToJSON(imports)

	// goutil.PrintToJSON(g.clientConfig.Bindings)

	b := bytes.NewBuffer(nil)
	err := g.templates.ExecuteTemplate(b, "types", map[string]interface{}{
		"Config":  g.config,
		"Types":   typeList,
		"Imports": imports,
		"Package": "types",
	})
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}
	content := b.String()

	return g.writer.WriteFile("types/types.go", content)
}

// collectTypeImports collects necessary imports for types
func (g *Generator) collectTypeImports() []string {
	imports := make(map[string]bool)

	for _, def := range g.schema.Types {
		if def.BuiltIn || strings.HasPrefix(def.Name, "__") {
			continue
		}

		if def.Kind == ast.Object || def.Kind == ast.Interface {
			for _, field := range def.Fields {
				g.checkTypeForImports(field.Type, imports)
			}
		}
	}

	// Convert to sorted slice
	importList := make([]string, 0, len(imports))
	for imp := range imports {
		importList = append(importList, imp)
	}
	sort.Strings(importList)

	return importList
}

// checkTypeForImports checks a type and adds necessary imports
func (g *Generator) checkTypeForImports(t *ast.Type, imports map[string]bool) {
	if t.Elem != nil {
		g.checkTypeForImports(t.Elem, imports)
		return
	}

	// Check if the type has a custom binding with an import
	if entry, ok := g.clientConfig.Bindings[t.NamedType]; ok {
		// Use GoImport from typegql if available
		if entry.GoImport != "" {
			imports[entry.GoImport] = true
			return
		}
		// // Fallback: check Model for standard library types
		// if strings.Contains(entry.Model, "time.Time") {
		// 	imports["time"] = true
		// } else if strings.Contains(entry.Model, "json.RawMessage") {
		// 	imports["encoding/json"] = true
		// }
		return
	}

	if g.schema.Types[t.NamedType].Kind == ast.Scalar {
		imports[g.config.Package+"/scalars"] = true
		return
	}

	fmt.Println("Enum: ", t.NamedType)
	if g.schema.Types[t.NamedType].Kind == ast.Enum {
		imports[g.config.Package+"/enums"] = true
		return
	}

	// Check built-in types that need imports
	// switch t.NamedType {
	// case "Time", "DateTime", "Date":
	// 	imports["time"] = true
	// case "JSON":
	// 	imports["encoding/json"] = true
	// }
}
