# GQLKit — TypeScript SDK Getting Started Guide

GQLKit generates fully typed TypeScript GraphQL client SDKs from a GraphQL schema. The generated SDK uses a **builder pattern** with type-safe field selection — only selected fields appear in the return type. Accessing unselected fields is a compile-time error.

---

## Prerequisites

- Go 1.21+ (required for code generation)
- Node.js 18+
- A GraphQL API endpoint or `.graphql` schema file

---

## Table of Contents

- [Step 1: Install the CLI tools](#step-1-install-the-cli-tools)
- [Step 2: Get your GraphQL schema](#step-2-get-your-graphql-schema)
- [Step 3: Install the runtime library](#step-3-install-the-runtime-library)
- [Step 4: Configure scalar type mappings](#step-4-configure-scalar-type-mappings)
- [Step 5: Generate the SDK](#step-5-generate-the-sdk)
- [Step 6: Use the generated SDK](#step-6-use-the-generated-sdk)
- [Examples](#examples)
- [Error Handling](#error-handling)
- [Client Options](#client-options)

---

## Step 1: Install the CLI tools

**Option A — Quick install (macOS / Linux):**

```bash
# Install gqlkit (SDK generator)
curl -sL https://raw.githubusercontent.com/khanakia/gqlkit/main/gqlkit/install.sh | sh

# Install gqlkit-sdl (schema fetcher)
curl -sL https://raw.githubusercontent.com/khanakia/gqlkit/main/gqlkit-sdl/install.sh | sh
```

**Option B — Download from GitHub Releases:**

Go to [Releases](https://github.com/khanakia/gqlkit/releases) and download the binary for your platform.

---

## Step 2: Get your GraphQL schema

If you already have a `.graphql` schema file, skip this step.

To fetch a schema from a running GraphQL endpoint via introspection:

```bash
gqlkit-sdl fetch \
  --url https://your-api.example.com/graphql \
  --output schema.graphql
```

With authentication:

```bash
gqlkit-sdl fetch \
  --url https://your-api.example.com/graphql \
  --output schema.graphql \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

## Step 3: Install the runtime library

```bash
npm install gqlkit-ts
```

This provides the runtime classes: `GraphQLClient`, `BaseBuilder`, `FieldSelection`, and `GraphQLErrors`.

---

## Step 4: Configure scalar type mappings

Create a `config.jsonc` file to map custom GraphQL scalars to TypeScript types:

```jsonc
{
  "bindings": {
    "Time": "string",
    "JSON": "Record<string, unknown>",
    "Cursor": "string"
  }
}
```

Built-in scalars (`String`, `Int`, `Float`, `Boolean`, `ID`) are mapped automatically. Unmapped custom scalars default to `any`.

---

## Step 5: Generate the SDK

> **Note:** The generator itself is written in Go even for the TypeScript SDK. You need Go installed to run the code generation step.

There are two ways to generate the SDK:

### Option A — CLI (recommended)

```bash
gqlkit generate-ts \
  --schema schema.graphql \
  --output ./sdk \
  --config config.jsonc
```

**CLI flags:**

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--schema` | `-s` | *(required)* | Path to `.graphql` schema file |
| `--output` | `-o` | `./sdk` | Output directory for generated SDK |
| `--config` | `-c` | *(required)* | Path to `config.jsonc` file |

### Option B — Go script (for more control)

Create a Go file (e.g., `cmd/generate/main.go`):

```go
package main

import (
    "fmt"

    "gqlkit/pkg/clientgents"
)

func main() {
    config := &clientgents.Config{
        SchemaPath: "cmd/generate/schema.graphql",
        OutputDir:  "./sdk",
        ConfigPath: "cmd/generate/config.jsonc",
    }

    gen, err := clientgents.New(config)
    if err != nil {
        fmt.Printf("failed to create generator: %v\n", err)
        return
    }

    if err := gen.Generate(); err != nil {
        fmt.Printf("failed to generate SDK: %v\n", err)
        return
    }

    fmt.Println("TypeScript SDK generation completed.")
}
```

Then run it:

```bash
go run cmd/generate/main.go
```

---

Either option creates the `sdk/` directory:

```
sdk/
├── builder/index.ts       # Re-exports from gqlkit-ts
├── enums/index.ts         # TypeScript enums
├── fields/                # Field selector classes (one per type)
│   ├── index.ts
│   ├── todo.ts
│   ├── user.ts
│   └── ...
├── inputs/index.ts        # Input type interfaces
├── mutations/             # Mutation builders + MutationRoot
│   ├── index.ts
│   ├── root.ts
│   └── ...
├── queries/               # Query builders + QueryRoot
│   ├── index.ts
│   ├── root.ts
│   └── ...
├── scalars/index.ts       # Scalar type aliases
└── types/index.ts         # TypeScript interfaces
```

---

## Step 6: Use the generated SDK

```typescript
import { GraphQLClient } from "gqlkit-ts";
import { QueryRoot } from "./sdk/queries";
import { MutationRoot } from "./sdk/mutations";

// 1. Create the client
const client = new GraphQLClient("https://your-api.example.com/graphql", {
  authToken: "YOUR_TOKEN",
  headers: { "X-Custom-Header": "value" },
});

// 2. Create query/mutation root
const qr = new QueryRoot(client);
const mr = new MutationRoot(client);

// 3. Execute queries
const todos = await qr
  .todos()
  .filter({ done: false })
  .pagination({ limit: 10, offset: 0 })
  .select((t) =>
    t.id().text().done().user((u) => u.id().name())
  )
  .execute();

console.log(todos);
// Only selected fields are available — accessing unselected fields is a compile-time error
```

---

## Examples

### Simple scalar queries

```typescript
const pingResult = await qr.ping().execute();

const echoResult = await qr.echo().message("Hello from SDK!").execute();

const sumResult = await qr.sum().a(10).b(20).execute();
```

### Query with nested field selection

```typescript
const users = await qr
  .users()
  .select((u) => u.id().name().email().role())
  .execute();
```

### Single item query

```typescript
const todo = await qr
  .todo()
  .id("todo-1")
  .select((t) => t.id().text().done().user((u) => u.id().name()))
  .execute();
```

### Pagination with cursor-based connections

```typescript
const result = await qr
  .todosConnection()
  .filter({ done: false })
  .pagination({ limit: 10, offset: 0 })
  .select((conn) =>
    conn
      .totalCount()
      .edges((e) =>
        e.cursor().node((t) =>
          t.id().text().done().priority().user((u) => u.id().name().email())
        )
      )
      .pageInfo((p) =>
        p.hasNextPage().hasPreviousPage().startCursor().endCursor()
      )
  )
  .execute();
```

### Mutations

```typescript
// Create
const created = await mr
  .createTodo()
  .input({ text: "Buy milk", userId: "user-1" })
  .select((t) => t.id().text().done())
  .execute();

// Delete
const deleted = await mr.deleteTodo().id("todo-1").execute();

// Scalar mutation
const completedCount = await mr.completeAllTodos().execute();
```

### Raw JSON access

```typescript
const rawData = await qr.ping().executeRaw();
// rawData is Record<string, unknown>
```

---

## Error Handling

```typescript
import { GraphQLErrors } from "gqlkit-ts";

try {
  const result = await qr.todos().select(/* ... */).execute();
} catch (err) {
  if (err instanceof GraphQLErrors) {
    // Structured GraphQL errors
    for (const e of err.errors) {
      console.error("GraphQL error:", e.message);
      console.error("  Path:", e.path);
      console.error("  Extensions:", e.extensions);
    }
  } else {
    // Network or other error
    console.error("Error:", err);
  }
}
```

---

## Client Options

```typescript
const client = new GraphQLClient("https://your-api.example.com/graphql", {
  // Bearer token authentication
  authToken: "YOUR_TOKEN",

  // Custom headers
  headers: {
    "X-Request-ID": "abc123",
    "X-Custom": "value",
  },

  // Custom fetch implementation (e.g., for Node.js < 18 or custom middleware)
  fetch: customFetchFunction,
});
```
