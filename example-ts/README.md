# example-ts — TypeScript SDK Example

Demonstrates generating and using a TypeScript SDK from the [mockapi](../mockapi) test GraphQL server. The generated SDK provides a fully typed, fluent API where **only selected fields appear in the return type** — unselected fields are compile-time errors.

## Quick Start

```bash
# 1. Start the mock API (in another terminal)
cd ../mockapi && go run server.go

# 2. Build the runtime library
cd ../gqlkit-ts && npm install && npm run build

# 3. Install dependencies + generate + type-check
cd ../example-ts
npm install
go run ./cmd/generate
npx tsc --noEmit

# 4. Run sample queries
npm run samples
```

Or use Taskfile:

```bash
task example-ts:setup    # First-time setup (steps 2-3)
task example-ts:run      # Run samples
```

## Structure

```
cmd/
  generate/
    main.go              TypeScript SDK generator entry point
    schema.graphql       GraphQL schema (same as mockapi)
    config.jsonc         Scalar type bindings (GraphQL → TypeScript)
  samples/
    main.ts              Sample queries and mutations
    type-test.ts         Compile-time type safety verification
sdk/                     Generated SDK (do not edit)
  builder/               Re-exports from gqlkit-ts runtime
  scalars/               TypeScript scalar type aliases
  enums/                 TypeScript enums
  types/                 TypeScript interfaces (Todo, User, etc.)
  inputs/                TypeScript input interfaces
  fields/                Field selector classes
  queries/               Query builder classes + QueryRoot
  mutations/             Mutation builder classes + MutationRoot
```

## SDK Usage Examples

```typescript
import { GraphQLClient } from "gqlkit-ts";
import { QueryRoot } from "./sdk/queries";
import { MutationRoot } from "./sdk/mutations";

const client = new GraphQLClient("http://localhost:8081/query");
const qr = new QueryRoot(client);
const mr = new MutationRoot(client);

// Simple queries
const pong = await qr.ping().execute();
const result = await qr.sum().a(10).b(20).execute();

// Query with nested field selection
const todos = await qr.todosConnection()
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

// Mutation
const created = await mr.createTodo()
    .input({ text: "Buy milk", userId: "1" })
    .select((t) => t.id().text().done())
    .execute();
```

## Type Safety

The SDK narrows return types based on field selection:

```typescript
// Only selected fields exist on the result type
const todo = await qr.todo().id("1")
    .select((t) => t.id().text())
    .execute();

todo.id;    // ✓ string
todo.text;  // ✓ string
todo.done;  // ✗ compile error — not selected
```

See `cmd/samples/type-test.ts` for comprehensive type safety tests.

## Configuration

`cmd/generate/config.jsonc` maps GraphQL scalars to TypeScript types:

```jsonc
{
  "bindings": {
    "DateTime": "string",
    "JSON": "Record<string, unknown>"
  }
}
```
