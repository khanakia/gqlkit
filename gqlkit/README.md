# gqlkit — GraphQL SDK Generator (Core)

The core library that generates fully type-safe GraphQL client SDKs in **Go** and **TypeScript** from a GraphQL SDL schema file.

## Architecture

```
cmd/cli/             CLI entry point (cobra-based)
pkg/
  schemagql/         Parse .graphql SDL files into an AST
  typegql/           Map GraphQL types → Go types (with go/types resolution)
  clientgen/         Go SDK code generator
  clientgents/       TypeScript SDK code generator
  builder/           Runtime builder used by generated Go SDKs
  graphqlclient/     Runtime HTTP client used by generated Go SDKs
  writer/            Go file writer (writes + gofmt)
  templater/         Template engine with embedded templates + helper funcs
  util/              String utilities (PascalCase, camelCase, snake_case)
```

## Generation Pipeline

```
SDL schema file (.graphql)
        │
        ▼
   schemagql.ParseSchemaFile()    ← parse into *ast.Schema
        │
        ▼
   typegql.Build() / TSTypeMap    ← map GraphQL scalars to Go/TS types
        │
        ▼
   clientgen.Generator.Generate() ← Go SDK
   clientgents.Generator.Generate() ← TypeScript SDK
        │
        ▼
   Generated SDK directory:
     scalars/   enums/   types/   inputs/
     fields/    queries/   mutations/   builder/
```

## Go SDK Generator (`pkg/clientgen`)

Reads a GraphQL schema and produces a complete Go module with:

| Output directory | Contents |
|-----------------|----------|
| `scalars/` | Custom scalar type aliases (e.g., `type Time = time.Time`) |
| `enums/` | String-typed enums with constants |
| `types/` | Go structs for object/interface types |
| `inputs/` | Go structs for input types |
| `fields/` | Field selector types (one per object type) |
| `queries/` | Query builder per query field + `QueryRoot` |
| `mutations/` | Mutation builder per mutation field + `MutationRoot` |
| `builder/` | Copies of `pkg/builder` runtime files |

### Generated SDK usage pattern

```go
client := graphqlclient.NewClient("http://localhost:8081/query")
qr := queries.NewQueryRoot(client)

todos, err := qr.Todos().
    Filter(&inputs.TodoFilter{Done: boolPtr(false)}).
    Select(func(f *fields.TodoFields) {
        f.ID().Text().Done().User(func(u *fields.UserFields) {
            u.ID().Name()
        })
    }).
    Execute(ctx)
```

## TypeScript SDK Generator (`pkg/clientgents`)

Reads the same GraphQL schema and produces TypeScript files:

| Output directory | Contents |
|-----------------|----------|
| `scalars/` | Scalar type aliases |
| `enums/` | TypeScript enums |
| `types/` | TypeScript interfaces for object types |
| `inputs/` | TypeScript interfaces for input types |
| `fields/` | Field selector classes (one per object type) |
| `queries/` | Query builder classes + `QueryRoot` |
| `mutations/` | Mutation builder classes + `MutationRoot` |
| `builder/` | Re-exports from `gqlkit-ts` runtime |

### Generated SDK usage pattern

```typescript
const client = new GraphQLClient("http://localhost:8081/query");
const qr = new QueryRoot(client);

const todos = await qr.todosConnection()
    .filter({ done: false })
    .select((conn) =>
        conn.totalCount().edges((e) =>
            e.node((t) => t.id().text().done())
        )
    )
    .execute();
```

## Runtime Libraries

### `pkg/builder` (Go runtime)

Provides `FieldSelection` and `BaseBuilder` — the foundation that every generated query/mutation builder extends. Handles:
- Tracking selected fields and nested selections
- Building GraphQL query strings with variables
- Executing queries via the `GraphQLClient` interface

### `pkg/graphqlclient` (Go runtime)

Lightweight HTTP client for GraphQL endpoints. Supports:
- Bearer token auth, custom headers
- JSON request/response marshaling
- Structured `GraphQLErrors` with message, location, path, and extensions

## Configuration

Each SDK project has a `config.jsonc` file for custom scalar bindings:

```jsonc
{
  "bindings": {
    "Time": { "model": "time.Time" },
    "JSON": { "model": "encoding/json.RawMessage" }
  }
}
```

## CLI Usage

```bash
go run ./cmd/cli generate \
  --schema path/to/schema.graphql \
  --output ./sdk \
  --package sdk \
  --module github.com/myorg/myproject/sdk
```

## Dependencies

- `github.com/99designs/gqlgen` — GraphQL AST parser (`gqlparser/v2`)
- Go 1.21+
