package clientgents

import (
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
)

// TSTypeMap maps GraphQL scalar names to their TypeScript type equivalents
// (e.g. "Int" -> "number", "Boolean" -> "boolean"). Custom scalars can be
// added via config bindings.
type TSTypeMap map[string]string

// BuiltInTSTypes returns the default mapping from GraphQL built-in and common
// extended scalars to TypeScript primitive types.
func BuiltInTSTypes() TSTypeMap {
	return TSTypeMap{
		"String":  "string",
		"Int":     "number",
		"Int64":   "number",
		"Int32":   "number",
		"Float":   "number",
		"Float64": "number",
		"Float32": "number",
		"Boolean": "boolean",
		"ID":      "string",
		"Uint":    "number",
		"Uint64":  "number",
		"Uint32":  "number",
	}
}

// ConfigTSBindings holds custom scalar-to-TypeScript type overrides loaded
// from the config.jsonc file.
type ConfigTSBindings map[string]string

// Merge merges config bindings into the type map.
func (m TSTypeMap) Merge(bindings ConfigTSBindings) TSTypeMap {
	for k, v := range bindings {
		m[k] = v
	}
	return m
}

// Get returns the TS type for a GraphQL scalar name, defaulting to "any".
func (m TSTypeMap) Get(name string) string {
	if t, ok := m[name]; ok {
		return t
	}
	return "any"
}

// graphQLToTSType converts a GraphQL AST type to its TypeScript equivalent.
// Returns the TS type string and whether the field is optional (nullable).
func (g *Generator) graphQLToTSType(t *ast.Type) (string, bool) {
	if t == nil {
		return "any", true
	}

	tsType := g.resolveTSType(t)
	isOptional := !t.NonNull

	return tsType, isOptional
}

// resolveTSType resolves the base TS type from a GraphQL type.
func (g *Generator) resolveTSType(t *ast.Type) string {
	if t.Elem != nil {
		elemType, _ := g.graphQLToTSType(t.Elem)
		return elemType + "[]"
	}
	return g.namedTypeToTS(t.NamedType)
}

// namedTypeToTS converts a named GraphQL type to TypeScript.
func (g *Generator) namedTypeToTS(name string) string {
	// Check custom binding first
	if t, ok := g.tsTypeMap[name]; ok {
		return t
	}

	def := g.schema.Types[name]
	if def == nil {
		return "any"
	}

	switch def.Kind {
	case ast.Scalar:
		return def.Name // Will reference the scalars type alias
	case ast.Enum:
		return def.Name
	case ast.Object, ast.Interface:
		return def.Name
	case ast.InputObject:
		return def.Name
	}

	return "any"
}

// graphQLToTSArgType converts a GraphQL argument type to its TypeScript
// equivalent for use in operation builder files. Unlike graphQLToTSType, this
// is used for builder method parameters where all types are referenced by name.
func (g *Generator) graphQLToTSArgType(t *ast.Type) string {
	if t == nil {
		return "any"
	}
	return g.resolveTSArgType(t)
}

// resolveTSArgType resolves the TS type for operation builder arguments.
func (g *Generator) resolveTSArgType(t *ast.Type) string {
	if t.Elem != nil {
		elemType := g.graphQLToTSArgType(t.Elem)
		return elemType + "[]"
	}

	name := t.NamedType
	def := g.schema.Types[name]
	if def == nil {
		return g.namedTypeToTS(name)
	}

	switch def.Kind {
	case ast.Object, ast.Interface:
		return def.Name
	case ast.InputObject:
		return def.Name
	case ast.Scalar:
		return g.tsTypeMap.Get(name)
	case ast.Enum:
		return def.Name
	}

	return "any"
}

// toKebabCase converts PascalCase to kebab-case for TypeScript file naming
// (e.g. "ChatbotConnection" -> "chatbot-connection"). Handles acronyms
// gracefully so "URLParser" becomes "url-parser" not "u-r-l-parser".
func toKebabCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			// Look ahead to handle acronyms like "URL" → "url" not "u-r-l"
			if i+1 < len(s) && s[i+1] >= 'a' && s[i+1] <= 'z' {
				b.WriteByte('-')
			} else if i > 0 && s[i-1] >= 'a' && s[i-1] <= 'z' {
				b.WriteByte('-')
			}
		}
		b.WriteRune(r)
	}
	return strings.ToLower(b.String())
}
