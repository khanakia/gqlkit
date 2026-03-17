# gqlkit-sdl — GraphQL Schema Introspection Fetcher

A CLI tool that fetches a GraphQL schema from a live endpoint via the standard introspection query and converts it to SDL (Schema Definition Language) format.

## Pipeline

```
GraphQL Endpoint
      │
      ▼  HTTP POST (introspection query)
schema.FetchSchema()
      │
      ▼  JSON → Go structs
schema.ConvertToSDL()
      │
      ▼  Go structs → SDL text
schema.SaveToFile()
      │
      ▼
schema.graphql
```

## Usage

```bash
go run . -url <graphql-endpoint> [options]
```

### Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `-url` | Yes | — | GraphQL endpoint URL |
| `-output` | No | `schema.graphql` | Output file path |
| `-auth` | No | — | `Authorization` header value |
| `-referer` | No | — | `Referer` header value |
| `-origin` | No | — | `Origin` header value |

### Examples

```bash
# Basic usage
go run . -url "https://api.example.com/graphql"

# With custom output file
go run . -url "https://api.example.com/graphql" -output my-schema.graphql

# With authentication
go run . -url "https://api.example.com/graphql" -auth "Bearer your-token"

# With referer and origin headers
go run . -url "http://localhost:2310/sa/query_playground" \
  -referer "http://localhost:2310/sa/gql?pkey=1234" \
  -origin "http://localhost:2310"
```

## Package Structure

```
main.go              CLI entry point — parses flags, orchestrates the pipeline
schema/
  types.go           Type definitions mirroring the GraphQL introspection schema
  fetcher.go         HTTP client that sends the introspection query
  converter.go       Converts introspection JSON structs into SDL text
```

### `schema/types.go`

Defines Go structs that mirror the standard GraphQL introspection response:
- `IntrospectionResponse` → `IntrospectionData` → `IntrospectionSchema`
- `FullType` — represents SCALAR, OBJECT, INTERFACE, UNION, ENUM, INPUT_OBJECT
- `TypeInfo` — recursive wrapper for type modifiers (`NON_NULL`, `LIST`, nested up to 7 levels)
- `Field`, `InputValue`, `EnumValue`, `Directive`

### `schema/fetcher.go`

Sends the standard introspection query via HTTP POST. The `TypeRef` fragment recurses 7 levels deep to handle arbitrarily nested type modifiers like `[String!]!`.

### `schema/converter.go`

Walks the introspection schema and emits SDL text:
- Filters out built-in types (`__Type`, `__Field`, etc.) and built-in scalars (`Int`, `Float`, `String`, `Boolean`, `ID`)
- Filters out built-in directives (`@skip`, `@include`, `@deprecated`, `@specifiedBy`)
- Sorts types alphabetically for deterministic output
- Handles descriptions (single-line and block quotes)
- Smart argument formatting (single-line for simple, multi-line for complex)

## Package API

The `schema` package can also be used programmatically:

```go
import "gqlkit-sdl/schema"

// Fetch schema from endpoint
opts := &schema.FetchOptions{
    Headers: map[string]string{
        "Authorization": "Bearer token",
        "Referer":       "http://example.com",
    },
}

introspectionSchema, err := schema.FetchSchema("https://api.example.com/graphql", opts)
if err != nil {
    log.Fatal(err)
}

// Convert to SDL
sdl := schema.ConvertToSDL(introspectionSchema)

// Save to file
err = schema.SaveToFile(sdl, "schema.graphql")
```

## Dependencies

None (standard library only).
