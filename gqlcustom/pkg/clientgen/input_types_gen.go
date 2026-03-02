package clientgen

import (
	"bytes"
	"fmt"
	"gqlcustom/pkg/util"
	"sort"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
)

type InputTypeDef struct {
	Name        string
	Description string
	Fields      []FieldDef
	IsInterface bool
}

type InputFieldDef struct {
	Name        string
	Description string
	GoType      string
	JSONTag     string
	OmitEmpty   bool
}

type InputTypeDefList []InputTypeDef

func (g *Generator) generateInputTypes() error {
	typeDefMap := make(map[string]TypeDef)

	for _, def := range g.schema.Types {
		if def.BuiltIn || strings.HasPrefix(def.Name, "__") {
			continue
		}

		// Skip Query, Mutation, Subscription root types
		if def.Kind == ast.InputObject && (def.Name == "Query" || def.Name == "Mutation" || def.Name == "Subscription") {
			continue
		}

		if def.Kind == ast.InputObject {
			// if def.Name != "AiModelOrder" {
			// 	continue
			// }
			// fmt.Println("InputObject: ", def.Name)

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
				jsonName := util.ToCamelCase(field.Name)

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
	imports := g.collectInputTypeImports()

	// goutil.PrintToJSON(imports)

	// goutil.PrintToJSON(g.clientConfig.Bindings)

	b := bytes.NewBuffer(nil)
	err := g.templates.ExecuteTemplate(b, "inputs", map[string]interface{}{
		"Config":  g.config,
		"Types":   typeList,
		"Imports": imports,
		"Package": "inputs",
	})
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}
	content := b.String()

	return g.writer.WriteFile("inputs/inputs.go", content)
}

// collectTypeImports collects necessary imports for types
func (g *Generator) collectInputTypeImports() []string {
	imports := make(map[string]bool)

	for _, def := range g.schema.Types {
		if def.BuiltIn || strings.HasPrefix(def.Name, "__") {
			continue
		}

		if def.Kind == ast.InputObject {
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
