# Contributing to gqlkit

## Architecture

The tool reads a GraphQL SDL schema file, parses it into an AST, maps types to Go/TypeScript equivalents, and generates a complete type-safe client SDK.

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

## Project Structure

```
cmd/cli/
  main.go              CLI entry point
  root.go              Cobra root command + version
pkg/
  schemagql/           Parse .graphql SDL files into an AST
  typegql/             Map GraphQL types → Go types (with go/types resolution)
  clientgen/           Go SDK code generator
  clientgents/         TypeScript SDK code generator
  builder/             Runtime builder used by generated Go SDKs
  graphqlclient/       Runtime HTTP client used by generated Go SDKs
  writer/              Go file writer (writes + gofmt)
  templater/           Template engine with embedded templates + helper funcs
  util/                String utilities (PascalCase, camelCase, snake_case)
.goreleaser.yml        Cross-platform build and release config
install.sh             Auto-detect install script
```

## Key Packages

### `pkg/schemagql`

Parses `.graphql` SDL files into a `*ast.Schema` using `gqlparser/v2`.

### `pkg/typegql`

Maps GraphQL scalar types to Go types with `go/types` resolution. Handles custom scalar bindings from `config.jsonc`.

### `pkg/clientgen` (Go SDK Generator)

Generates a complete Go module with:

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

### `pkg/clientgents` (TypeScript SDK Generator)

Generates TypeScript files with the same structure — interfaces, enums, field selectors, and query/mutation builders.

### `pkg/builder` (Go runtime)

Provides `FieldSelection` and `BaseBuilder` — the foundation that every generated query/mutation builder extends. Handles tracking selected fields, building GraphQL query strings with variables, and executing queries.

### `pkg/graphqlclient` (Go runtime)

Lightweight HTTP client for GraphQL endpoints. Supports bearer token auth, custom headers, and structured `GraphQLErrors`.

## Development

### Prerequisites

- Go 1.25.5+

### Run locally

```bash
cd gqlkit
go run ./cmd/cli generate --schema path/to/schema.graphql
```

### Build locally

```bash
cd gqlkit
go build -o gqlkit ./cmd/cli
```

### Run tests

```bash
cd gqlkit
go test ./...
```

### Test GoReleaser locally

```bash
cd gqlkit
goreleaser release --snapshot --clean
# Built binaries will be in dist/
```

## Releasing

Releases are automated via GitHub Actions. To publish a new version:

```bash
git tag v0.2.0
git push origin v0.2.0
```

This triggers `.github/workflows/release.yml` which runs GoReleaser to build binaries for all platforms (linux/darwin, amd64/arm64, windows) and creates a GitHub Release.

## Dependencies

- [cobra](https://github.com/spf13/cobra) — CLI framework
- [gqlparser/v2](https://github.com/vektah/gqlparser) — GraphQL SDL parser
- [doublestar](https://github.com/bmatcuk/doublestar) — Glob pattern matching
