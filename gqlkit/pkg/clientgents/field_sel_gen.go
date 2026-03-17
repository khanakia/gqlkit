package clientgents

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
)

// TSFieldSelectorData is the template data for generating a single TypeScript
// field selector class. Each GraphQL object type gets its own selector class
// that extends FieldSelection with typed methods for each field.
type TSFieldSelectorData struct {
	TypeName      string
	SelectorName  string
	Fields        []TSFieldSelField
	Imports       []TSFieldImport
	EnumImports   []string
	ScalarImports []string
}

// TSFieldSelField describes one field on a TypeScript field selector class.
// Scalar fields produce addField calls; object fields produce nested selector
// callbacks with the appropriate generic type parameter.
type TSFieldSelField struct {
	FieldName      string // original GraphQL field name (camelCase)
	MethodName     string // camelCase method name
	IsObject       bool
	NestedSelector string // e.g., "UserFields"
	TSType         string // TypeScript type for scalar fields (e.g., "string", "number", "string[]")
	IsNullable     bool   // whether the field is nullable (optional property)
	IsList         bool   // for object fields: wraps U in U[]
}

// TSFieldImport describes one import of another field selector class needed
// for nested object selection methods.
type TSFieldImport struct {
	ClassName string // e.g., "UserFields"
	FilePath  string // e.g., "./user"
}

// TSTypeImport describes a type-only import (e.g. enum or scalar types)
// needed for field selector files.
type TSTypeImport struct {
	Names []string // e.g., ["Role", "Status"]
	Path  string   // e.g., "../enums"
}

// TSFieldIndexEntry holds one entry for the fields/index.ts barrel file that
// re-exports all field selector classes.
type TSFieldIndexEntry struct {
	SelectorName string
	FileName     string
}

// buildFieldSelectorData builds template data for one object type's field selector
func (g *Generator) buildFieldSelectorData(def *ast.Definition) TSFieldSelectorData {
	data := TSFieldSelectorData{
		TypeName:     def.Name,
		SelectorName: def.Name + "Fields",
		Fields:       make([]TSFieldSelField, 0, len(def.Fields)),
	}

	importMap := make(map[string]TSFieldImport) // keyed by NestedSelector

	for _, field := range def.Fields {
		if strings.HasPrefix(field.Name, "__") {
			continue
		}
		baseName := getBaseTypeName(field.Type)
		isObj := isObjectType(g.schema, field.Type) && baseName != def.Name
		isNullable := !field.Type.NonNull
		isList := field.Type.Elem != nil

		f := TSFieldSelField{
			FieldName:  field.Name,
			MethodName: field.Name,
			IsObject:   isObj,
			IsNullable: isNullable,
			IsList:     isList,
		}
		if isObj {
			f.NestedSelector = baseName + "Fields"
			filePath := "./" + toKebabCase(baseName)
			importMap[f.NestedSelector] = TSFieldImport{
				ClassName: f.NestedSelector,
				FilePath:  filePath,
			}
		} else {
			f.TSType = g.fieldTSType(field.Type)
		}
		data.Fields = append(data.Fields, f)
	}

	// Sort imports by class name for deterministic output
	for _, imp := range importMap {
		data.Imports = append(data.Imports, imp)
	}
	sort.Slice(data.Imports, func(i, j int) bool {
		return data.Imports[i].ClassName < data.Imports[j].ClassName
	})

	// Collect enum and custom scalar imports
	enumImports, scalarImports := g.collectFieldSelectorTypeImports(def)
	data.EnumImports = enumImports
	data.ScalarImports = scalarImports

	return data
}

// generateFieldSelectionFiles generates one TS file per object type in the
// fields/ directory, plus a barrel index.ts that re-exports all selectors.
func (g *Generator) generateFieldSelectionFiles() error {
	names := g.getSortedObjectTypeNames()

	var entries []TSFieldIndexEntry

	for _, name := range names {
		def := g.schema.Types[name]
		if def == nil {
			continue
		}
		data := g.buildFieldSelectorData(def)

		var buf bytes.Buffer
		if err := g.templates.ExecuteTemplate(&buf, "ts_field_selector.tmpl", data); err != nil {
			return fmt.Errorf("failed to execute field selector template for %s: %w", name, err)
		}

		fileName := toKebabCase(name) + ".ts"
		if err := g.writer.WriteFile("fields/"+fileName, buf.String()); err != nil {
			return fmt.Errorf("write fields/%s: %w", fileName, err)
		}

		entries = append(entries, TSFieldIndexEntry{
			SelectorName: data.SelectorName,
			FileName:     toKebabCase(name),
		})
	}

	// Generate fields/index.ts barrel file
	var buf bytes.Buffer
	if err := g.templates.ExecuteTemplate(&buf, "ts_field_selector_index.tmpl", entries); err != nil {
		return fmt.Errorf("failed to execute field selector index template: %w", err)
	}

	return g.writer.WriteFile("fields/index.ts", buf.String())
}
