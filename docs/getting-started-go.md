# GQLKit — Go SDK Getting Started Guide

GQLKit generates fully typed GraphQL client SDKs from a GraphQL schema. The generated SDK uses a **builder pattern** with type-safe field selection — only selected fields appear in the return type.

---

## Prerequisites

- Go 1.21+
- A GraphQL API endpoint or `.graphql` schema file

---

## Table of Contents

- [Step 1: Install the CLI tools](#step-1-install-the-cli-tools)
- [Step 2: Get your GraphQL schema](#step-2-get-your-graphql-schema)
- [Step 3: Configure scalar type mappings](#step-3-configure-scalar-type-mappings)
- [Step 4: Generate the SDK](#step-4-generate-the-sdk)
- [Step 5: Use the generated SDK](#step-5-use-the-generated-sdk)
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

## Step 3: Configure scalar type mappings

Create a `config.jsonc` file to map custom GraphQL scalars to Go types:

```jsonc
{
  "bindings": {
    "Time": {
      "model": "time.Time"
    },
    "JSON": {
      "model": "encoding/json.RawMessage"
    },
    "UUID": {
      "model": "github.com/google/uuid.UUID"
    }
  }
}
```

Built-in scalars (`String`, `Int`, `Float`, `Boolean`, `ID`) are mapped automatically. You only need to configure custom scalars. Unmapped custom scalars default to `any`.

---

## Step 4: Generate the SDK

There are two ways to generate the SDK:

### Option A — CLI (recommended)

Use the `gqlkit` CLI for a quick, no-code setup:

```bash
gqlkit generate \
  --schema schema.graphql \
  --output ./sdk \
  --package sdk \
  --module github.com/yourorg/yourproject/sdk
```

**CLI flags:**

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--schema` | `-s` | *(required)* | Path to `.graphql` schema file |
| `--output` | `-o` | `./sdk` | Output directory for generated SDK |
| `--package` | `-p` | `sdk` | Go package name |
| `--module` | `-m` | | Go module path (e.g., `github.com/user/myapi`) |

### Option B — Go script (for more control)

Create a Go file (e.g., `cmd/generate/main.go`) to invoke the generator programmatically:

```go
package main

import (
    "fmt"

    "gqlkit/pkg/clientgen"
)

func main() {
    config := &clientgen.Config{
        SchemaPath:  "cmd/generate/schema.graphql",
        OutputDir:   "./sdk",
        PackageName: "sdk",
        ModulePath:  "github.com/yourorg/yourproject/sdk",
        ConfigPath:  "cmd/generate/config.jsonc",
        Package:     "yourmodule/sdk",
    }

    gen, err := clientgen.New(config)
    if err != nil {
        fmt.Printf("failed to create generator: %v\n", err)
        return
    }

    if err := gen.Generate(); err != nil {
        fmt.Printf("failed to generate SDK: %v\n", err)
        return
    }

    fmt.Println("SDK generation completed.")
}
```

Then run it:

```bash
go run cmd/generate/main.go
```

---

Either option creates the `sdk/` directory with the following packages:

```
sdk/
├── builder/     # Runtime builder types (included in SDK, no extra dependency needed)
├── enums/       # GraphQL enums as Go types
├── fields/      # Field selector types (one per GraphQL type)
├── inputs/      # Input type structs
├── mutations/   # Mutation builders + MutationRoot
├── queries/     # Query builders + QueryRoot
├── scalars/     # Custom scalar type aliases
└── types/       # Go struct definitions for GraphQL types
```

---

## Step 5: Use the generated SDK

Add the `gqlkit` runtime dependency:

```bash
go get gqlkit/pkg/graphqlclient
```

> **Note:** The `builder` package is already copied into your generated `sdk/builder/` directory — no need to install it separately.

Then use the SDK in your code:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "gqlkit/pkg/graphqlclient"

    "yourmodule/sdk/fields"
    "yourmodule/sdk/inputs"
    "yourmodule/sdk/mutations"
    "yourmodule/sdk/queries"
)

func main() {
    // 1. Create the client
    client := graphqlclient.NewClient(
        "https://your-api.example.com/graphql",
        graphqlclient.WithAuthToken("YOUR_TOKEN"),
    )

    ctx := context.Background()

    // 2. Create query/mutation root
    qr := queries.NewQueryRoot(client)
    mr := mutations.NewMutationRoot(client)

    // 3. Execute queries
    todos, err := qr.Todos().
        Filter(&inputs.TodoFilter{Done: boolPtr(false)}).
        Pagination(&inputs.PaginationInput{Limit: 10, Offset: 0}).
        Select(func(f *fields.TodoFields) {
            f.ID().Text().Done().
                User(func(u *fields.UserFields) {
                    u.ID().Name()
                })
        }).
        Execute(ctx)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Got %d todos\n", len(todos))
}

func boolPtr(b bool) *bool { return &b }
```

---

## Examples

### Simple scalar queries

```go
// No field selection needed for scalar return types
pingResult, err := qr.Ping().Execute(ctx)

echoResult, err := qr.Echo().Message("Hello from SDK").Execute(ctx)

sumResult, err := qr.Sum().A(40).B(2).Execute(ctx)
```

### Query with nested field selection

```go
users, err := qr.Users().
    Select(func(u *fields.UserFields) {
        u.ID().Name().Email().Role()
    }).
    Execute(ctx)
```

### Pagination with cursor-based connections

```go
conn, err := qr.TodosConnection().
    Filter(&inputs.TodoFilter{Done: boolPtr(false)}).
    Pagination(&inputs.PaginationInput{Limit: 5, Offset: 0}).
    Select(func(c *fields.TodoConnectionFields) {
        c.TotalCount().
            PageInfo(func(p *fields.PageInfoFields) {
                p.HasNextPage().HasPreviousPage().StartCursor().EndCursor()
            }).
            Edges(func(e *fields.TodoEdgeFields) {
                e.Cursor().Node(func(t *fields.TodoFields) {
                    t.ID().Text().Done()
                })
            })
    }).
    Execute(ctx)
```

### Mutations (create, update, delete)

```go
// Create
created, err := mr.CreateTodo().
    Input(inputs.NewTodo{
        Text:   "SDK-created todo",
        UserID: "1",
        Tags:   []string{"sdk", "created"},
    }).
    Select(func(f *fields.TodoFields) {
        f.ID().Text().Done().Tags()
    }).
    Execute(ctx)

// Update
updated, err := mr.UpdateTodo().
    ID(created.ID).
    Input(inputs.UpdateTodoInput{
        Text: stringPtr("Updated via SDK"),
        Done: boolPtr(true),
    }).
    Select(func(f *fields.TodoFields) {
        f.ID().Text().Done().Tags()
    }).
    Execute(ctx)

// Delete
deleted, err := mr.DeleteTodo().ID(created.ID).Execute(ctx)

// Scalar mutation (returns int)
completedCount, err := mr.CompleteAllTodos().Execute(ctx)
```

### Raw JSON access

```go
rawData, err := qr.Ping().ExecuteRaw(ctx)
// rawData is map[string]interface{}
```

### Union types

```go
builder := qr.Search().Term("a")
selection := builder.GetSelection()

todoFrag := sdkbuilder.NewFieldSelection()
todoFrag.AddField("id")
todoFrag.AddField("text")
selection.AddChild("... on Todo", todoFrag)

userFrag := sdkbuilder.NewFieldSelection()
userFrag.AddField("id")
userFrag.AddField("name")
selection.AddChild("... on User", userFrag)

results, err := builder.Execute(ctx)
```

---

## Error Handling

```go
import (
    "errors"
    "gqlkit/pkg/graphqlclient"
)

result, err := qr.Todos().Select(/* ... */).Execute(ctx)
if err != nil {
    var gqlErrs graphqlclient.GraphQLErrors
    if errors.As(err, &gqlErrs) {
        // Structured GraphQL errors
        for _, e := range gqlErrs {
            fmt.Printf("GraphQL error: %s\n", e.Message)
            fmt.Printf("  Path: %v\n", e.Path)
            fmt.Printf("  Extensions: %v\n", e.Extensions)
        }
    } else {
        // Network or other error
        fmt.Printf("Error: %v\n", err)
    }
}
```

---

## Client Options

```go
client := graphqlclient.NewClient(
    "https://your-api.example.com/graphql",

    // Bearer token authentication
    graphqlclient.WithAuthToken("YOUR_TOKEN"),

    // Custom headers
    graphqlclient.WithHeader("X-Request-ID", "abc123"),
    graphqlclient.WithHeaders(map[string]string{
        "X-Custom": "value",
    }),

    // Custom HTTP client (for timeouts, proxies, TLS, etc.)
    graphqlclient.WithHTTPClient(&http.Client{
        Timeout: 30 * time.Second,
    }),
)
```
