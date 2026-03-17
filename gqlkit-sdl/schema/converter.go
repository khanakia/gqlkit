package schema

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// ConvertToSDL converts an IntrospectionSchema into a GraphQL SDL (Schema
// Definition Language) string. It outputs the schema definition (if custom root
// types exist), all user-defined types sorted alphabetically, and any custom
// directives. Built-in types (prefixed with "__") and built-in scalars/directives
// are omitted from the output.
func ConvertToSDL(schema *IntrospectionSchema) string {
	var sb strings.Builder

	// Write the explicit schema block only if root types have non-default names.
	writeSchemaDefinition(&sb, schema)

	// Sort types alphabetically by name for deterministic, diff-friendly output.
	types := make([]FullType, len(schema.Types))
	copy(types, schema.Types)
	sort.Slice(types, func(i, j int) bool {
		return types[i].Name < types[j].Name
	})

	// Write each user-defined type, skipping introspection meta-types (e.g., __Type, __Field).
	for _, t := range types {
		if strings.HasPrefix(t.Name, "__") {
			continue
		}

		writeType(&sb, t)
	}

	// Write custom directives, skipping built-in ones (@skip, @include, @deprecated, @specifiedBy).
	for _, d := range schema.Directives {
		if isBuiltInDirective(d.Name) {
			continue
		}
		writeDirective(&sb, d)
	}

	return sb.String()
}

// writeSchemaDefinition writes an explicit `schema { ... }` block only when
// the root operation types use non-standard names (i.e., something other than
// "Query", "Mutation", "Subscription"). If all root types use the default names,
// the schema block is omitted since GraphQL infers them by convention.
func writeSchemaDefinition(sb *strings.Builder, schema *IntrospectionSchema) {
	hasCustomRootTypes := false

	if schema.QueryType != nil && schema.QueryType.Name != "Query" {
		hasCustomRootTypes = true
	}
	if schema.MutationType != nil && schema.MutationType.Name != "Mutation" {
		hasCustomRootTypes = true
	}
	if schema.SubscriptionType != nil && schema.SubscriptionType.Name != "Subscription" {
		hasCustomRootTypes = true
	}

	if hasCustomRootTypes {
		sb.WriteString("schema {\n")
		if schema.QueryType != nil {
			sb.WriteString(fmt.Sprintf("  query: %s\n", schema.QueryType.Name))
		}
		if schema.MutationType != nil {
			sb.WriteString(fmt.Sprintf("  mutation: %s\n", schema.MutationType.Name))
		}
		if schema.SubscriptionType != nil {
			sb.WriteString(fmt.Sprintf("  subscription: %s\n", schema.SubscriptionType.Name))
		}
		sb.WriteString("}\n\n")
	}
}

// writeType dispatches to the appropriate writer based on the GraphQL type kind.
func writeType(sb *strings.Builder, t FullType) {
	switch t.Kind {
	case "SCALAR":
		writeScalar(sb, t)
	case "OBJECT":
		writeObject(sb, t)
	case "INTERFACE":
		writeInterface(sb, t)
	case "UNION":
		writeUnion(sb, t)
	case "ENUM":
		writeEnum(sb, t)
	case "INPUT_OBJECT":
		writeInputObject(sb, t)
	}
}

// writeScalar writes a `scalar <Name>` declaration. Built-in scalars
// (Int, Float, String, Boolean, ID) are skipped.
func writeScalar(sb *strings.Builder, t FullType) {
	if isBuiltInScalar(t.Name) {
		return
	}

	writeDescription(sb, t.Description, "")
	sb.WriteString(fmt.Sprintf("scalar %s\n\n", t.Name))
}

// writeObject writes a `type <Name> [implements ...] { fields }` block,
// including any interfaces the object type implements.
func writeObject(sb *strings.Builder, t FullType) {
	writeDescription(sb, t.Description, "")
	sb.WriteString(fmt.Sprintf("type %s", t.Name))

	// Append "implements X & Y" clause if the type implements interfaces.
	if len(t.Interfaces) > 0 {
		interfaces := make([]string, len(t.Interfaces))
		for i, iface := range t.Interfaces {
			if iface.Name != nil {
				interfaces[i] = *iface.Name
			}
		}
		sb.WriteString(fmt.Sprintf(" implements %s", strings.Join(interfaces, " & ")))
	}

	sb.WriteString(" {\n")
	writeFields(sb, t.Fields)
	sb.WriteString("}\n\n")
}

// writeInterface writes an `interface <Name> [implements ...] { fields }` block.
// Interfaces can implement other interfaces (per the GraphQL spec).
func writeInterface(sb *strings.Builder, t FullType) {
	writeDescription(sb, t.Description, "")
	sb.WriteString(fmt.Sprintf("interface %s", t.Name))

	// Append parent interface implementations if present.
	if len(t.Interfaces) > 0 {
		interfaces := make([]string, len(t.Interfaces))
		for i, iface := range t.Interfaces {
			if iface.Name != nil {
				interfaces[i] = *iface.Name
			}
		}
		sb.WriteString(fmt.Sprintf(" implements %s", strings.Join(interfaces, " & ")))
	}

	sb.WriteString(" {\n")
	writeFields(sb, t.Fields)
	sb.WriteString("}\n\n")
}

// writeUnion writes a `union <Name> = TypeA | TypeB | ...` declaration
// listing all possible member types.
func writeUnion(sb *strings.Builder, t FullType) {
	writeDescription(sb, t.Description, "")

	types := make([]string, len(t.PossibleTypes))
	for i, pt := range t.PossibleTypes {
		if pt.Name != nil {
			types[i] = *pt.Name
		}
	}

	sb.WriteString(fmt.Sprintf("union %s = %s\n\n", t.Name, strings.Join(types, " | ")))
}

// writeEnum writes an `enum <Name> { VALUE1 VALUE2 ... }` block, including
// @deprecated directives on individual enum values when applicable.
func writeEnum(sb *strings.Builder, t FullType) {
	writeDescription(sb, t.Description, "")
	sb.WriteString(fmt.Sprintf("enum %s {\n", t.Name))

	for _, ev := range t.EnumValues {
		writeDescription(sb, ev.Description, "  ")
		sb.WriteString(fmt.Sprintf("  %s", ev.Name))

		// Append @deprecated directive with optional reason.
		if ev.IsDeprecated {
			if ev.DeprecationReason != nil && *ev.DeprecationReason != "" {
				sb.WriteString(fmt.Sprintf(" @deprecated(reason: %q)", *ev.DeprecationReason))
			} else {
				sb.WriteString(" @deprecated")
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("}\n\n")
}

// writeInputObject writes an `input <Name> { field1: Type ... }` block
// for INPUT_OBJECT types, including default values where specified.
func writeInputObject(sb *strings.Builder, t FullType) {
	writeDescription(sb, t.Description, "")
	sb.WriteString(fmt.Sprintf("input %s {\n", t.Name))

	for _, f := range t.InputFields {
		writeDescription(sb, f.Description, "  ")
		sb.WriteString(fmt.Sprintf("  %s: %s", f.Name, formatType(f.Type)))

		// Append default value literal if one is defined.
		if f.DefaultValue != nil {
			sb.WriteString(fmt.Sprintf(" = %s", *f.DefaultValue))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("}\n\n")
}

// writeFields writes a list of field definitions inside a type body. Each field
// includes its arguments (if any), return type, and deprecation status.
func writeFields(sb *strings.Builder, fields []Field) {
	for _, f := range fields {
		writeDescription(sb, f.Description, "  ")
		sb.WriteString(fmt.Sprintf("  %s", f.Name))

		if len(f.Args) > 0 {
			writeArguments(sb, f.Args)
		}

		sb.WriteString(fmt.Sprintf(": %s", formatType(f.Type)))

		// Append @deprecated directive with optional reason.
		if f.IsDeprecated {
			if f.DeprecationReason != nil && *f.DeprecationReason != "" {
				sb.WriteString(fmt.Sprintf(" @deprecated(reason: %q)", *f.DeprecationReason))
			} else {
				sb.WriteString(" @deprecated")
			}
		}
		sb.WriteString("\n")
	}
}

// writeArguments writes a field's argument list in either single-line or
// multi-line format. Multi-line format is used when any argument has a
// description or there are more than 2 arguments, for readability.
func writeArguments(sb *strings.Builder, args []InputValue) {
	if len(args) == 0 {
		return
	}

	// Determine if any argument has a description to decide on formatting.
	hasDescription := false
	for _, arg := range args {
		if arg.Description != nil && *arg.Description != "" {
			hasDescription = true
			break
		}
	}

	if hasDescription || len(args) > 2 {
		// Multi-line format: each argument on its own line with extra indentation.
		sb.WriteString("(\n")
		for _, arg := range args {
			writeDescription(sb, arg.Description, "    ")
			sb.WriteString(fmt.Sprintf("    %s: %s", arg.Name, formatType(arg.Type)))
			if arg.DefaultValue != nil {
				sb.WriteString(fmt.Sprintf(" = %s", *arg.DefaultValue))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("  )")
	} else {
		// Single-line format: compact "(arg1: Type, arg2: Type)" for simple cases.
		sb.WriteString("(")
		argStrs := make([]string, len(args))
		for i, arg := range args {
			argStr := fmt.Sprintf("%s: %s", arg.Name, formatType(arg.Type))
			if arg.DefaultValue != nil {
				argStr += fmt.Sprintf(" = %s", *arg.DefaultValue)
			}
			argStrs[i] = argStr
		}
		sb.WriteString(strings.Join(argStrs, ", "))
		sb.WriteString(")")
	}
}

// writeDirective writes a `directive @<name>(...) on LOCATION | LOCATION` declaration.
func writeDirective(sb *strings.Builder, d Directive) {
	writeDescription(sb, d.Description, "")
	sb.WriteString(fmt.Sprintf("directive @%s", d.Name))

	if len(d.Args) > 0 {
		writeArguments(sb, d.Args)
	}

	if len(d.Locations) > 0 {
		sb.WriteString(" on ")
		sb.WriteString(strings.Join(d.Locations, " | "))
	}

	sb.WriteString("\n\n")
}

// writeDescription writes a GraphQL description string above a definition.
// Single-line descriptions use the `"text"` format; multi-line descriptions
// use the triple-quote `"""..."""` block format. The indent parameter controls
// the indentation level (e.g., "" for top-level, "  " for fields).
func writeDescription(sb *strings.Builder, description *string, indent string) {
	if description == nil || *description == "" {
		return
	}

	desc := *description
	if strings.Contains(desc, "\n") {
		// Multi-line block string format.
		sb.WriteString(fmt.Sprintf("%s\"\"\"\n", indent))
		lines := strings.Split(desc, "\n")
		for _, line := range lines {
			sb.WriteString(fmt.Sprintf("%s%s\n", indent, line))
		}
		sb.WriteString(fmt.Sprintf("%s\"\"\"\n", indent))
	} else {
		// Single-line string literal format.
		sb.WriteString(fmt.Sprintf("%s\"%s\"\n", indent, escapeString(desc)))
	}
}

// formatType recursively converts a TypeInfo wrapper chain into an SDL type
// string. NON_NULL appends "!", LIST wraps in "[]", and named types return
// their name directly. Example: NON_NULL(LIST(NON_NULL(String))) -> "[String!]!"
func formatType(t TypeInfo) string {
	switch t.Kind {
	case "NON_NULL":
		if t.OfType != nil {
			return formatType(*t.OfType) + "!"
		}
	case "LIST":
		if t.OfType != nil {
			return "[" + formatType(*t.OfType) + "]"
		}
	default:
		if t.Name != nil {
			return *t.Name
		}
	}
	return ""
}

// escapeString escapes backslashes and double quotes in a string for use
// inside a GraphQL single-line string literal.
func escapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// isBuiltInScalar returns true if the given type name is one of the five
// built-in GraphQL scalar types (Int, Float, String, Boolean, ID) that
// should not be emitted in SDL output.
func isBuiltInScalar(name string) bool {
	builtInScalars := map[string]bool{
		"Int":     true,
		"Float":   true,
		"String":  true,
		"Boolean": true,
		"ID":      true,
	}
	return builtInScalars[name]
}

// isBuiltInDirective returns true if the given directive name is one of the
// standard built-in GraphQL directives that should not be emitted in SDL output.
func isBuiltInDirective(name string) bool {
	builtInDirectives := map[string]bool{
		"skip":        true,
		"include":     true,
		"deprecated":  true,
		"specifiedBy": true,
	}
	return builtInDirectives[name]
}

// SaveToFile writes the given SDL string to a file at the specified path
// with 0644 permissions. This is the final step in the introspection-to-SDL pipeline.
func SaveToFile(sdl string, filepath string) error {
	return os.WriteFile(filepath, []byte(sdl), 0644)
}
