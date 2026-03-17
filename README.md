# GQLKit — GraphQL SDK Generator

Generate fully typed GraphQL client SDKs for **Go** and **TypeScript** from any GraphQL schema. Built on a **builder pattern** with type-safe field selection — only the fields you select appear in the return type.

## Install

```bash
# macOS / Linux
curl -sL https://raw.githubusercontent.com/khanakia/gqlkit/main/gqlkit/install.sh | sh
curl -sL https://raw.githubusercontent.com/khanakia/gqlkit/main/gqlkit-sdl/install.sh | sh
```

Or download binaries from [Releases](https://github.com/khanakia/gqlkit/releases).

## Quick Start — Go

**1. Fetch your schema** (skip if you already have a `.graphql` file):

```bash
gqlkit-sdl fetch --url https://your-api.example.com/graphql -o schema.graphql
```

**2. Generate the SDK:**

```bash
gqlkit generate --schema schema.graphql --output ./sdk --package sdk
```

**3. Use it:**

```go
import (
    "gqlkit/pkg/graphqlclient"
    "yourmodule/sdk/queries"
    "yourmodule/sdk/fields"
)

client := graphqlclient.NewClient("https://your-api.example.com/graphql",
    graphqlclient.WithAuthToken("YOUR_TOKEN"),
)

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

Full guide: [docs/getting-started-go.md](./docs/getting-started-go.md)

## Quick Start — TypeScript

**1. Fetch your schema** (skip if you already have a `.graphql` file):

```bash
gqlkit-sdl fetch --url https://your-api.example.com/graphql -o schema.graphql
```

**2. Install the runtime:**

```bash
npm install gqlkit-ts
```

**3. Generate the SDK:**

```bash
gqlkit generate-ts --schema schema.graphql --output ./sdk --config config.jsonc
```

**4. Use it:**

```typescript
import { GraphQLClient } from "gqlkit-ts";
import { QueryRoot } from "./sdk/queries";

const client = new GraphQLClient("https://your-api.example.com/graphql", {
  authToken: "YOUR_TOKEN",
});

const qr = new QueryRoot(client);

const todos = await qr
  .todos()
  .filter({ done: false })
  .select((t) =>
    t.id().text().done().user((u) => u.id().name())
  )
  .execute();
```

Full guide: [docs/getting-started-typescript.md](./docs/getting-started-typescript.md)

## Why GQLKit?

Existing GraphQL code generators like [genqlient](https://github.com/Khan/genqlient) (Go) and [GraphQL Code Generator](https://the-guild.dev/graphql/codegen) (TypeScript) take a **query-first** approach: you write every query as a static string upfront, then run codegen to produce types for each one. This creates real problems as your project grows.

### The duplicate types problem

In genqlient, every query generates its own unique types — even when they return the same GraphQL type. Two queries that both fetch a `User` produce two completely separate Go structs:

```graphql
# Two queries, same underlying User type
query GetUser($id: ID!)   { user(id: $id)   { id name email } }
query GetViewer            { viewer           { id name email } }
```

```go
// genqlient generates two unrelated structs — you can't pass one where the other is expected
type GetUserUser struct { Id string; Name string; Email string }
type GetViewerViewerUser struct { Id string; Name string; Email string }
```

With GQLKit, `User` is just `User`. One shared type across all queries:

```go
// GQLKit — one type, used everywhere
users, _ := qr.Users().Select(func(u *fields.UserFields) { u.ID().Name().Email() }).Execute(ctx)
user, _  := qr.User().ID("1").Select(func(u *fields.UserFields) { u.ID().Name().Email() }).Execute(ctx)
// Both return types.User
```

### Static queries vs. dynamic field selection

With genqlient and GraphQL Code Generator, field selection is locked at build time. Want the same query with different fields? Write another query, run codegen again, get another set of types.

GQLKit lets you choose fields at call time with full type safety:

```go
// Lightweight list view — only fetch what you need
qr.Todos().Select(func(f *fields.TodoFields) { f.ID().Text() }).Execute(ctx)

// Detail view — same query, more fields, no codegen step
qr.Todos().Select(func(f *fields.TodoFields) {
    f.ID().Text().Done().Priority().Tags().User(func(u *fields.UserFields) {
        u.ID().Name().Email().Role()
    })
}).Execute(ctx)
```

```typescript
// TypeScript — same flexibility, compile-time narrowed return types
const light = await qr.todos().select((t) => t.id().text()).execute();
const full  = await qr.todos().select((t) => t.id().text().done().user((u) => u.id().name())).execute();
// light.done → compile error (not selected)
// full.done  → boolean (selected)
```

### Comparison

| | GQLKit | [genqlient](https://github.com/Khan/genqlient) (Go) | [GraphQL Code Generator](https://the-guild.dev/graphql/codegen) (TS) |
|---|---|---|---|
| **Approach** | Schema-first — generates builders from the schema | Query-first — generates types from predefined `.graphql` operations | Query-first — generates types from predefined operations |
| **Field selection** | Dynamic at call time, type-safe | Static, locked at codegen time | Static, locked at codegen time |
| **Type per GraphQL type** | One shared type (e.g., `User`) | One per query path (e.g., `GetUserUser`, `GetViewerViewerUser`) | One per operation (e.g., `GetUserQuery`, `GetViewerQuery`) |
| **Adding a new query** | Just call the builder — no codegen step | Write `.graphql` file, re-run codegen | Write query string, re-run codegen |
| **Changing field selection** | Change the `.Select()` call | Write a new query variant, re-run codegen | Write a new query variant, re-run codegen |
| **Config complexity** | One `config.jsonc` for scalar mappings | `genqlient.yaml` + `@genqlient` directives per field | 60+ plugins; typical project needs 2–5 configured together |
| **Output structure** | Organized packages (`queries/`, `fields/`, `types/`, etc.) | Single `generated.go` file | Varies by plugin; often one large file |
| **Schema introspection** | Built-in (`gqlkit-sdl fetch`) | Not supported — must provide local SDL files | Supported via config |
| **Runtime overhead** | Minimal — lightweight HTTP client | Minimal | Generated code includes duplicate query strings; needs Babel/SWC plugin to optimize |
| **Languages** | Go + TypeScript from same schema | Go only | TypeScript/JavaScript only |

### TL;DR

- **genqlient / GraphQL Code Generator**: You write queries as static strings → codegen produces types for each one → types proliferate → changing fields means re-running codegen.
- **GQLKit**: You generate builders once from the schema → compose queries in code with dynamic field selection → one type per GraphQL type → no codegen loop for day-to-day work.

### AI-friendly by design

GQLKit's builder pattern works naturally with AI coding assistants (Copilot, Cursor, Claude). With query-first tools, the AI has to write raw GraphQL strings in a separate file, then you manually run codegen before the types exist — breaking the AI's flow. With GQLKit, the AI just writes code:

```
"fetch todos with user names" →

qr.Todos().Select(func(f *fields.TodoFields) {
    f.ID().Text().User(func(u *fields.UserFields) { u.ID().Name() })
}).Execute(ctx)
```

No `.graphql` files to create, no codegen to run, no types that don't exist yet. The builder API is fully discoverable from method signatures — the AI sees `.ID()`, `.Text()`, `.Done()`, `.User()` and chains them directly.

Deep dive: [docs/ai-friendly.md](./docs/ai-friendly.md)

## Key Features

- **Type-safe field selection** — only selected fields exist on the return type; unselected fields are compile-time errors
- **Builder pattern** — fluent API for queries, mutations, arguments, and nested field selection
- **Go + TypeScript** — generate SDKs for both languages from the same schema
- **Schema introspection** — fetch schemas from any GraphQL endpoint with `gqlkit-sdl`
- **Custom scalar mappings** — configure how GraphQL scalars map to language types via `config.jsonc`
- **Zero runtime overhead** — generated code with minimal dependencies

## Tools

| Tool | Description |
|------|-------------|
| `gqlkit` | CLI — generates Go and TypeScript SDKs from GraphQL schemas |
| `gqlkit-sdl` | CLI — fetches GraphQL schemas via introspection |
| [`gqlkit-ts`](https://www.npmjs.com/package/gqlkit-ts) | npm package — lightweight TypeScript runtime for generated SDKs |

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) for development setup, architecture, and how to run the examples.
