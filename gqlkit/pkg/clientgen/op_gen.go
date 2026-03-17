package clientgen

import (
	"bytes"
	"fmt"
	"gqlkit/pkg/util"
	"sort"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
)

// ArgumentData describes a single argument on a generated operation builder,
// including its GraphQL name, Go method name, Go type, and the original
// GraphQL type annotation string used in SetArg calls.
type ArgumentData struct {
	ArgName     string
	MethodName  string
	GoType      string
	GraphQLType string
}

// OperationBuilderData holds all the template data needed to generate a single
// query or mutation builder Go file. Each builder embeds BaseBuilder and adds
// typed argument setters, an optional Select method, and an Execute method.
type OperationBuilderData struct {
	BuilderName  string
	OpType       string
	FieldName    string
	Arguments    []ArgumentData
	HasSelect    bool
	SelectorName string
	ReturnType   string
	ZeroValue    string
}

// graphQLOpToGoType converts a GraphQL type to the Go type used in operation
// builder files. Unlike graphQLToGoType, this version prefixes object types
// with "types." and input objects with "inputs." since builders live in
// separate packages from the type definitions.
func (g *Generator) graphQLOpToGoType(t *ast.Type) string {
	if t == nil {
		return "interface{}"
	}

	goType := g.resolveOpType(t)

	// If nullable (not NonNull), make it a pointer (except for slices).
	if !t.NonNull && !strings.HasPrefix(goType, "[]") {
		goType = "*" + goType
	}

	return goType
}

// resolveOpType resolves the base Go type for operation builders.
func (g *Generator) resolveOpType(t *ast.Type) string {
	if t.Elem != nil {
		// It's a list type.
		elemType := g.graphQLOpToGoType(t.Elem)
		return "[]" + elemType
	}

	// Named type.
	name := t.NamedType
	def := g.schema.Types[name]
	if def == nil {
		// Fallback to the generic mapper.
		return g.namedTypeToGo(name)
	}

	switch def.Kind {
	case ast.Object, ast.Interface:
		return "types." + def.Name
	case ast.InputObject:
		return "inputs." + def.Name
	default:
		return g.namedTypeToGo(name)
	}
}

// formatGraphQLType reconstructs the GraphQL type notation string from an AST
// type, e.g. "[AiModelOrder!]" or "Int!". This string is passed to SetArg so
// the builder can include proper variable type declarations in the query.
func formatGraphQLType(t *ast.Type) string {
	if t == nil {
		return ""
	}

	var result string
	if t.Elem != nil {
		result = "[" + formatGraphQLType(t.Elem) + "]"
	} else {
		result = t.NamedType
	}

	if t.NonNull {
		result += "!"
	}

	return result
}

// getZeroValue returns the Go zero value expression for a type string.
// Used in generated Execute methods to provide a default return on error.
func getZeroValue(goType string) string {
	// Pointer, slice, map, interface and alias types default to nil.
	if strings.HasPrefix(goType, "*") ||
		strings.HasPrefix(goType, "[]") ||
		strings.HasPrefix(goType, "map[") ||
		goType == "interface{}" ||
		goType == "any" {
		return "nil"
	}

	switch goType {
	case "string":
		return `""`
	case "bool":
		return "false"
	case "int", "int32", "int64",
		"uint", "uint32", "uint64",
		"float32", "float64":
		return "0"
	}

	// For all other types, use the composite literal.
	return goType + "{}"
}

// buildOperationBuilderData builds template data for a single query/mutation field.
func (g *Generator) buildOperationBuilderData(opType string, field *ast.FieldDefinition) OperationBuilderData {
	returnTypeName := getBaseTypeName(field.Type)
	returnType := g.graphQLOpToGoType(field.Type)
	isObjReturn := isObjectType(g.schema, field.Type)

	// Builder name: AiModelsBuilder or AiModelsMutationBuilder.
	methodName := util.ToPascalCase(field.Name)
	builderName := methodName + "Builder"
	if opType == "mutation" {
		builderName = methodName + "MutationBuilder"
	}

	data := OperationBuilderData{
		BuilderName: builderName,
		OpType:      opType,
		FieldName:   field.Name,
		Arguments:   make([]ArgumentData, 0, len(field.Arguments)),
		HasSelect:   isObjReturn,
		ReturnType:  returnType,
		ZeroValue:   getZeroValue(returnType),
	}

	if isObjReturn {
		data.SelectorName = returnTypeName + "Fields"
	}

	for _, arg := range field.Arguments {
		argData := ArgumentData{
			ArgName:     arg.Name,
			MethodName:  util.ToPascalCase(arg.Name),
			GoType:      g.graphQLOpToGoType(arg.Type),
			GraphQLType: formatGraphQLType(arg.Type),
		}
		data.Arguments = append(data.Arguments, argData)
	}

	return data
}

// collectOperationImports collects imports required for a single operation.
func (g *Generator) collectOperationImports(field *ast.FieldDefinition) []string {
	imports := make(map[string]bool)

	// External or scalar/enum imports based on bindings.
	g.checkTypeForImports(field.Type, imports)
	for _, arg := range field.Arguments {
		g.checkTypeForImports(arg.Type, imports)
	}

	// Ensure deterministic order.
	list := make([]string, 0, len(imports))
	for imp := range imports {
		list = append(list, imp)
	}
	sort.Strings(list)
	return list
}

// generateOperationFiles generates per-field builder files for all queries and
// mutations, plus the root.go entrypoint files that provide factory methods.
func (g *Generator) generateOperationFiles() error {
	if g.schema.Query != nil {
		if err := g.generateOperationFilesForType("query", g.schema.Query); err != nil {
			return err
		}
		if err := g.generateQueryRootFile(); err != nil {
			return err
		}
	}

	if g.schema.Mutation != nil {
		if err := g.generateOperationFilesForType("mutation", g.schema.Mutation); err != nil {
			return err
		}
		if err := g.generateMutationRootFile(); err != nil {
			return err
		}
	}

	return nil
}

// generateOperationFilesForType emits one Go builder file per field on the
// given root type (Query or Mutation). Each file contains a builder struct,
// argument setters, optional Select method, and an Execute method.
func (g *Generator) generateOperationFilesForType(opType string, def *ast.Definition) error {
	if def == nil {
		return nil
	}

	subDir := "queries"
	pkgName := "queries"
	if opType == "mutation" {
		subDir = "mutations"
		pkgName = "mutations"
	}

	for _, field := range def.Fields {
		if strings.HasPrefix(field.Name, "__") {
			continue
		}

		data := g.buildOperationBuilderData(opType, field)

		// Base imports.
		importSet := map[string]bool{
			"context": true,
		}

		// Root package for generated SDK (e.g., gqlkit/sdk).
		rootPkg := strings.TrimSuffix(g.config.Package, "/")
		if rootPkg != "" {
			importSet[rootPkg+"/builder"] = true
		}

		// Add types/inputs packages if referenced.
		if strings.Contains(data.ReturnType, "types.") {
			importSet[rootPkg+"/types"] = true
		}
		if strings.Contains(data.ReturnType, "inputs.") {
			importSet[rootPkg+"/inputs"] = true
		}
		for _, arg := range data.Arguments {
			if strings.Contains(arg.GoType, "types.") {
				importSet[rootPkg+"/types"] = true
			}
			if strings.Contains(arg.GoType, "inputs.") {
				importSet[rootPkg+"/inputs"] = true
			}
		}

		// Fields package only needed when there's a Select.
		if data.HasSelect && rootPkg != "" {
			importSet[rootPkg+"/fields"] = true
		}

		// Imports based on scalar/enum/binding types.
		for _, imp := range g.collectOperationImports(field) {
			importSet[imp] = true
		}

		// Convert imports to sorted slice, excluding "" (defensive).
		var imports []string
		for imp := range importSet {
			if imp == "" {
				continue
			}
			imports = append(imports, imp)
		}
		sort.Strings(imports)

		var buf bytes.Buffer
		if err := g.templates.ExecuteTemplate(&buf, "operation_builder", data); err != nil {
			return fmt.Errorf("operation template for %s %s: %w", opType, field.Name, err)
		}

		var file bytes.Buffer
		file.WriteString("// Code generated by gqlkit. DO NOT EDIT.\n\n")
		fmt.Fprintf(&file, "package %s\n\n", pkgName)
		file.WriteString("import (\n")
		for _, imp := range imports {
			fmt.Fprintf(&file, "\t%q\n", imp)
		}
		file.WriteString(")\n\n")
		file.Write(buf.Bytes())

		filename := fmt.Sprintf("%s/%s_%s.go", subDir, opType, util.ToSnakeCase(field.Name))
		if err := g.writer.WriteFile(filename, file.String()); err != nil {
			return fmt.Errorf("write %s: %w", filename, err)
		}
	}

	return nil
}

// generateQueryRootFile generates queries/root.go containing QueryRoot, a
// struct with factory methods that create individual query builders.
func (g *Generator) generateQueryRootFile() error {
	if g.schema.Query == nil {
		return nil
	}

	rootPkg := strings.TrimSuffix(g.config.Package, "/")

	var sb strings.Builder
	sb.WriteString("// Code generated by gqlkit. DO NOT EDIT.\n\n")
	sb.WriteString("package queries\n\n")
	if rootPkg != "" {
		fmt.Fprintf(&sb, "import %q\n\n", rootPkg+"/builder")
	} else {
		sb.WriteString("import \"gqlkit/sdk/builder\"\n\n")
	}

	sb.WriteString("// QueryRoot is the entry point for queries\n")
	sb.WriteString("type QueryRoot struct {\n")
	sb.WriteString("\tclient builder.GraphQLClient\n")
	sb.WriteString("}\n\n")

	sb.WriteString("// NewQueryRoot creates a new QueryRoot\n")
	sb.WriteString("func NewQueryRoot(client builder.GraphQLClient) *QueryRoot {\n")
	sb.WriteString("\treturn &QueryRoot{client: client}\n")
	sb.WriteString("}\n")

	for _, field := range g.schema.Query.Fields {
		if strings.HasPrefix(field.Name, "__") {
			continue
		}

		methodName := util.ToPascalCase(field.Name)
		builderName := methodName + "Builder"

		sb.WriteString("\n")
		fmt.Fprintf(&sb, "// %s creates a new %s\n", methodName, builderName)
		fmt.Fprintf(&sb, "func (q *QueryRoot) %s() *%s {\n", methodName, builderName)
		fmt.Fprintf(&sb, "\treturn &%s{\n", builderName)
		fmt.Fprintf(&sb, "\t\tBaseBuilder: builder.NewBaseBuilder(q.client, %q, %q, %q),\n", "query", methodName, field.Name)
		sb.WriteString("\t}\n")
		sb.WriteString("}\n")
	}

	return g.writer.WriteFile("queries/root.go", sb.String())
}

// generateMutationRootFile generates mutations/root.go containing MutationRoot,
// a struct with factory methods that create individual mutation builders.
func (g *Generator) generateMutationRootFile() error {
	if g.schema.Mutation == nil {
		return nil
	}

	rootPkg := strings.TrimSuffix(g.config.Package, "/")

	var sb strings.Builder
	sb.WriteString("// Code generated by gqlkit. DO NOT EDIT.\n\n")
	sb.WriteString("package mutations\n\n")
	if rootPkg != "" {
		fmt.Fprintf(&sb, "import %q\n\n", rootPkg+"/builder")
	} else {
		sb.WriteString("import \"gqlkit/sdk/builder\"\n\n")
	}

	sb.WriteString("// MutationRoot is the entry point for mutations\n")
	sb.WriteString("type MutationRoot struct {\n")
	sb.WriteString("\tclient builder.GraphQLClient\n")
	sb.WriteString("}\n\n")

	sb.WriteString("// NewMutationRoot creates a new MutationRoot\n")
	sb.WriteString("func NewMutationRoot(client builder.GraphQLClient) *MutationRoot {\n")
	sb.WriteString("\treturn &MutationRoot{client: client}\n")
	sb.WriteString("}\n")

	for _, field := range g.schema.Mutation.Fields {
		if strings.HasPrefix(field.Name, "__") {
			continue
		}

		methodName := util.ToPascalCase(field.Name)
		builderName := methodName + "MutationBuilder"

		sb.WriteString("\n")
		fmt.Fprintf(&sb, "// %s creates a new %s\n", methodName, builderName)
		fmt.Fprintf(&sb, "func (m *MutationRoot) %s() *%s {\n", methodName, builderName)
		fmt.Fprintf(&sb, "\treturn &%s{\n", builderName)
		fmt.Fprintf(&sb, "\t\tBaseBuilder: builder.NewBaseBuilder(m.client, %q, %q, %q),\n", "mutation", methodName, field.Name)
		sb.WriteString("\t}\n")
		sb.WriteString("}\n")
	}

	return g.writer.WriteFile("mutations/root.go", sb.String())
}
