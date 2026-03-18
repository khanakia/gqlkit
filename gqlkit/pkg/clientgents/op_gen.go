package clientgents

import (
	"bytes"
	"fmt"
	"github.com/khanakia/gqlkit/gqlkit/pkg/util"
	"sort"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
)

// TSArgumentData describes a single argument on a TypeScript operation builder,
// including the original GraphQL name, the TS method name, and the TS/GraphQL
// type strings.
type TSArgumentData struct {
	ArgName     string // original GraphQL arg name
	MethodName  string // camelCase method name
	TSType      string // TypeScript type for the arg
	GraphQLType string // e.g., "Cursor", "Int", "[ChatbotOrder!]"
}

// TSOperationBuilderData holds all the template data for generating a single
// TypeScript query or mutation builder class. Each builder extends BaseBuilder
// and adds typed argument setters, an optional select method, and an execute method.
type TSOperationBuilderData struct {
	BuilderName      string
	OpType           string // "query" or "mutation"
	FieldName        string // original GraphQL field name
	MethodName       string // PascalCase operation name
	Arguments        []TSArgumentData
	HasSelect        bool
	SelectorName     string // e.g., "ChatbotConnectionFields"
	ReturnType       string // TypeScript return type (full, for default generic)
	DefaultGeneric   string // Default generic parameter value (e.g., "TodoConnection", "Todo[]", "Todo | null")
	SelectWrapper    string // How T is wrapped in select return (e.g., "T", "T[]", "T | null")
	IsListReturn     bool   // Whether the return type is a list
	IsNullableReturn bool   // Whether the return type is nullable
}

// TSOperationImport describes a single import statement needed in an operation
// builder file, with optional type-only flag for TS "import type" syntax.
type TSOperationImport struct {
	Names    []string
	Path     string
	TypeOnly bool
}

// TSOperationTemplateData wraps the builder data and its imports for passing
// to the ts_operation.tmpl template.
type TSOperationTemplateData struct {
	Data    TSOperationBuilderData
	Imports []TSOperationImport
}

// TSOperationRootBuilder holds the data for one builder factory method in a
// root file (QueryRoot or MutationRoot).
type TSOperationRootBuilder struct {
	BuilderName string
	FieldName   string
	FileName    string
}

// TSOperationRootData is the template data for generating queries/root.ts or
// mutations/root.ts, which contain the root class with factory methods.
type TSOperationRootData struct {
	ClassName string
	OpType    string
	Builders  []TSOperationRootBuilder
}

// TSOperationIndexData is the template data for generating the barrel
// index.ts file that re-exports the root class and all individual builders.
type TSOperationIndexData struct {
	RootClassName string
	Builders      []TSOperationRootBuilder
}

// buildOperationBuilderData builds template data for a single query/mutation field.
func (g *Generator) buildOperationBuilderData(opType string, field *ast.FieldDefinition) TSOperationBuilderData {
	returnTypeName := getBaseTypeName(field.Type)
	isObjReturn := isObjectType(g.schema, field.Type)

	methodName := util.ToPascalCase(field.Name)
	builderName := methodName + "Builder"
	if opType == "mutation" {
		builderName = methodName + "MutationBuilder"
	}

	returnType := g.graphQLToTSArgType(field.Type)

	isListReturn := field.Type.Elem != nil
	isNullableReturn := !field.Type.NonNull

	defaultGeneric := returnType
	if isNullableReturn && isObjReturn {
		defaultGeneric = returnType + " | null"
	}

	selectWrapper := "T"
	if isListReturn && isNullableReturn {
		selectWrapper = "T[] | null"
	} else if isListReturn {
		selectWrapper = "T[]"
	} else if isNullableReturn {
		selectWrapper = "T | null"
	}

	data := TSOperationBuilderData{
		BuilderName:      builderName,
		OpType:           opType,
		FieldName:        field.Name,
		MethodName:       methodName,
		Arguments:        make([]TSArgumentData, 0, len(field.Arguments)),
		HasSelect:        isObjReturn,
		ReturnType:       returnType,
		DefaultGeneric:   defaultGeneric,
		SelectWrapper:    selectWrapper,
		IsListReturn:     isListReturn,
		IsNullableReturn: isNullableReturn,
	}

	if isObjReturn {
		data.SelectorName = returnTypeName + "Fields"
	}

	for _, arg := range field.Arguments {
		argData := TSArgumentData{
			ArgName:     arg.Name,
			MethodName:  arg.Name,
			TSType:      g.graphQLToTSArgType(arg.Type),
			GraphQLType: formatGraphQLType(arg.Type),
		}
		data.Arguments = append(data.Arguments, argData)
	}

	return data
}

// generateOperationFiles generates per-field builders for all queries and mutations.
func (g *Generator) generateOperationFiles() error {
	if g.schema.Query != nil {
		if err := g.generateOperationFilesForType("query", g.schema.Query); err != nil {
			return err
		}
		if err := g.generateQueryRootFile(); err != nil {
			return err
		}
		if err := g.generateQueryIndexFile(); err != nil {
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
		if err := g.generateMutationIndexFile(); err != nil {
			return err
		}
	}

	return nil
}

// generateOperationFilesForType emits builders for all fields of Query or Mutation.
func (g *Generator) generateOperationFilesForType(opType string, def *ast.Definition) error {
	if def == nil {
		return nil
	}

	subDir := "queries"
	if opType == "mutation" {
		subDir = "mutations"
	}

	for _, field := range def.Fields {
		if strings.HasPrefix(field.Name, "__") {
			continue
		}

		data := g.buildOperationBuilderData(opType, field)
		imports := g.collectOperationImports(data, field)

		tmplData := TSOperationTemplateData{
			Data:    data,
			Imports: imports,
		}

		var buf bytes.Buffer
		if err := g.templates.ExecuteTemplate(&buf, "ts_operation.tmpl", tmplData); err != nil {
			return fmt.Errorf("failed to execute operation template for %s: %w", field.Name, err)
		}

		fileName := toKebabCase(field.Name) + ".ts"
		if err := g.writer.WriteFile(subDir+"/"+fileName, buf.String()); err != nil {
			return fmt.Errorf("write %s/%s: %w", subDir, fileName, err)
		}
	}

	return nil
}

// collectOperationImports gathers all TypeScript import statements needed for
// a single operation builder file, including builder base, field selectors,
// types, scalars, enums, and input types.
func (g *Generator) collectOperationImports(data TSOperationBuilderData, field *ast.FieldDefinition) []TSOperationImport {
	var imports []TSOperationImport

	builderNames := []string{"BaseBuilder", "GraphQLClient"}
	imports = append(imports, TSOperationImport{
		Names: builderNames,
		Path:  "../builder",
	})

	if data.HasSelect {
		imports = append(imports, TSOperationImport{
			Names: []string{data.SelectorName},
			Path:  "../fields",
		})
	}

	typeImports := make(map[string]bool)
	scalarImports := make(map[string]bool)
	enumImports := make(map[string]bool)
	inputImports := make(map[string]bool)

	g.classifyTypeImport(field.Type, typeImports, scalarImports, enumImports, inputImports)

	for _, arg := range field.Arguments {
		g.classifyTypeImport(arg.Type, typeImports, scalarImports, enumImports, inputImports)
	}

	if len(typeImports) > 0 {
		names := sortedKeys(typeImports)
		imports = append(imports, TSOperationImport{
			Names:    names,
			Path:     "../types",
			TypeOnly: true,
		})
	}

	if len(scalarImports) > 0 {
		names := sortedKeys(scalarImports)
		imports = append(imports, TSOperationImport{
			Names:    names,
			Path:     "../scalars",
			TypeOnly: true,
		})
	}

	if len(enumImports) > 0 {
		names := sortedKeys(enumImports)
		imports = append(imports, TSOperationImport{
			Names:    names,
			Path:     "../enums",
			TypeOnly: true,
		})
	}

	if len(inputImports) > 0 {
		names := sortedKeys(inputImports)
		imports = append(imports, TSOperationImport{
			Names:    names,
			Path:     "../inputs",
			TypeOnly: true,
		})
	}

	return imports
}

// classifyTypeImport recursively walks a GraphQL type and sorts each referenced
// named type into the correct import bucket (types, scalars, enums, or inputs).
func (g *Generator) classifyTypeImport(t *ast.Type, types, scalars, enums, inputs map[string]bool) {
	if t == nil {
		return
	}
	if t.Elem != nil {
		g.classifyTypeImport(t.Elem, types, scalars, enums, inputs)
		return
	}

	name := t.NamedType
	def := g.schema.Types[name]
	if def == nil {
		return
	}

	switch def.Kind {
	case ast.Object, ast.Interface:
		if name != "Query" && name != "Mutation" && name != "Subscription" {
			types[name] = true
		}
	case ast.Scalar:
		if tsType, ok := g.tsTypeMap[name]; ok && isSimpleTSType(tsType) {
			return
		}
		tsName := name
		if tsName == "String" {
			tsName = "GqlString"
		}
		scalars[tsName] = true
	case ast.Enum:
		enums[name] = true
	case ast.InputObject:
		inputs[name] = true
	}
}

// buildOperationRootBuilders builds the list of builder entries for the root
// and index files, one per field on the Query or Mutation type.
func (g *Generator) buildOperationRootBuilders(opType string, def *ast.Definition) []TSOperationRootBuilder {
	var builders []TSOperationRootBuilder
	suffix := "Builder"
	if opType == "mutation" {
		suffix = "MutationBuilder"
	}

	for _, field := range def.Fields {
		if strings.HasPrefix(field.Name, "__") {
			continue
		}
		methodName := util.ToPascalCase(field.Name)
		builders = append(builders, TSOperationRootBuilder{
			BuilderName: methodName + suffix,
			FieldName:   field.Name,
			FileName:    toKebabCase(field.Name),
		})
	}
	return builders
}

// generateQueryRootFile generates queries/root.ts
func (g *Generator) generateQueryRootFile() error {
	if g.schema.Query == nil {
		return nil
	}

	data := TSOperationRootData{
		ClassName: "QueryRoot",
		OpType:    "query",
		Builders:  g.buildOperationRootBuilders("query", g.schema.Query),
	}

	var buf bytes.Buffer
	if err := g.templates.ExecuteTemplate(&buf, "ts_operation_root.tmpl", data); err != nil {
		return fmt.Errorf("failed to execute query root template: %w", err)
	}

	return g.writer.WriteFile("queries/root.ts", buf.String())
}

// generateMutationRootFile generates mutations/root.ts
func (g *Generator) generateMutationRootFile() error {
	if g.schema.Mutation == nil {
		return nil
	}

	data := TSOperationRootData{
		ClassName: "MutationRoot",
		OpType:    "mutation",
		Builders:  g.buildOperationRootBuilders("mutation", g.schema.Mutation),
	}

	var buf bytes.Buffer
	if err := g.templates.ExecuteTemplate(&buf, "ts_operation_root.tmpl", data); err != nil {
		return fmt.Errorf("failed to execute mutation root template: %w", err)
	}

	return g.writer.WriteFile("mutations/root.ts", buf.String())
}

// generateQueryIndexFile generates queries/index.ts barrel file
func (g *Generator) generateQueryIndexFile() error {
	if g.schema.Query == nil {
		return nil
	}

	data := TSOperationIndexData{
		RootClassName: "QueryRoot",
		Builders:      g.buildOperationRootBuilders("query", g.schema.Query),
	}

	var buf bytes.Buffer
	if err := g.templates.ExecuteTemplate(&buf, "ts_operation_index.tmpl", data); err != nil {
		return fmt.Errorf("failed to execute query index template: %w", err)
	}

	return g.writer.WriteFile("queries/index.ts", buf.String())
}

// generateMutationIndexFile generates mutations/index.ts barrel file
func (g *Generator) generateMutationIndexFile() error {
	if g.schema.Mutation == nil {
		return nil
	}

	data := TSOperationIndexData{
		RootClassName: "MutationRoot",
		Builders:      g.buildOperationRootBuilders("mutation", g.schema.Mutation),
	}

	var buf bytes.Buffer
	if err := g.templates.ExecuteTemplate(&buf, "ts_operation_index.tmpl", data); err != nil {
		return fmt.Errorf("failed to execute mutation index template: %w", err)
	}

	return g.writer.WriteFile("mutations/index.ts", buf.String())
}

// sortedKeys returns the keys of a map[string]bool in alphabetical order.
func sortedKeys(m map[string]bool) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
