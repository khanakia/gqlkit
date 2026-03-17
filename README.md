# GQLKit — GraphQL SDK Generator

A Go workspace that generates fully typed GraphQL client SDKs (Go and TypeScript) from a GraphQL schema. The generated SDKs use a **builder pattern** with type-safe field selection — only selected fields appear in the return type.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        gqlkit (core)                         │
│                                                              │
│  GraphQL SDL ──→ schemagql ──→ typegql ──→ clientgen (Go)    │
│                                       └──→ clientgents (TS)  │
│                                                              │
│  Runtime: builder + graphqlclient (Go), gqlkit-ts (TS)       │
└─────────────────────────────────────────────────────────────┘

┌──────────────┐   introspection   ┌──────────────┐
│  gqlkit-sdl  │ ────────────────→ │ .graphql SDL  │
│  (CLI tool)  │   fetch + convert │   (schema)    │
└──────────────┘                   └──────┬───────┘
                                          │
                              ┌───────────┴───────────┐
                              ▼                       ▼
                    Go SDK (example-go-*)    TS SDK (example-ts)
```

## Modules

| Module | Description |
|--------|-------------|
| [gqlkit](./gqlkit) | Core SDK generator — parses schema, generates Go and TypeScript client code |
| [gqlkit-ts](./gqlkit-ts) | TypeScript runtime library (npm) — `GraphQLClient`, `BaseBuilder`, `FieldSelection` |
| [gqlkit-sdl](./gqlkit-sdl) | CLI tool — fetches GraphQL schema via introspection, outputs SDL |
| [mockapi](./mockapi) | Test GraphQL API (gqlgen) — todo/user CRUD with filtering and pagination |
| [example-go-chat](./example-go-chat) | Example: Go SDK from a production chatbot schema (~50 queries, ~57 mutations) |
| [example-go-mockapi](./example-go-mockapi) | Example: Go SDK from the test API |
| [example-ts](./example-ts) | Example: TypeScript SDK from the test API |

## Requirements

* Go 1.21+
* Node.js 18+ (for TypeScript SDK)
* [Task](https://taskfile.dev) (optional, for task runner)

***

## TypeScript SDK Generator

Generates a fully typed TypeScript SDK from a GraphQL schema. Returns **only the selected fields** — unselected fields are compile-time errors.

Full technical docs: [example-ts/DOCS.md](./example-ts/DOCS.md)

### 1. Start the GraphQL server

```bash
cd mockapi
go run server.go
# → running on http://localhost:8081/query
```

### 2. Build the TypeScript runtime library

```bash
cd gqlkit-ts
npm install
npm run build
```

### 3. Install TypeScript dependencies

```bash
cd example-ts
npm install
```

### 4. Build the Go generator

```bash
cd example-ts
go build ./cmd/generate/
```

### 5. Generate the TypeScript SDK

```bash
cd example-ts
go run cmd/generate/main.go
```

Output goes to `example-ts/sdk/` (~30 `.ts` files).

### 6. Type-check the generated SDK

```bash
cd example-ts
npx tsc --noEmit
# no output = no errors
```

### 7. Run sample queries against the live server

```bash
cd example-ts
npm run samples
```

### Quick re-test (after changing generator code)

```bash
cd example-ts
task example-ts:test
# or manually:
go build ./cmd/generate/ && rm -rf sdk && go run cmd/generate/main.go && npx tsc --noEmit
```

### Tasks

```bash
task example-ts:setup           # Steps 2-6 in one command (first-time)
task example-ts:test            # Build + vet + clean generate + typecheck
task example-ts:generate        # Generate SDK
task example-ts:generate:clean  # rm -rf sdk + generate
task example-ts:typecheck       # tsc --noEmit
task example-ts:run             # Run samples
task gqlkit-ts:setup            # Install + build runtime library
```

***

## Go SDK Generator

### Generate the Go SDK

```bash
cd example-go-chat
go run ./cmd/generate
```

### End-to-end with test API

```bash
# 1) Start API (in one terminal)
task mockapi:run

# 2) Fetch schema from test API
task example-go-mockapi:fetch-schema

# 3) Generate SDK
task example-go-mockapi:generate

# 4) Run sample queries
task example-go-mockapi:run
```

***
