# Contributing to gqlkit-sdl

## Architecture

The tool follows a simple pipeline: fetch introspection JSON from a GraphQL endpoint, convert it to SDL text, and write it to a file.

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

## Project Structure

```
main.go              CLI entry point — cobra commands, orchestrates the pipeline
schema/
  types.go           Type definitions mirroring the GraphQL introspection schema
  fetcher.go         HTTP client that sends the introspection query
  converter.go       Converts introspection JSON structs into SDL text
.goreleaser.yml      Cross-platform build and release config
```

## Key Packages

### `schema/types.go`

Go structs that mirror the standard GraphQL introspection response:
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

## Development

### Prerequisites

- Go 1.25.5+

### Run locally

```bash
cd gqlkit-sdl
go run . fetch --url "http://localhost:8080/graphql"
```

### Build locally

```bash
cd gqlkit-sdl
go build -o gqlkit-sdl .
```

### Test GoReleaser locally

```bash
cd gqlkit-sdl
goreleaser release --snapshot --clean
# Built binaries will be in dist/
```

## Releasing

Releases are automated via GitHub Actions. To publish a new version:

```bash
git tag v0.1.0
git push origin v0.1.0
```

This triggers `.github/workflows/release.yml` which runs GoReleaser to build binaries for all platforms (linux/darwin, amd64/arm64, windows) and creates a GitHub Release.

## Dependencies

- [cobra](https://github.com/spf13/cobra) — CLI framework
