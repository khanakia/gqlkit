# gqlkit-ts — TypeScript GraphQL Client Runtime

The TypeScript runtime library that generated TypeScript SDKs depend on. Published as an npm package that provides the base classes and HTTP client used by all generated code.

## Exports

```typescript
import {
  GraphQLClient,    // HTTP client for executing GraphQL queries
  GraphQLErrors,    // Error class wrapping GraphQL error responses
  ClientOptions,    // Configuration type for GraphQLClient
  FieldSelection,   // Tracks selected fields for query building
  BaseBuilder,      // Base class for generated operation builders
} from "gqlkit-ts";
```

## Components

### `GraphQLClient`

Lightweight HTTP client for GraphQL endpoints.

```typescript
const client = new GraphQLClient("http://localhost:8081/query", {
  authToken: "Bearer <token>",    // Sets Authorization header
  headers: { "X-Custom": "val" }, // Additional headers
  fetch: customFetch,             // Custom fetch implementation (optional)
});

// Typed query execution
const data = await client.execute<{ user: User }>(query, variables);

// Raw query (returns unknown)
const raw = await client.rawQuery(query, variables);
```

**Error handling:** Throws `GraphQLErrors` when the response contains a non-empty `errors` array. Each error includes `message`, `locations`, `path`, and `extensions`.

### `FieldSelection`

Tracks which fields to include in a GraphQL selection set. Used internally by generated field selector classes.

```typescript
const sel = new FieldSelection();
sel.addField("id");
sel.addField("name");

const nested = new FieldSelection();
nested.addField("id");
sel.addChild("user", nested);

sel.build(0);
// → "id\nname\nuser {\n  id\n}"
```

### `BaseBuilder`

Base class that all generated query/mutation builders extend. Handles:
- Argument storage with GraphQL type annotations
- Field selection management
- Query string assembly (operation name, variable declarations, argument passing, field selection)
- Execution via the provided `GraphQLClient`

```typescript
// Generated code extends BaseBuilder:
class TodosBuilder extends BaseBuilder {
  filter(v: TodoFilter) { this.setArg("filter", v, "TodoFilter"); return this; }
  select(fn: (f: TodoFields) => void) { fn(new TodoFields(this.getSelection())); return this; }
  async execute() { return this.executeRaw(); }
}
```

## Build

```bash
npm install
npm run build    # Compiles to dist/
```

## Package Structure

```
src/
  index.ts           Public API re-exports
  graphqlclient.ts   GraphQLClient, GraphQLErrors, ClientOptions
  builder.ts         FieldSelection, BaseBuilder
dist/                Compiled JS + type declarations
```

## Configuration

- **Target:** ES2020
- **Module:** CommonJS
- **TypeScript:** 5.4.0+
- **No runtime dependencies** (uses native `fetch`)
