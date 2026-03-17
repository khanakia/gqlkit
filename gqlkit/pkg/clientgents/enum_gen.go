package clientgents

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
)

// TSEnumDef holds the metadata for a single TypeScript enum generated from a
// GraphQL enum type. Values are the raw enum value strings.
type TSEnumDef struct {
	Name   string
	Values []string
}

// generateEnums collects all non-built-in enum types from the schema, sorts
// them alphabetically, and writes enums/index.ts via the ts_enums template.
func (g *Generator) generateEnums() error {
	var enums []TSEnumDef

	for _, def := range g.schema.Types {
		if def.BuiltIn || strings.HasPrefix(def.Name, "__") {
			continue
		}
		if def.Kind != ast.Enum {
			continue
		}

		enumDef := TSEnumDef{
			Name: def.Name,
		}
		for _, val := range def.EnumValues {
			enumDef.Values = append(enumDef.Values, val.Name)
		}
		enums = append(enums, enumDef)
	}

	sort.Slice(enums, func(i, j int) bool {
		return enums[i].Name < enums[j].Name
	})

	var buf bytes.Buffer
	if err := g.templates.ExecuteTemplate(&buf, "ts_enums.tmpl", enums); err != nil {
		return fmt.Errorf("failed to execute enums template: %w", err)
	}

	return g.writer.WriteFile("enums/index.ts", buf.String())
}
