package clientgents

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
)

// TSInputsData is the template data passed to ts_inputs.tmpl to generate
// inputs/index.ts. It includes enum and scalar import lists needed by the
// input type interfaces.
type TSInputsData struct {
	EnumImports   []string
	ScalarImports []string
	Types         []TSTypeDef
}

// generateInputTypes collects all GraphQL InputObject types, converts them to
// TSTypeDef structs, gathers required imports, and writes inputs/index.ts.
func (g *Generator) generateInputTypes() error {
	var inputs []TSTypeDef

	for _, def := range g.schema.Types {
		if def.BuiltIn || strings.HasPrefix(def.Name, "__") {
			continue
		}
		if def.Kind != ast.InputObject {
			continue
		}

		typeDef := TSTypeDef{
			Name: def.Name,
		}

		for _, field := range def.Fields {
			fieldName := field.Name
			tsType := g.fieldTSType(field.Type)
			optional := !field.Type.NonNull

			typeDef.Fields = append(typeDef.Fields, TSFieldDef{
				Name:     fieldName,
				Optional: optional,
				TSType:   tsType,
			})
		}

		inputs = append(inputs, typeDef)
	}

	sort.Slice(inputs, func(i, j int) bool {
		return inputs[i].Name < inputs[j].Name
	})

	// Collect needed imports
	var inputDefs []*ast.Definition
	for _, def := range g.schema.Types {
		if def.BuiltIn || strings.HasPrefix(def.Name, "__") {
			continue
		}
		if def.Kind == ast.InputObject {
			inputDefs = append(inputDefs, def)
		}
	}

	data := TSInputsData{
		EnumImports:   g.collectEnumImportNames(inputDefs),
		ScalarImports: g.collectScalarImportNames(inputDefs),
		Types:         inputs,
	}

	var buf bytes.Buffer
	if err := g.templates.ExecuteTemplate(&buf, "ts_inputs.tmpl", data); err != nil {
		return fmt.Errorf("failed to execute inputs template: %w", err)
	}

	return g.writer.WriteFile("inputs/index.ts", buf.String())
}
