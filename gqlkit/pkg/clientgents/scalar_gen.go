package clientgents

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
)

// TSScalar holds the metadata for a single TypeScript type alias generated from
// a GraphQL scalar (e.g. "export type DateTime = string;").
type TSScalar struct {
	Name       string
	TSType     string
	Import     string // npm package to import from (empty = inline alias)
	ImportType string // type name to import from the package
}

// generateScalars collects all custom (non-built-in) scalars from the schema,
// resolves their TS types, and writes scalars/index.ts.
func (g *Generator) generateScalars() error {
	var scalars []TSScalar

	for _, def := range g.schema.Types {
		if def.Kind != ast.Scalar {
			continue
		}
		if def.BuiltIn && isGraphQLBuiltIn(def.Name) {
			continue
		}

		tsType := g.tsTypeMap.Get(def.Name)

		// Special case: "String" scalar → "GqlString" to avoid TS conflict
		name := def.Name
		if name == "String" {
			name = "GqlString"
		}

		scalar := TSScalar{
			Name:   name,
			TSType: tsType,
		}

		// If the binding has an external import, carry that through
		if binding, ok := g.clientConfig.Bindings[def.Name]; ok && binding.Import != "" {
			scalar.Import = binding.Import
			scalar.ImportType = binding.Type
		}

		scalars = append(scalars, scalar)
	}

	sort.Slice(scalars, func(i, j int) bool {
		return scalars[i].Name < scalars[j].Name
	})

	var buf bytes.Buffer
	if err := g.templates.ExecuteTemplate(&buf, "ts_scalar.tmpl", scalars); err != nil {
		return fmt.Errorf("failed to execute scalar template: %w", err)
	}

	return g.writer.WriteFile("scalars/index.ts", buf.String())
}

// isGraphQLBuiltIn returns true for the 5 built-in GraphQL scalars that we DON'T skip
// (we actually want to emit them all - they get proper TS types)
func isGraphQLBuiltIn(name string) bool {
	// We don't skip any - all scalars including String, Int, etc. get emitted
	return false
}

// collectScalarImportNames returns the names of custom scalars referenced by
// the given type definitions, needed for import statements from "../scalars".
func (g *Generator) collectScalarImportNames(defs []*ast.Definition) []string {
	seen := make(map[string]bool)
	for _, def := range defs {
		for _, field := range def.Fields {
			g.collectScalarRefs(field.Type, seen)
		}
	}

	var names []string
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// collectScalarRefs recursively walks a GraphQL type and records any custom
// scalar references into the seen map.
func (g *Generator) collectScalarRefs(t *ast.Type, seen map[string]bool) {
	if t == nil {
		return
	}
	if t.Elem != nil {
		g.collectScalarRefs(t.Elem, seen)
		return
	}
	name := t.NamedType
	def := g.schema.Types[name]
	if def == nil {
		return
	}
	if def.Kind == ast.Scalar {
		// Only collect custom scalars (not built-in mapped ones like String→string)
		if _, isBuiltIn := g.tsTypeMap[name]; !isBuiltIn || !isSimpleTSType(g.tsTypeMap[name]) {
			tsName := name
			if tsName == "String" {
				tsName = "GqlString"
			}
			seen[tsName] = true
		}
	}
}

// isSimpleTSType returns true for basic TS types that don't need imports
func isSimpleTSType(t string) bool {
	return t == "string" || t == "number" || t == "boolean" || t == "any"
}

// collectEnumImportNames returns enum type names referenced by the given
// definitions, needed for import statements from "../enums".
func (g *Generator) collectEnumImportNames(defs []*ast.Definition) []string {
	seen := make(map[string]bool)
	for _, def := range defs {
		for _, field := range def.Fields {
			g.collectEnumRefs(field.Type, seen)
		}
	}

	var names []string
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// collectEnumRefs recursively walks a GraphQL type and records any enum
// references into the seen map.
func (g *Generator) collectEnumRefs(t *ast.Type, seen map[string]bool) {
	if t == nil {
		return
	}
	if t.Elem != nil {
		g.collectEnumRefs(t.Elem, seen)
		return
	}
	name := t.NamedType
	def := g.schema.Types[name]
	if def != nil && def.Kind == ast.Enum {
		seen[name] = true
	}
}

// fieldTSType returns the TypeScript type string for a field's GraphQL type.
// Object references use the plain type name (same-file or cross-referenced),
// while scalars use either a direct TS primitive or a custom type alias.
func (g *Generator) fieldTSType(t *ast.Type) string {
	if t == nil {
		return "any"
	}
	if t.Elem != nil {
		return g.fieldTSType(t.Elem) + "[]"
	}

	name := t.NamedType
	def := g.schema.Types[name]
	if def == nil {
		return g.tsTypeMap.Get(name)
	}

	switch def.Kind {
	case ast.Scalar:
		// Built-in scalars → use direct TS type
		if tsType, ok := g.tsTypeMap[name]; ok && isSimpleTSType(tsType) {
			return tsType
		}
		// Custom scalars → reference the scalar type alias
		if name == "String" {
			return "GqlString"
		}
		return name
	case ast.Enum:
		return name
	case ast.Object, ast.Interface, ast.InputObject:
		return name
	}

	return "any"
}

// fieldTSTypeStr builds a full TypeScript interface field declaration string
// like "field?: Type;" or "field: Type;" based on nullability.
func (g *Generator) fieldTSTypeStr(fieldName string, t *ast.Type) string {
	tsType := g.fieldTSType(t)
	if t.Elem != nil {
		// Already handled in fieldTSType
	}

	optional := !t.NonNull
	if optional {
		return fmt.Sprintf("%s?: %s;", fieldName, tsType)
	}
	return fmt.Sprintf("%s: %s;", fieldName, tsType)
}

// isCustomScalarRef returns true if the type references a custom scalar that
// is not a simple TS primitive (string/number/boolean/any).
func (g *Generator) isCustomScalarRef(t *ast.Type) bool {
	if t == nil {
		return false
	}
	if t.Elem != nil {
		return g.isCustomScalarRef(t.Elem)
	}
	name := t.NamedType
	def := g.schema.Types[name]
	if def == nil || def.Kind != ast.Scalar {
		return false
	}
	if tsType, ok := g.tsTypeMap[name]; ok && isSimpleTSType(tsType) {
		return false
	}
	return true
}

// collectFieldSelectorTypeImports collects enum and custom scalar import names
// needed by a field selector file. Scalar fields reference their TS type in
// generic return types, so their imports must be present.
func (g *Generator) collectFieldSelectorTypeImports(def *ast.Definition) (enumImports []string, scalarImports []string) {
	enumSeen := make(map[string]bool)
	scalarSeen := make(map[string]bool)

	for _, field := range def.Fields {
		if strings.HasPrefix(field.Name, "__") {
			continue
		}
		baseName := getBaseTypeName(field.Type)
		// Skip object fields (they use generic U, not a concrete type name)
		if isObjectType(g.schema, field.Type) && baseName != def.Name {
			continue
		}
		g.collectEnumRefs(field.Type, enumSeen)
		g.collectScalarRefs(field.Type, scalarSeen)
	}

	for name := range enumSeen {
		enumImports = append(enumImports, name)
	}
	sort.Strings(enumImports)

	for name := range scalarSeen {
		scalarImports = append(scalarImports, name)
	}
	sort.Strings(scalarImports)
	return
}

// collectTypeEnumImports collects enum type names that need importing for types/index.ts
func (g *Generator) collectTypeEnumImports() []string {
	seen := make(map[string]bool)
	for _, def := range g.schema.Types {
		if def.BuiltIn || strings.HasPrefix(def.Name, "__") {
			continue
		}
		if def.Kind != ast.Object && def.Kind != ast.Interface {
			continue
		}
		if def.Name == "Query" || def.Name == "Mutation" || def.Name == "Subscription" {
			continue
		}
		for _, field := range def.Fields {
			g.collectEnumRefs(field.Type, seen)
		}
	}

	var names []string
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// collectTypeScalarImports collects scalar type names that need importing for types/index.ts
func (g *Generator) collectTypeScalarImports() []string {
	seen := make(map[string]bool)
	for _, def := range g.schema.Types {
		if def.BuiltIn || strings.HasPrefix(def.Name, "__") {
			continue
		}
		if def.Kind != ast.Object && def.Kind != ast.Interface {
			continue
		}
		if def.Name == "Query" || def.Name == "Mutation" || def.Name == "Subscription" {
			continue
		}
		for _, field := range def.Fields {
			g.collectScalarRefs(field.Type, seen)
		}
	}

	var names []string
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
