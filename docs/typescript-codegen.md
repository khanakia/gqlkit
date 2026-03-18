# TypeScript Code Generation — Technical Reference

This document is the full technical reference for the gqlkit TypeScript SDK code generator. It covers architecture, configuration, type mapping, every generation module, templates, the runtime library, CLI usage, and the generated output structure.

---

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Pipeline Stages](#pipeline-stages)
- [CLI Usage](#cli-usage)
- [Configuration](#configuration)
  - [Generator Config (Go flags)](#generator-config-go-flags)
  - [Client Config (config.jsonc)](#client-config-configjsonc)
  - [Scalar Bindings](#scalar-bindings)
  - [External Package Imports](#external-package-imports)
- [Type Mapping System](#type-mapping-system)
  - [TSTypeMap](#tstypemap)
  - [Built-in Type Mappings](#built-in-type-mappings)
  - [Type Resolution Functions](#type-resolution-functions)
  - [Import Classification](#import-classification)
- [Generation Modules](#generation-modules)
  - [Builder Index](#1-builder-index)
  - [Scalars](#2-scalars)
  - [Enums](#3-enums)
  - [Types (Interfaces)](#4-types-interfaces)
  - [Input Types](#5-input-types)
  - [Field Selectors](#6-field-selectors)
  - [Operation Builders](#7-operation-builders)
- [Templates](#templates)
- [Runtime Library (gqlkit-ts)](#runtime-library-gqlkit-ts)
  - [FieldSelection](#fieldselection)
  - [BaseBuilder](#basebuilder)
  - [GraphQLClient](#graphqlclient)
  - [GraphQLErrors](#graphqlerrors)
- [Generated Output Structure](#generated-output-structure)
- [Data Flow Walkthrough](#data-flow-walkthrough)
- [Design Patterns](#design-patterns)

---

## Architecture Overview

```
                     ┌─────────────────────┐
                     │  GraphQL Schema SDL  │
                     └─────────┬───────────┘
                               │
                     ┌─────────▼───────────┐
                     │  gqlparser (AST)     │
                     └─────────┬───────────┘
                               │
  ┌──────────────┐   ┌────────▼─────────┐   ┌──────────────────┐
  │ config.jsonc │──▶│    Generator      │◀──│  9 Go Templates  │
  │ (bindings)   │   │  (clientgents)    │   │  (embed.FS)      │
  └──────────────┘   └────────┬─────────┘   └──────────────────┘
                               │
                     ┌─────────▼───────────┐
                     │   TSWriter           │
                     │   (file I/O)         │
                     └─────────┬───────────┘
                               │
              ┌────────────────▼────────────────┐
              │     Generated TypeScript SDK     │
              │                                  │
              │  sdk/                             │
              │  ├── builder/index.ts             │
              │  ├── scalars/index.ts             │
              │  ├── enums/index.ts               │
              │  ├── types/index.ts               │
              │  ├── inputs/index.ts              │
              │  ├── fields/*.ts + index.ts       │
              │  ├── queries/*.ts + root + index  │
              │  └── mutations/*.ts + root + index│
              └─────────────┬───────────────────┘
                            │
                  ┌─────────▼───────────┐
                  │  gqlkit-ts runtime   │
                  │  (npm package)       │
                  └─────────────────────┘
```

The system is implemented in the Go package `gqlkit/pkg/clientgents`. It reads a GraphQL schema SDL file and a JSONC configuration file, then generates a structured TypeScript SDK organized into functional modules. The generated code depends on the `gqlkit-ts` npm package as a runtime library.

### Source Files

| File | Purpose |
|------|---------|
| `clientgents.go` | Generator struct, `New()` constructor, `Generate()` orchestrator |
| `config.go` | Config/ClientConfig structs, JSONC loading |
| `ts_type_map.go` | TSTypeMap, TSBinding, type resolution functions |
| `scalar_gen.go` | Scalar type aliases, import collection helpers |
| `enum_gen.go` | Enum generation |
| `types_gen.go` | Interface generation for Object/Interface types |
| `input_types_gen.go` | Interface generation for InputObject types |
| `field_sel_gen.go` | Field selector class generation |
| `op_gen.go` | Operation builder (query/mutation) generation |
| `writer.go` | TSWriter file I/O |
| `errors.go` | Sentinel errors |
| `template/*.tmpl` | 9 Go text/template files |

---

## Pipeline Stages

The `Generate()` method runs these steps in order:

| Step | Method | Output |
|------|--------|--------|
| 1 | `generateBuilderIndex()` | `builder/index.ts` |
| 2 | `generateScalars()` | `scalars/index.ts` |
| 3 | `generateEnums()` | `enums/index.ts` |
| 4 | `generateTypes()` | `types/index.ts` |
| 5 | `generateInputTypes()` | `inputs/index.ts` |
| 6 | `generateFieldSelectionFiles()` | `fields/*.ts` + `fields/index.ts` |
| 7 | `generateOperationFiles()` | `queries/*.ts`, `queries/root.ts`, `queries/index.ts`, `mutations/*.ts`, `mutations/root.ts`, `mutations/index.ts` |

Each step walks the parsed GraphQL AST, builds template data structs, executes the appropriate template, and writes the result to disk via `TSWriter`.

---

## CLI Usage

```bash
gqlkit generate-ts \
  --schema path/to/schema.graphql \
  --config path/to/config.jsonc \
  --output ./sdk
```

| Flag | Short | Required | Default | Description |
|------|-------|----------|---------|-------------|
| `--schema` | `-s` | Yes | — | Path to GraphQL SDL file |
| `--config` | `-c` | Yes | — | Path to config.jsonc file |
| `--output` | `-o` | No | `./sdk` | Output directory for generated SDK |

### Programmatic Usage

```go
config := &clientgents.Config{
    SchemaPath: "schema.graphql",
    OutputDir:  "./sdk",
    ConfigPath: "config.jsonc",
}

gen, err := clientgents.New(config)
if err != nil {
    log.Fatal(err)
}

if err := gen.Generate(); err != nil {
    log.Fatal(err)
}
```

---

## Configuration

### Generator Config (Go flags)

```go
type Config struct {
    SchemaPath string  // Required: path to .graphql SDL file
    OutputDir  string  // Optional: defaults to "./sdk"
    ConfigPath string  // Required: path to config.jsonc
}
```

`Validate()` enforces required fields and applies the `./sdk` default.

### Client Config (config.jsonc)

The config file uses JSONC format (JSON with comments), parsed via `tidwall/jsonc`.

```go
type ClientConfig struct {
    Bindings ConfigTSBindings `json:"bindings"`
}
```

### Scalar Bindings

Bindings map custom GraphQL scalars to TypeScript types. Each binding can be either a plain string (inline type alias) or an object with `type` and `import` fields (external package import).

```go
type TSBinding struct {
    Type   string `json:"type"`
    Import string `json:"import,omitempty"`
}

type ConfigTSBindings map[string]TSBinding
```

`TSBinding` implements a custom `UnmarshalJSON` that accepts both forms:

```jsonc
{
  "bindings": {
    // Plain string — backward compatible, produces: export type Cursor = string;
    "Cursor": "string",

    // Object with import — produces: export type { DateTime } from "luxon";
    "DateTime": { "type": "DateTime", "import": "luxon" },

    // Object with import + rename — produces: export type { JsonValue as JSON } from "type-fest";
    "JSON": { "type": "JsonValue", "import": "type-fest" }
  }
}
```

Scalars not listed in bindings fall back to `any`.

### External Package Imports

When a binding has an `import` field:

1. **`scalars/index.ts`** — emits a re-export from the npm package instead of a type alias
2. **All other files** — import the scalar by its GraphQL name from `../scalars` (transparent)

The `Merge` function handles this:

```go
func (m TSTypeMap) Merge(bindings ConfigTSBindings) TSTypeMap {
    for k, v := range bindings {
        if v.Import != "" {
            m[k] = k  // use scalar name — re-exported from scalars/
        } else {
            m[k] = v.Type
        }
    }
    return m
}
```

When `Import` is set, the type map stores the scalar name itself as the value. This ensures:
- `isSimpleTSType()` returns `false` — the type is not inlined
- `collectScalarRefs()` collects it — the scalar gets imported from `../scalars`
- `fieldTSType()` returns the scalar name — used in interface declarations

Generated output examples:

```typescript
// scalars/index.ts
export type Cursor = string;                           // plain binding
export type { DateTime } from "luxon";                 // same-name import
export type { JsonValue as JSON } from "type-fest";    // renamed import
export type Metadata = any;                            // unmapped fallback

// types/index.ts — imports transparently from ../scalars
import type { DateTime, Metadata } from "../scalars";

export interface Todo {
  createdAt: DateTime;   // uses the re-exported type
  metadata?: Metadata;
}

// queries/server-info.ts — references the scalar name
import type { JSON } from "../scalars";
async execute(): Promise<JSON> { ... }
```

---

## Type Mapping System

### TSTypeMap

```go
type TSTypeMap map[string]string
```

Maps GraphQL scalar names to their TypeScript type string. Built from `BuiltInTSTypes()` merged with user config bindings.

### Built-in Type Mappings

```go
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
```

Config bindings can override any of these.

### Type Resolution Functions

The generator uses several type resolution functions, each suited for a different context:

#### `graphQLToTSType(t *ast.Type) (string, bool)`

General-purpose converter. Returns the TS type string and whether the field is optional (nullable).

```
Int!     → ("number", false)
String   → ("string", true)
[Todo!]! → ("Todo[]", false)
```

#### `resolveTSType(t *ast.Type) string`

Core resolver called by `graphQLToTSType`. Handles list wrapping recursively:

```
t.Elem != nil → resolveTSType(t.Elem) + "[]"
t.Elem == nil → namedTypeToTS(t.NamedType)
```

#### `namedTypeToTS(name string) string`

Maps a named GraphQL type to TypeScript. Checks custom bindings first, then dispatches on schema definition kind:

| Kind | Result |
|------|--------|
| Scalar (in map) | The mapped type |
| Scalar (not in map) | The scalar name |
| Enum | The enum name |
| Object/Interface | The type name |
| InputObject | The input name |
| Unknown | `"any"` |

#### `graphQLToTSArgType(t *ast.Type) string`

Used for operation builder argument parameters. Similar to `graphQLToTSType` but for builder method signatures. For scalars, uses `tsTypeMap.Get(name)` directly.

#### `fieldTSType(t *ast.Type) string`

Used in interface field declarations (`types/index.ts`, `inputs/index.ts`). Key difference from other resolvers: built-in scalars with simple types (`string`, `number`, `boolean`) are inlined directly, while custom scalars reference the scalar type alias:

```go
case ast.Scalar:
    if tsType, ok := g.tsTypeMap[name]; ok && isSimpleTSType(tsType) {
        return tsType  // inline: "string", "number", etc.
    }
    return name  // reference: "DateTime", "Metadata"
```

#### `isSimpleTSType(t string) bool`

Returns `true` for `"string"`, `"number"`, `"boolean"`, `"any"`. Simple types are inlined and don't need imports from `../scalars`.

#### `toKebabCase(s string) string`

PascalCase to kebab-case for TypeScript file naming. Handles acronyms gracefully:

```
"ChatbotConnection" → "chatbot-connection"
"URLParser"         → "url-parser"
"TodoEdge"          → "todo-edge"
```

### Import Classification

#### `classifyTypeImport(t *ast.Type, types, scalars, enums, inputs)`

Recursively walks a GraphQL type tree and sorts each named reference into the correct import bucket:

| Schema Kind | Import Bucket | Import Path |
|-------------|---------------|-------------|
| Object/Interface | `types` | `../types` |
| Scalar (non-simple) | `scalars` | `../scalars` |
| Enum | `enums` | `../enums` |
| InputObject | `inputs` | `../inputs` |
| Scalar (simple) | — | (inlined, no import) |

#### `collectScalarRefs(t *ast.Type, seen map[string]bool)`

Recursively collects custom scalar references. Only collects scalars that are not simple TS types (i.e., need importing from `../scalars`). Handles the `String` → `GqlString` rename.

#### `collectEnumRefs(t *ast.Type, seen map[string]bool)`

Recursively collects enum type references.

#### `collectFieldSelectorTypeImports(def *ast.Definition) (enumImports, scalarImports []string)`

Collects enum and scalar imports needed by a specific field selector file.

#### `collectTypeEnumImports() / collectTypeScalarImports() []string`

Collects all enum/scalar imports needed by `types/index.ts`.

---

## Generation Modules

### 1. Builder Index

**Method:** `generateBuilderIndex()`
**Output:** `builder/index.ts`

Static file that re-exports runtime types from the `gqlkit-ts` npm package:

```typescript
export { FieldSelection, BaseBuilder } from "gqlkit-ts";
export { GraphQLClient } from "gqlkit-ts";
```

This indirection allows generated code to import from `../builder` uniformly.

### 2. Scalars

**Method:** `generateScalars()`
**Output:** `scalars/index.ts`
**Template:** `ts_scalar.tmpl`

#### Data Struct

```go
type TSScalar struct {
    Name       string  // GraphQL scalar name (or "GqlString" for "String")
    TSType     string  // TypeScript type (e.g., "string", "any")
    Import     string  // npm package path (empty = inline alias)
    ImportType string  // type name to import from package
}
```

#### Process

1. Iterate all schema scalars (skip GraphQL built-ins that `isGraphQLBuiltIn` returns true for)
2. Resolve TS type via `g.tsTypeMap.Get(def.Name)`
3. Special case: GraphQL `String` scalar → `GqlString` to avoid TS keyword conflict
4. If binding has external import, set `Import` and `ImportType` fields
5. Sort alphabetically by name
6. Execute template

#### Generated Output

```typescript
// Code generated by gqlkit. DO NOT EDIT.

export type Boolean = boolean;
export type Cursor = string;
export type { DateTime } from "luxon";
export type Float = number;
export type GqlString = string;
export type ID = string;
export type Int = number;
export type { JsonValue as JSON } from "type-fest";
export type Metadata = any;
```

### 3. Enums

**Method:** `generateEnums()`
**Output:** `enums/index.ts`
**Template:** `ts_enums.tmpl`

#### Data Struct

```go
type TSEnumDef struct {
    Name   string
    Values []string
}
```

#### Process

1. Collect all non-built-in enum types
2. Extract enum values
3. Sort alphabetically
4. Execute template

#### Generated Output

```typescript
// Code generated by gqlkit. DO NOT EDIT.

export enum Role {
  ADMIN = "ADMIN",
  USER = "USER",
  GUEST = "GUEST",
}
```

TypeScript string enums are used (value = name) to match GraphQL enum semantics.

### 4. Types (Interfaces)

**Method:** `generateTypes()`
**Output:** `types/index.ts`
**Template:** `ts_types.tmpl`

#### Data Structs

```go
type TSTypeDef struct {
    Name   string
    Fields []TSFieldDef
}

type TSFieldDef struct {
    Name     string  // camelCase field name
    Optional bool    // nullable = optional
    TSType   string  // resolved TypeScript type
}

type TSTypesData struct {
    EnumImports   []string   // enum names to import from ../enums
    ScalarImports []string   // scalar names to import from ../scalars
    Types         []TSTypeDef
}
```

#### Process

1. Iterate schema types, skip built-in, `__` prefixed, Query/Mutation/Subscription
2. Keep only Object and Interface kinds
3. For each field: resolve TS type via `fieldTSType()`, mark optional if nullable
4. Collect enum and scalar imports via `collectTypeEnumImports()` / `collectTypeScalarImports()`
5. Sort types alphabetically
6. Execute template

#### Generated Output

```typescript
// Code generated by gqlkit. DO NOT EDIT.

import type { Role } from "../enums";
import type { DateTime, Metadata } from "../scalars";

export interface Todo {
  id: string;
  text: string;
  done: boolean;
  priority?: number;
  tags: string[];
  user: User;
  createdAt: DateTime;
  metadata?: Metadata;
}

export interface User {
  id: string;
  name: string;
  email?: string;
  role: Role;
}
```

### 5. Input Types

**Method:** `generateInputTypes()`
**Output:** `inputs/index.ts`
**Template:** `ts_inputs.tmpl`

Same process and template as types, but filters for `ast.InputObject` kind instead of `ast.Object`/`ast.Interface`.

#### Data Struct

```go
type TSInputsData struct {
    EnumImports   []string
    ScalarImports []string
    Types         []TSTypeDef  // reuses TSTypeDef
}
```

#### Generated Output

```typescript
// Code generated by gqlkit. DO NOT EDIT.

export interface NewTodo {
  text: string;
  userId: string;
  priority?: number;
  tags?: string[];
}

export interface TodoFilter {
  textContains?: string;
  done?: boolean;
  minPriority?: number;
  hasTag?: string;
}
```

### 6. Field Selectors

**Method:** `generateFieldSelectionFiles()`
**Output:** `fields/{kebab-name}.ts` + `fields/index.ts`
**Templates:** `ts_field_selector.tmpl`, `ts_field_selector_index.tmpl`

Field selectors are the core type-safety mechanism. Each GraphQL object type gets a selector class that uses TypeScript generics with intersection types to build up a fully-typed result.

#### Data Structs

```go
type TSFieldSelectorData struct {
    TypeName      string              // GraphQL type name
    SelectorName  string              // e.g., "TodoFields"
    Fields        []TSFieldSelField
    Imports       []TSFieldImport     // other selector classes
    EnumImports   []string
    ScalarImports []string
}

type TSFieldSelField struct {
    FieldName      string  // GraphQL field name
    MethodName     string  // camelCase method name
    IsObject       bool    // nested object → callback pattern
    NestedSelector string  // e.g., "UserFields"
    TSType         string  // TypeScript type (for scalar/enum fields)
    IsNullable     bool    // nullable field → optional property
    IsList         bool    // list field → wraps U in U[]
}

type TSFieldImport struct {
    ClassName string  // e.g., "UserFields"
    FilePath  string  // e.g., "./user"
}

type TSFieldIndexEntry struct {
    SelectorName string
    FileName     string
}
```

#### Process

1. Get sorted object type names (excluding built-in, `__` prefixed, root types)
2. For each type, call `buildFieldSelectorData(def)`:
   - For each field (skip `__` fields):
     - If field type is an Object/Interface (and not self-referential): `IsObject=true`, set `NestedSelector`
     - Otherwise: set `TSType` via `fieldTSType()`
   - Collect imports for other selector classes
   - Collect enum and scalar imports
3. Execute `ts_field_selector.tmpl` for each → `fields/{kebab-name}.ts`
4. Collect entries
5. Execute `ts_field_selector_index.tmpl` → `fields/index.ts`

#### Generated Output — Scalar Fields

```typescript
// fields/user.ts
import { FieldSelection } from "../builder";
import type { Role } from "../enums";

export class UserFields<T extends object = {}> {
  private selection: FieldSelection;

  constructor(selection: FieldSelection) {
    this.selection = selection;
  }

  id(): UserFields<T & { id: string }> {
    this.selection.addField("id");
    return this as any;
  }

  name(): UserFields<T & { name: string }> {
    this.selection.addField("name");
    return this as any;
  }

  email(): UserFields<T & { email?: string }> {
    this.selection.addField("email");
    return this as any;
  }

  role(): UserFields<T & { role: Role }> {
    this.selection.addField("role");
    return this as any;
  }
}
```

#### Generated Output — Nested Object Fields

```typescript
// fields/todo.ts
import { FieldSelection } from "../builder";
import { UserFields } from "./user";
import type { DateTime, Metadata } from "../scalars";

export class TodoFields<T extends object = {}> {
  private selection: FieldSelection;

  constructor(selection: FieldSelection) {
    this.selection = selection;
  }

  id(): TodoFields<T & { id: string }> {
    this.selection.addField("id");
    return this as any;
  }

  // ... scalar fields ...

  user<U extends object>(
    selector: (f: UserFields) => UserFields<U>
  ): TodoFields<T & { user: U }> {
    const child = new FieldSelection();
    selector(new UserFields(child));
    this.selection.addChild("user", child);
    return this as any;
  }

  createdAt(): TodoFields<T & { createdAt: DateTime }> {
    this.selection.addField("createdAt");
    return this as any;
  }

  metadata(): TodoFields<T & { metadata?: Metadata }> {
    this.selection.addField("metadata");
    return this as any;
  }
}
```

#### Generated Output — Barrel File

```typescript
// fields/index.ts
export { NodeFields } from "./node";
export { PageInfoFields } from "./page-info";
export { TodoFields } from "./todo";
export { TodoConnectionFields } from "./todo-connection";
export { TodoEdgeFields } from "./todo-edge";
export { UserFields } from "./user";
```

### 7. Operation Builders

**Method:** `generateOperationFiles()`
**Output:** `{queries|mutations}/{kebab-name}.ts`, `{queries|mutations}/root.ts`, `{queries|mutations}/index.ts`
**Templates:** `ts_operation.tmpl`, `ts_operation_root.tmpl`, `ts_operation_index.tmpl`

#### Data Structs

```go
type TSArgumentData struct {
    ArgName     string  // GraphQL arg name
    MethodName  string  // camelCase setter method name
    TSType      string  // TypeScript parameter type
    GraphQLType string  // e.g., "ID!", "[TodoFilter]", "Int!"
}

type TSOperationBuilderData struct {
    BuilderName      string  // e.g., "TodoBuilder", "CreateTodoMutationBuilder"
    OpType           string  // "query" or "mutation"
    FieldName        string  // root field name (e.g., "todo", "createTodo")
    MethodName       string  // PascalCase (e.g., "Todo", "CreateTodo")
    Arguments        []TSArgumentData
    HasSelect        bool    // return type is object → select() method
    SelectorName     string  // e.g., "TodoFields"
    ReturnType       string  // e.g., "Todo", "Todo[]", "string"
    DefaultGeneric   string  // e.g., "Todo | null", "Todo[]"
    SelectWrapper    string  // how T wraps: "T", "T[]", "T | null", "T[] | null"
    IsListReturn     bool
    IsNullableReturn bool
}

type TSOperationImport struct {
    Names    []string
    Path     string
    TypeOnly bool  // use "import type" syntax
}

type TSOperationTemplateData struct {
    Data    TSOperationBuilderData
    Imports []TSOperationImport
}

type TSOperationRootBuilder struct {
    BuilderName string  // e.g., "TodoBuilder"
    FieldName   string  // original field name
    FileName    string  // kebab-case
}

type TSOperationRootData struct {
    ClassName string  // "QueryRoot" or "MutationRoot"
    OpType    string
    Builders  []TSOperationRootBuilder
}

type TSOperationIndexData struct {
    RootClassName string
    Builders      []TSOperationRootBuilder
}
```

#### Builder Name Convention

| Operation Type | Suffix | Example |
|----------------|--------|---------|
| Query | `Builder` | `TodoBuilder`, `UsersBuilder` |
| Mutation | `MutationBuilder` | `CreateTodoMutationBuilder` |

#### SelectWrapper Logic

The `SelectWrapper` determines how the generic type `T` is wrapped in the `select()` method return:

| IsListReturn | IsNullableReturn | SelectWrapper | Example |
|:---:|:---:|---|---|
| false | false | `T` | `T` |
| false | true | `T \| null` | `Todo \| null` |
| true | false | `T[]` | `Todo[]` |
| true | true | `T[] \| null` | `Todo[] \| null` |

#### Import Collection

`collectOperationImports()` gathers all imports for a single builder file:

1. Always: `BaseBuilder`, `GraphQLClient` from `../builder`
2. If `HasSelect`: `SelectorName` from `../fields`
3. Walk return type + all argument types through `classifyTypeImport()`:
   - Object/Interface (non-root) → `import type { ... } from "../types"`
   - Scalar (non-simple) → `import type { ... } from "../scalars"`
   - Enum → `import type { ... } from "../enums"`
   - InputObject → `import type { ... } from "../inputs"`

#### Generated Output — Query Builder (with select)

```typescript
// queries/todo.ts
import { BaseBuilder, GraphQLClient } from "../builder";
import { TodoFields } from "../fields";
import type { Todo } from "../types";

/** TodoBuilder builds a query for todo */
export class TodoBuilder<TResult = Todo | null> {
  private builder: BaseBuilder;

  constructor(client: GraphQLClient) {
    this.builder = new BaseBuilder(client, "query", "Todo", "todo");
  }

  /** Sets the id argument */
  id(v: string): this {
    this.builder.setArg("id", v, "ID!");
    return this;
  }

  /** Configures which fields to return */
  select<T extends object>(
    selector: (f: TodoFields) => TodoFields<T>
  ): TodoBuilder<T | null> {
    selector(new TodoFields(this.builder.getSelection()));
    return this as any;
  }

  /** Execute runs the query and returns the result */
  async execute(): Promise<TResult> {
    const query = this.builder.buildQuery();
    const variables = this.builder.getVariables();
    const response = await this.builder
      .getClient()
      .execute<{ todo: TResult }>(query, variables);
    return response.todo;
  }

  /** ExecuteRaw runs the query and returns the raw response */
  async executeRaw(): Promise<Record<string, unknown>> {
    return await this.builder.executeRaw();
  }
}
```

#### Generated Output — Scalar Query (no select)

```typescript
// queries/echo.ts
import { BaseBuilder, GraphQLClient } from "../builder";

/** EchoBuilder builds a query for echo */
export class EchoBuilder {
  private builder: BaseBuilder;

  constructor(client: GraphQLClient) {
    this.builder = new BaseBuilder(client, "query", "Echo", "echo");
  }

  /** Sets the message argument */
  message(v: string): this {
    this.builder.setArg("message", v, "String!");
    return this;
  }

  /** Execute runs the query and returns the result */
  async execute(): Promise<string> {
    const query = this.builder.buildQuery();
    const variables = this.builder.getVariables();
    const response = await this.builder
      .getClient()
      .execute<{ echo: string }>(query, variables);
    return response.echo;
  }

  /** ExecuteRaw runs the query and returns the raw response */
  async executeRaw(): Promise<Record<string, unknown>> {
    return await this.builder.executeRaw();
  }
}
```

#### Generated Output — Root File

```typescript
// queries/root.ts
import { GraphQLClient } from "../builder";
import { EchoBuilder } from "./echo";
import { PingBuilder } from "./ping";
import { TodoBuilder } from "./todo";
import { TodosBuilder } from "./todos";
import { UsersBuilder } from "./users";
// ...

/** QueryRoot is the entry point for querys */
export class QueryRoot {
  private client: GraphQLClient;

  constructor(client: GraphQLClient) {
    this.client = client;
  }

  ping(): PingBuilder {
    return new PingBuilder(this.client);
  }

  todo(): TodoBuilder {
    return new TodoBuilder(this.client);
  }

  todos(): TodosBuilder {
    return new TodosBuilder(this.client);
  }

  users(): UsersBuilder {
    return new UsersBuilder(this.client);
  }
  // ...
}
```

#### Generated Output — Barrel Index

```typescript
// queries/index.ts
export { QueryRoot } from "./root";
export { EchoBuilder } from "./echo";
export { PingBuilder } from "./ping";
export { TodoBuilder } from "./todo";
export { TodosBuilder } from "./todos";
export { UsersBuilder } from "./users";
// ...
```

---

## Templates

All templates are embedded in the Go binary via `//go:embed template/*` and compiled with these helper functions:

| Function | Behavior |
|----------|----------|
| `lower` | `strings.ToLower` |
| `upper` | `strings.ToUpper` |
| `joinComma` | `strings.Join(s, ", ")` |

### Template Reference

| Template File | Data Type | Output |
|---------------|-----------|--------|
| `ts_scalar.tmpl` | `[]TSScalar` | `scalars/index.ts` |
| `ts_enums.tmpl` | `[]TSEnumDef` | `enums/index.ts` |
| `ts_types.tmpl` | `TSTypesData` | `types/index.ts` |
| `ts_inputs.tmpl` | `TSInputsData` | `inputs/index.ts` |
| `ts_field_selector.tmpl` | `TSFieldSelectorData` | `fields/{name}.ts` |
| `ts_field_selector_index.tmpl` | `[]TSFieldIndexEntry` | `fields/index.ts` |
| `ts_operation.tmpl` | `TSOperationTemplateData` | `queries/{name}.ts` or `mutations/{name}.ts` |
| `ts_operation_root.tmpl` | `TSOperationRootData` | `queries/root.ts` or `mutations/root.ts` |
| `ts_operation_index.tmpl` | `TSOperationIndexData` | `queries/index.ts` or `mutations/index.ts` |

### Template: ts_scalar.tmpl

```
// Code generated by gqlkit. DO NOT EDIT.
{{ range . }}
{{- if .Import }}
{{- if eq .ImportType .Name }}
export type { {{ .Name }} } from "{{ .Import }}";
{{- else }}
export type { {{ .ImportType }} as {{ .Name }} } from "{{ .Import }}";
{{- end }}
{{- else }}
export type {{ .Name }} = {{ .TSType }};
{{- end }}
{{- end }}
```

### Template: ts_enums.tmpl

```
// Code generated by gqlkit. DO NOT EDIT.
{{ range $i, $enum := . }}
{{ if $i }}
{{ end -}}
export enum {{ $enum.Name }} {
{{- range $enum.Values }}
  {{ . }} = "{{ . }}",
{{- end }}
}
{{- end }}
```

### Template: ts_types.tmpl / ts_inputs.tmpl

Both use the same template structure:

```
// Code generated by gqlkit. DO NOT EDIT.
{{ if .EnumImports }}
import type { {{ joinComma .EnumImports }} } from "../enums";
{{- end }}
{{ if .ScalarImports }}
import type { {{ joinComma .ScalarImports }} } from "../scalars";
{{- end }}
{{ if or .EnumImports .ScalarImports }}
{{ end }}
{{- range $i, $td := .Types }}
{{ if $i }}
{{ end -}}
export interface {{ $td.Name }} {
{{- range $td.Fields }}
  {{ .Name }}{{ if .Optional }}?{{ end }}: {{ .TSType }};
{{- end }}
}
{{- end }}
```

### Template: ts_field_selector.tmpl

```
// Code generated by gqlkit. DO NOT EDIT.

import { FieldSelection } from "../builder";
{{- range .Imports }}
import { {{ .ClassName }} } from "{{ .FilePath }}";
{{- end }}
{{- if .EnumImports }}
import type { {{ joinComma .EnumImports }} } from "../enums";
{{- end }}
{{- if .ScalarImports }}
import type { {{ joinComma .ScalarImports }} } from "../scalars";
{{- end }}

/** {{ .SelectorName }} provides type-safe field selection for {{ .TypeName }} */
export class {{ .SelectorName }}<T extends object = {}> {
  private selection: FieldSelection;

  constructor(selection: FieldSelection) {
    this.selection = selection;
  }
{{ range .Fields }}
{{ if .IsObject }}  {{ .MethodName }}<U extends object>(
    selector: (f: {{ .NestedSelector }}) => {{ .NestedSelector }}<U>
  ): {{ $.SelectorName }}<T & { {{ .FieldName }}{{ if .IsNullable }}?{{ end }}: {{ if .IsList }}U[]{{ else }}U{{ end }} }> {
    const child = new FieldSelection();
    selector(new {{ .NestedSelector }}(child));
    this.selection.addChild("{{ .FieldName }}", child);
    return this as any;
  }
{{ else }}  {{ .MethodName }}(): {{ $.SelectorName }}<T & { {{ .FieldName }}{{ if .IsNullable }}?{{ end }}: {{ .TSType }} }> {
    this.selection.addField("{{ .FieldName }}");
    return this as any;
  }
{{ end }}
{{- end -}}
}
```

### Template: ts_operation.tmpl

```
// Code generated by gqlkit. DO NOT EDIT.
{{ range .Imports }}
{{- if .TypeOnly }}
import type { {{ joinComma .Names }} } from "{{ .Path }}";
{{- else }}
import { {{ joinComma .Names }} } from "{{ .Path }}";
{{- end }}
{{- end }}

/** {{ .Data.BuilderName }} builds a {{ .Data.OpType }} for {{ .Data.FieldName }} */
{{- if .Data.HasSelect }}
export class {{ .Data.BuilderName }}<TResult = {{ .Data.DefaultGeneric }}> {
{{- else }}
export class {{ .Data.BuilderName }} {
{{- end }}
  private builder: BaseBuilder;

  constructor(client: GraphQLClient) {
    this.builder = new BaseBuilder(client, "{{ .Data.OpType }}", "{{ .Data.MethodName }}", "{{ .Data.FieldName }}");
  }
{{ range .Data.Arguments }}
  /** Sets the {{ .ArgName }} argument */
  {{ .MethodName }}(v: {{ .TSType }}): this {
    this.builder.setArg("{{ .ArgName }}", v, "{{ .GraphQLType }}");
    return this;
  }
{{ end }}
{{- if .Data.HasSelect }}
  /** Configures which fields to return */
  select<T extends object>(
    selector: (f: {{ .Data.SelectorName }}) => {{ .Data.SelectorName }}<T>
  ): {{ .Data.BuilderName }}<{{ .Data.SelectWrapper }}> {
    selector(new {{ .Data.SelectorName }}(this.builder.getSelection()));
    return this as any;
  }
{{ end }}
  /** Execute runs the {{ .Data.OpType }} and returns the result */
{{- if .Data.HasSelect }}
  async execute(): Promise<TResult> { ... }
{{- else }}
  async execute(): Promise<{{ .Data.ReturnType }}> { ... }
{{- end }}

  /** ExecuteRaw runs the {{ .Data.OpType }} and returns the raw response */
  async executeRaw(): Promise<Record<string, unknown>> {
    return await this.builder.executeRaw();
  }
}
```

### Template: ts_operation_root.tmpl

```
// Code generated by gqlkit. DO NOT EDIT.

import { GraphQLClient } from "../builder";
{{- range .Builders }}
import { {{ .BuilderName }} } from "./{{ .FileName }}";
{{- end }}

/** {{ .ClassName }} is the entry point for {{ .OpType }}s */
export class {{ .ClassName }} {
  private client: GraphQLClient;

  constructor(client: GraphQLClient) {
    this.client = client;
  }
{{ range .Builders }}
  {{ .FieldName }}(): {{ .BuilderName }} {
    return new {{ .BuilderName }}(this.client);
  }
{{ end -}}
}
```

### Template: ts_operation_index.tmpl

```
// Code generated by gqlkit. DO NOT EDIT.

export { {{ .RootClassName }} } from "./root";
{{- range .Builders }}
export { {{ .BuilderName }} } from "./{{ .FileName }}";
{{- end }}
```

---

## Runtime Library (gqlkit-ts)

The `gqlkit-ts` npm package provides three classes that the generated SDK depends on. It has zero runtime dependencies.

### Public Exports

```typescript
export { FieldSelection } from "./builder";
export { BaseBuilder } from "./builder";
export { GraphQLClient } from "./graphqlclient";
export { GraphQLErrors } from "./graphqlclient";
export type { ClientOptions } from "./graphqlclient";
```

### FieldSelection

Tracks which fields are selected in a GraphQL query tree. Maintains two data structures:

```typescript
class FieldSelection {
  private fields: string[] = [];                         // scalar field names
  private children: Map<string, FieldSelection> = new Map();  // nested objects
}
```

| Method | Description |
|--------|-------------|
| `addField(name)` | Add a scalar field |
| `addChild(name, child)` | Add a nested object with its own sub-selection |
| `build(indent = 2)` | Serialize to GraphQL selection set string |
| `isEmpty()` | Check if any fields are selected |

`build()` output example:

```
  id
  name
  address {
    city
    zip
  }
```

### BaseBuilder

Core builder for query/mutation operations. Generated classes delegate to this.

```typescript
class BaseBuilder {
  private client: GraphQLClient;
  private opType: string;       // "query" or "mutation"
  private opName: string;       // operation name (e.g., "GetUser")
  private fieldName: string;    // root field name (e.g., "user")
  private args: Map<string, { value: unknown; graphqlType: string }>;
  private selection: FieldSelection;
}
```

| Method | Description |
|--------|-------------|
| `setArg(name, value, graphqlType)` | Register an argument with its GraphQL type |
| `getSelection()` | Access the field selection tree |
| `getClient()` | Access the GraphQL client |
| `getVariables()` | Extract runtime variable values as `Record<string, unknown>` |
| `buildQuery()` | Serialize to complete GraphQL operation string |
| `executeRaw()` | Build query and execute, returning raw data object |

`buildQuery()` output example:

```graphql
query GetUser($id: ID!) {
  user(id: $id) {
    id
    name
    email
  }
}
```

### GraphQLClient

HTTP transport layer. Sends GraphQL operations as JSON POST requests.

```typescript
class GraphQLClient {
  constructor(endpoint: string, options?: ClientOptions)
  async execute<T>(query: string, variables?: Record<string, unknown>): Promise<T>
  async rawQuery(query: string, variables?: Record<string, unknown>): Promise<unknown>
}

interface ClientOptions {
  headers?: Record<string, string>;   // extra HTTP headers
  authToken?: string;                 // Bearer token
  fetch?: typeof fetch;               // custom fetch (SSR/testing)
}
```

`execute()` flow:

1. Resolve fetch function (custom or `globalThis.fetch`)
2. Build headers: `Content-Type: application/json` + user headers + optional Bearer auth
3. POST `{ query, variables }` to endpoint
4. Parse JSON response
5. If `errors` array present → throw `GraphQLErrors`
6. If no `data` → throw generic Error
7. Return typed `response.data`

### GraphQLErrors

```typescript
class GraphQLErrors extends Error {
  public errors: GraphQLError[];
  // message = all error messages joined by "; "
}

interface GraphQLError {
  message: string;
  locations?: { line: number; column: number }[];
  path?: (string | number)[];
  extensions?: Record<string, unknown>;
}
```

---

## Generated Output Structure

```
sdk/
├── builder/
│   └── index.ts              ← re-exports from gqlkit-ts
├── scalars/
│   └── index.ts              ← type aliases + external re-exports
├── enums/
│   └── index.ts              ← string enums
├── types/
│   └── index.ts              ← interfaces for Object/Interface types
├── inputs/
│   └── index.ts              ← interfaces for InputObject types
├── fields/
│   ├── {type-name}.ts        ← field selector class per type
│   └── index.ts              ← barrel re-exports
├── queries/
│   ├── {field-name}.ts       ← query builder class per field
│   ├── root.ts               ← QueryRoot factory class
│   └── index.ts              ← barrel re-exports
└── mutations/
    ├── {field-name}.ts       ← mutation builder class per field
    ├── root.ts               ← MutationRoot factory class
    └── index.ts              ← barrel re-exports
```

All generated files begin with `// Code generated by gqlkit. DO NOT EDIT.`

File naming uses kebab-case: `TodoConnection` → `todo-connection.ts`.

---

## Data Flow Walkthrough

### Example: `todos(filter: TodoFilter, pagination: PaginationInput): [Todo!]!`

**Step 1 — Schema Parsing**

gqlparser produces an AST `FieldDefinition`:

```
Name: "todos"
Arguments: [
  { Name: "filter",     Type: TodoFilter (nullable) },
  { Name: "pagination", Type: PaginationInput (nullable) }
]
Type: [Todo!]!  // non-null list of non-null Todo
```

**Step 2 — buildOperationBuilderData("query", field)**

```
builderName      = "TodosBuilder"       (PascalCase + "Builder")
opType           = "query"
fieldName        = "todos"
methodName       = "Todos"
returnType       = "Todo[]"             (graphQLToTSArgType)
isListReturn     = true                 (field.Type.Elem != nil)
isNullableReturn = false                (field.Type.NonNull)
defaultGeneric   = "Todo[]"
selectWrapper    = "T[]"
hasSelect        = true                 (Todo is Object type)
selectorName     = "TodoFields"
arguments        = [
  { ArgName: "filter",     TSType: "TodoFilter",     GraphQLType: "TodoFilter" },
  { ArgName: "pagination", TSType: "PaginationInput", GraphQLType: "PaginationInput" }
]
```

**Step 3 — collectOperationImports()**

```
BaseBuilder, GraphQLClient → ../builder
TodoFields                 → ../fields
TodoFilter, PaginationInput → import type from ../inputs
Todo                       → import type from ../types
```

**Step 4 — Template Renders TodosBuilder**

The `ts_operation.tmpl` template produces the complete builder class.

**Step 5 — Runtime Execution**

```typescript
const result = await qr
  .todos()
  .filter({ done: false })
  .pagination({ limit: 10, offset: 0 })
  .select((t) => t.id().text().done())
  .execute();
// TypeScript type: { id: string; text: string; done: boolean }[]
```

`buildQuery()` produces:

```graphql
query Todos($filter: TodoFilter, $pagination: PaginationInput) {
  todos(filter: $filter, pagination: $pagination) {
    id
    text
    done
  }
}
```

`getVariables()` returns:

```json
{
  "filter": { "done": false },
  "pagination": { "limit": 10, "offset": 0 }
}
```

---

## Design Patterns

### Type-Safe Field Selection via Generic Intersection Types

Each method on a field selector returns a new generic instantiation that intersects the previous type with the new field:

```typescript
class TodoFields<T extends object = {}> {
  id():   TodoFields<T & { id: string }>   { ... }
  text(): TodoFields<T & { text: string }> { ... }
  done(): TodoFields<T & { done: boolean }> { ... }
}

// Chaining: TodoFields<{} & {id: string} & {text: string}> = TodoFields<{id: string; text: string}>
```

TypeScript resolves the intersection type at compile time, giving full autocomplete and type checking on the result.

### Nested Selection via Callbacks

Object fields use a callback that receives a fresh selector for the nested type:

```typescript
user<U extends object>(
  selector: (f: UserFields) => UserFields<U>
): TodoFields<T & { user: U }> {
  const child = new FieldSelection();
  selector(new UserFields(child));
  this.selection.addChild("user", child);
  return this as any;
}
```

The callback's return type `UserFields<U>` captures what fields were selected, and `U` flows into the parent's generic.

### Builder Pattern with Method Chaining

Argument setters return `this` for fluent chaining:

```typescript
qr.todo()
  .id("1")
  .select((t) => t.id().text())
  .execute()
```

### Transparent Scalar Re-exports

External package types are re-exported from `scalars/index.ts` under the GraphQL scalar name. All other generated files import from `../scalars` without knowing the original source:

```
luxon → DateTime → scalars/index.ts → types/index.ts, queries/*.ts, fields/*.ts
```

### Barrel File Pattern

Each module exports everything from a single `index.ts`:

```typescript
// fields/index.ts
export { TodoFields } from "./todo";
export { UserFields } from "./user";
```

Enables clean imports: `import { TodoFields, UserFields } from "./fields"`.

### File Naming Convention

- Go structs/types: PascalCase
- Generated TS files: kebab-case (e.g., `TodoConnection` → `todo-connection.ts`)
- Query builders: `{PascalField}Builder` (e.g., `TodoBuilder`)
- Mutation builders: `{PascalField}MutationBuilder` (e.g., `CreateTodoMutationBuilder`)
- Field selectors: `{TypeName}Fields` (e.g., `TodoFields`)
- Root classes: `QueryRoot`, `MutationRoot`

### File Writer

`TSWriter` writes files to disk with no formatting applied. Generated TypeScript is expected to be formatted by external tools (Prettier, ESLint, etc.) as part of the project's build pipeline.

```go
type TSWriter struct {
    outputDir string
}

func (w *TSWriter) WriteFile(filename, content string) error
func (w *TSWriter) EnsureDir() error
```

---

## Error Handling

### Generation-Time Errors

```go
// Sentinel errors
ErrSchemaPathRequired = errors.New("schema path is required")
ErrConfigPathRequired = errors.New("config path is required")

// Contextual errors (wrapped)
fmt.Errorf("failed to load client config: %w", err)
fmt.Errorf("failed to parse schema: %w", err)
fmt.Errorf("failed to execute scalar template: %w", err)
// etc.
```

### Runtime Errors

```typescript
// Server returned GraphQL errors
throw new GraphQLErrors(errors)  // errors: GraphQLError[]

// Response has no data
throw new Error("No data returned from GraphQL query")
```
