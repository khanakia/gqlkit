# TypeScript SDK Generator (`clientgents`) - Technical Documentation

## Overview

`clientgents` is a Go-based code generator that reads a GraphQL SDL schema and produces a fully typed TypeScript SDK. It is the TypeScript counterpart of the existing Go generator (`clientgen`). The generated SDK uses the `gqlkit-ts` runtime library for query building and execution.

**Key feature:** The SDK returns **only the selected fields** in the TypeScript return type. When you call `.select(t => t.id().name())`, the `execute()` return type is `{ id: string; name: string }` — not the full interface. Unselected fields are compile-time errors.

---

## Quick Start

### Prerequisites

- Go 1.21+
- Node.js 18+ with npm
- The `gqlkit-ts` runtime library built (`cd ../gqlkit-ts && npm install && npm run build`)

### Generate the SDK

```bash
cd example-ts
go run cmd/generate/main.go
```

### Verify the output

```bash
# Check TypeScript compiles cleanly
npx tsc --noEmit
```

---

## Testing Commands

```bash
# --- 1. Build the Go generator ---
cd example-ts
go build ./cmd/generate/
go vet ./cmd/generate/

# --- 2. Build the runtime library (required for TS type checking) ---
cd ../gqlkit-ts
npm install
npm run build

# --- 3. Install dependencies in example-ts ---
cd ../example-ts
npm install

# --- 4. Generate the TypeScript SDK ---
go run cmd/generate/main.go

# --- 5. Verify generated SDK compiles cleanly (zero errors in sdk/) ---
npx tsc --noEmit 2>&1 | grep "^sdk/"
# Expected: no output (no errors)

# --- 6. Clean regenerate (from scratch) ---
rm -rf sdk
go run cmd/generate/main.go
npx tsc --noEmit 2>&1 | grep "^sdk/"

# --- 7. Count generated files ---
find sdk -name "*.ts" | wc -l

# --- 8. List generated directory structure ---
find sdk -type d | sort

# --- 9. Run sample queries against a live server (manual test) ---
# First start the test API: cd ../mockapi && go run server.go
npm run samples
```

---

## Architecture

### Generator Package: `gqlkit/pkg/clientgents/`

```
clientgents/
├── clientgents.go      # Generator struct, Generate() pipeline, shared helpers
├── config.go           # Config struct, ClientConfig, JSONC loading
├── errors.go           # Error constants
├── ts_type_map.go      # GraphQL → TypeScript type mapping, toKebabCase()
├── scalar_gen.go       # Generates scalars/index.ts + import collectors
├── enum_gen.go         # Generates enums/index.ts
├── types_gen.go        # Generates types/index.ts (interfaces)
├── input_types_gen.go  # Generates inputs/index.ts (input interfaces)
├── field_sel_gen.go    # Generates fields/*.ts + fields/index.ts
├── op_gen.go           # Generates queries/, mutations/ (builders, roots, indexes)
├── writer.go           # TSWriter - writes raw content (no go/format)
└── template/
    ├── ts_scalar.tmpl  # Scalar type alias template
    └── ts_enums.tmpl   # Enum template
```

### Entry Point: `example-ts/cmd/generate/`

```
cmd/generate/
├── main.go             # Calls clientgents.New(config).Generate()
├── config.jsonc        # Custom scalar → TS type overrides
└── schema.graphql      # GraphQL SDL input
```

---

## Generated SDK Structure

```
sdk/
├── builder/
│   └── index.ts            # Re-exports from gqlkit-ts runtime
├── scalars/
│   └── index.ts            # export type ID = string; export type Cursor = any; ...
├── enums/
│   └── index.ts            # export enum Role { ADMIN = "ADMIN", ... }
├── types/
│   └── index.ts            # export interface User { id: string; name: string; ... }
├── inputs/
│   └── index.ts            # export interface NewTodo { text: string; userId: string; ... }
├── fields/
│   ├── index.ts            # Barrel: export { TodoFields } from "./todo"; ...
│   ├── todo.ts             # TodoFields class with id(), text(), user(selector) methods
│   ├── todo-edge.ts        # TodoEdgeFields with cursor(), node(selector)
│   ├── todo-connection.ts  # TodoConnectionFields with edges(selector), pageInfo(selector)
│   ├── user.ts             # UserFields with id(), name(), email(), role()
│   └── page-info.ts        # PageInfoFields with hasNextPage(), endCursor()
├── queries/
│   ├── index.ts            # Barrel: export { QueryRoot } from "./root"; ...
│   ├── root.ts             # QueryRoot class with ping(), todos(), users() factory methods
│   ├── ping.ts             # PingBuilder with execute(): Promise<string>
│   ├── todos.ts            # TodosBuilder with filter(v), select(fn), execute()
│   └── ...                 # One file per query field
└── mutations/
    ├── index.ts            # Barrel: export { MutationRoot } from "./root"; ...
    ├── root.ts             # MutationRoot class with createTodo(), deleteTodo() factory methods
    ├── create-todo.ts      # CreateTodoMutationBuilder with input(v), select(fn), execute()
    └── ...                 # One file per mutation field
```

---

## Generation Pipeline

```
1. Validate config (SchemaPath, OutputDir, ConfigPath)
2. Load config.jsonc (custom scalar bindings)
3. Parse GraphQL schema (schemagql.GetSchema)
4. Build TypeScript type map (built-in + config overrides)
5. Parse embedded Go templates
6. Generate files in order:
   a. builder/index.ts       - Re-export from gqlkit-ts
   b. scalars/index.ts       - Scalar type aliases
   c. enums/index.ts         - TypeScript enums
   d. types/index.ts         - Object type interfaces
   e. inputs/index.ts        - Input type interfaces
   f. fields/*.ts + index.ts - Field selector classes (one per object type)
   g. queries/*.ts           - Query builders + root + index
   h. mutations/*.ts         - Mutation builders + root + index
```

---

## Type Mapping (GraphQL → TypeScript)

| GraphQL Type | TypeScript Type | Notes |
|---|---|---|
| `String` | `string` | |
| `Int`, `Float` | `number` | |
| `Boolean` | `boolean` | |
| `ID` | `string` | |
| `Uint`, `Uint64`, `Int64`, `Float64` | `number` | |
| Custom scalars (unknown) | `any` | Configurable via config.jsonc |
| `Time` | `string` | Default; configurable |
| Nullable field | `field?: Type` | Optional property |
| Non-null field | `field: Type` | Required property |
| List `[Type!]!` | `Type[]` | |
| Nullable list `[Type]` | `Type[]` | |
| Object type | interface name | |
| Enum | enum name | |
| Input type | interface name | |

### Config Override Example (config.jsonc)

```jsonc
{
  // Custom scalar → TypeScript type overrides
  "bindings": {
    "Time": "string",       // Time scalar becomes string
    "JSON": "Record<string, unknown>",  // Custom mapping
    "Cursor": "string"      // Override default 'any'
  }
}
```

---

## File Naming Conventions

| Context | Convention | Example |
|---|---|---|
| Field selector file | kebab-case | `TodoConnection` → `todo-connection.ts` |
| Query builder file | kebab-case of field name | `todosConnection` → `todos-connection.ts` |
| Mutation builder file | kebab-case of field name | `createTodo` → `create-todo.ts` |
| Field selector class | PascalCase + "Fields" | `Todo` → `TodoFields` |
| Query builder class | PascalCase + "Builder" | `todos` → `TodosBuilder` |
| Mutation builder class | PascalCase + "MutationBuilder" | `createTodo` → `CreateTodoMutationBuilder` |
| Interface field names | Original GraphQL name (camelCase) | `createdAt` → `createdAt` |
| Method names | Original GraphQL name (camelCase) | `totalCount` → `totalCount()` |

---

## Import Strategy

| Import Source | Style | Example |
|---|---|---|
| Runtime library | Value import | `import { FieldSelection } from "../builder"` |
| Field selectors | Value import | `import { UserFields } from "./user"` |
| Type interfaces | Type-only import | `import type { User } from "../types"` |
| Scalar types | Type-only import | `import type { Cursor } from "../scalars"` |
| Enum types | Type-only import | `import type { Role } from "../enums"` |
| Input types | Type-only import | `import type { NewTodo } from "../inputs"` |

---

## Differences from Go Generator (`clientgen`)

| Aspect | Go (`clientgen`) | TypeScript (`clientgents`) |
|---|---|---|
| Output language | Go structs/methods | TypeScript classes/interfaces |
| Nullable handling | Pointer types (`*string`) | Optional properties (`field?: string`) |
| Type formatting | `go/format` | Raw output (no formatter) |
| Builder runtime | Embedded via template (`builder.tmpl`) | Re-exported from `gqlkit-ts` npm package |
| File naming | `snake_case` with prefix (`field_todo.go`) | `kebab-case` (`todo.ts`) |
| Package system | Go packages (one per directory) | ES modules with barrel `index.ts` files |
| Field/method names | PascalCase (`CreatedAt()`) | camelCase (`createdAt()`) — uses original GraphQL names |
| Type bindings | Go type strings (`"time.Time"`) | TS type strings (`"string"`) |
| Acronym handling | ID→ID, URL→URL via ent conventions | Not applied (uses GraphQL names directly) |

---

## Reused Code from Go Generator

| Package | What's Reused |
|---|---|
| `gqlkit/pkg/schemagql` | Schema parsing (`GetSchema`) — unchanged |
| `gqlkit/pkg/util` | `ToPascalCase()`, `ToCamelCase()` for operation/builder names |
| `github.com/tidwall/jsonc` | Config loading |
| `github.com/vektah/gqlparser/v2` | GraphQL AST types |

---

## Runtime Library: `gqlkit-ts`

The generated SDK depends on the `gqlkit-ts` npm package which provides:

- **`FieldSelection`** — Tracks selected fields, builds GraphQL field selection strings
  - `addField(name)` — Add scalar field
  - `addChild(name, child)` — Add nested field with sub-selection
  - `build(indent)` — Render to GraphQL string

- **`BaseBuilder`** — Common query/mutation builder functionality
  - `setArg(name, value, graphqlType)` — Set operation argument
  - `getSelection()` — Get field selection instance
  - `getClient()` — Get GraphQL client
  - `buildQuery()` — Build complete GraphQL query string
  - `getVariables()` — Get variables map
  - `executeRaw()` — Execute and return raw response

- **`GraphQLClient`** — HTTP client for GraphQL
  - `execute<T>(query, variables)` — Execute operation, return typed data
  - `rawQuery(query, variables)` — Execute and return raw response

---

## Type-Safe Field Selection

The SDK returns **only the selected fields** in the TypeScript return type. This is achieved through four mechanisms:

### 1. Generic field selectors

Each field selector class carries a generic `T extends object = {}` that accumulates selected fields via TypeScript intersection types. Every method call adds its field to `T`:

```typescript
// Generated: fields/todo.ts
export class TodoFields<T extends object = {}> {
  id():       TodoFields<T & { id: string }>       { ... }
  text():     TodoFields<T & { text: string }>      { ... }
  done():     TodoFields<T & { done: boolean }>     { ... }
  priority(): TodoFields<T & { priority?: number }> { ... }  // nullable → optional
  tags():     TodoFields<T & { tags: string[] }>    { ... }  // list type

  // Object fields capture the inner selector's accumulated type as U
  user<U extends object>(
    selector: (f: UserFields) => UserFields<U>
  ): TodoFields<T & { user: U }> { ... }
}
```

Chaining `t.id().text().done()` produces `TodoFields<{ id: string } & { text: string } & { done: boolean }>` which TypeScript simplifies to `TodoFields<{ id: string; text: string; done: boolean }>`.

### 2. Generic operation builders

Operation builders carry `<TResult = FullType>`. The `select()` method captures the accumulated type `T` from the field selector callback and narrows `TResult`:

```typescript
// Generated: queries/todos-connection.ts
export class TodosConnectionBuilder<TResult = TodoConnection> {
  filter(v: TodoFilter): this { ... }     // preserves TResult via `this`
  pagination(v: PaginationInput): this { ... }

  select<T extends object>(
    selector: (f: TodoConnectionFields) => TodoConnectionFields<T>
  ): TodosConnectionBuilder<T> { ... }    // narrows TResult to T

  async execute(): Promise<TResult> { ... }
}
```

### 3. Three return type patterns

The `select()` wrapper varies based on the GraphQL return type:

| GraphQL return | Builder default | `select()` returns | `execute()` returns |
|---|---|---|---|
| `TodoConnection!` (non-null) | `<TResult = TodoConnection>` | `Builder<T>` | `Promise<T>` |
| `Todo` (nullable) | `<TResult = Todo \| null>` | `Builder<T \| null>` | `Promise<T \| null>` |
| `[Todo!]!` (list) | `<TResult = Todo[]>` | `Builder<T[]>` | `Promise<T[]>` |

Without `.select()`, `execute()` returns the full interface type (backwards compatible).
With `.select()`, `execute()` returns only the selected fields — unselected fields are compile-time errors.

### 4. Enum/scalar imports in field selectors

Since scalar field methods now reference actual TypeScript types in their generic return type (e.g., `{ role: Role }`), the generator automatically collects and emits enum and custom scalar imports for each field selector file.

### Behavior in practice

```typescript
// Without select → returns full TodoConnection type
const full = await qr.todosConnection().execute();
full.totalCount; // ✓ number
full.edges;      // ✓ TodoEdge[]

// With select → returns ONLY the selected fields
const narrow = await qr
  .todosConnection()
  .select((conn) => conn.totalCount())
  .execute();

narrow.totalCount; // ✓ number
narrow.edges;      // ✗ COMPILE ERROR — not selected

// Nested selection → deeply narrowed types
const nested = await qr
  .todosConnection()
  .select((conn) =>
    conn.edges((e) =>
      e.node((t) => t.id().text().user((u) => u.id().name()))
    )
  )
  .execute();

nested.edges[0].node.id;          // ✓ string
nested.edges[0].node.user.name;   // ✓ string
nested.edges[0].node.done;        // ✗ COMPILE ERROR — not selected
nested.edges[0].node.user.email;  // ✗ COMPILE ERROR — not selected

// Arguments can go before or after select — both work
const r1 = await qr.todosConnection().filter({done: true}).select(c => c.totalCount()).execute();
const r2 = await qr.todosConnection().select(c => c.totalCount()).filter({done: true}).execute();
// Both return { totalCount: number }
```

---

## Testing Commands

### Full test flow (from scratch)

```bash
# --- 1. Build the runtime library ---
cd gqlkit-ts
npm install
npm run build

# --- 2. Build the Go generator ---
cd ../example-ts
go build ./cmd/generate/
go vet ./cmd/generate/

# --- 3. Install TS dependencies ---
npm install

# --- 4. Generate the TypeScript SDK ---
go run cmd/generate/main.go

# --- 5. Verify generated SDK compiles cleanly (zero errors in sdk/) ---
npx tsc --noEmit 2>&1 | grep "^sdk/"
# Expected: no output (no errors)

# --- 6. Verify ALL files compile (sdk + sample code) ---
npx tsc --noEmit
# Expected: no output (no errors)

# --- 7. Clean regenerate (from scratch) ---
rm -rf sdk
go run cmd/generate/main.go
npx tsc --noEmit
# Expected: no output (no errors)

# --- 8. Inspect generated output ---
find sdk -name "*.ts" | wc -l        # Count generated files
find sdk -type d | sort               # List directory structure
cat sdk/fields/todo.ts                # Inspect a field selector (should have generics)
cat sdk/queries/todos-connection.ts   # Inspect an operation builder (should have TResult generic)

# --- 9. Verify type-safe narrowing works ---
# Add a negative test to confirm unselected fields are errors:
cat > cmd/samples/negative-test.ts << 'EOF'
import { GraphQLClient } from "gqlkit-ts";
import { QueryRoot } from "../../sdk/queries";
const qr = new QueryRoot(new GraphQLClient("http://localhost/graphql"));
async function test() {
  const result = await qr.todosConnection().select(c => c.totalCount()).execute();
  // @ts-expect-error — edges was NOT selected, this must be a type error
  console.log(result.edges);
}
test();
EOF
npx tsc --noEmit
# Expected: no output (the @ts-expect-error consumed the error, proving narrowing works)
rm cmd/samples/negative-test.ts

# --- 10. Run sample queries against a live server (manual/integration test) ---
# First start the test API in another terminal:
#   cd ../mockapi && go run server.go
npm run samples
```

### Quick re-test after generator changes

```bash
cd example-ts
go build ./cmd/generate/ && rm -rf sdk && go run cmd/generate/main.go && npx tsc --noEmit
```

---

## Usage Example (TypeScript)

```typescript
import { GraphQLClient } from "gqlkit-ts";
import { QueryRoot } from "./sdk/queries";
import { MutationRoot } from "./sdk/mutations";

async function main() {
  const client = new GraphQLClient("http://localhost:8081/query");
  const qr = new QueryRoot(client);
  const mr = new MutationRoot(client);

  // Simple scalar query
  const pong = await qr.ping().execute();
  console.log(pong); // "pong"

  // Query with field selection — return type is narrowed!
  const todos = await qr
    .todosConnection()
    .filter({ done: false })
    .pagination({ limit: 10, offset: 0 })
    .select((conn) =>
      conn
        .totalCount()
        .edges((e) =>
          e.cursor().node((t) =>
            t.id().text().done().user((u) => u.id().name())
          )
        )
        .pageInfo((p) => p.hasNextPage().endCursor())
    )
    .execute();

  // TypeScript knows exactly what's available:
  todos.totalCount;                    // ✓ number
  todos.edges[0].node.id;             // ✓ string
  todos.edges[0].node.user.name;      // ✓ string
  // todos.edges[0].node.priority;    // ✗ compile error — not selected

  // Mutation with narrowed return
  const newTodo = await mr
    .createTodo()
    .input({ text: "Buy milk", userId: "user-1" })
    .select((t) => t.id().text().done())
    .execute();

  newTodo.id;   // ✓ string
  newTodo.done; // ✓ boolean

  // Nullable return — result is { id: string; text: string } | null
  const maybeTodo = await qr
    .todo()
    .id("todo-1")
    .select((t) => t.id().text())
    .execute();

  if (maybeTodo) {
    maybeTodo.id;   // ✓ string
    maybeTodo.text; // ✓ string
  }

  // List return — result is { id: string; done: boolean }[]
  const todoList = await qr
    .todos()
    .select((t) => t.id().done())
    .execute();

  todoList[0].id;   // ✓ string
  todoList[0].done; // ✓ boolean

  // Without select — returns full type (backwards compatible)
  const fullResult = await qr.todosConnection().execute();
  fullResult.totalCount; // ✓ number (full TodoConnection type)
  fullResult.edges;      // ✓ TodoEdge[]
}
```
