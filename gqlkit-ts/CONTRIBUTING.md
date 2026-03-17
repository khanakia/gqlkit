# Contributing to gqlkit-ts

## Architecture

The library provides the runtime foundation for generated TypeScript GraphQL SDKs. Generated code imports and extends these base classes to create typed, ergonomic query builders.

```
GraphQLClient          HTTP transport — sends { query, variables } as JSON POST
      ▲
      │ used by
BaseBuilder            Assembles operation string + variables, executes via client
      ▲
      │ extended by
Generated SDK          Typed setters, field selectors, exec() methods
```

## Project Structure

```
src/
  index.ts             Public API re-exports
  graphqlclient.ts     GraphQLClient, GraphQLErrors, ClientOptions
  builder.ts           FieldSelection, BaseBuilder
dist/                  Compiled JS + type declarations (generated)
```

## Key Components

### `graphqlclient.ts`

- `GraphQLClient` — Sends POST requests with JSON-encoded `{ query, variables }`. Merges custom headers and optional Bearer auth. Supports injecting a custom `fetch` for SSR or testing.
- `GraphQLErrors` — Custom error class thrown when the response contains a non-empty `errors` array. Aggregates all error messages into a single string.
- `ClientOptions` — Configuration interface for headers, auth token, and custom fetch.

### `builder.ts`

- `FieldSelection` — Recursive tree structure tracking scalar fields and nested object selections. Serializes into a GraphQL selection set string with proper indentation.
- `BaseBuilder` — Base class for generated operation builders. Stores arguments with their GraphQL type annotations, manages the field selection tree, builds the complete operation string (variable declarations, argument passing, selection set), and executes via `GraphQLClient`.

## Development

### Prerequisites

- Node.js 18+
- npm

### Setup

```bash
cd gqlkit-ts
npm install
```

### npm Scripts

| Script | Command | Description |
|---|---|---|
| `build` | `npm run build` | Compile TypeScript to `dist/` |
| `clean` | `npm run clean` | Remove `dist/` directory |
| `rebuild` | `npm run rebuild` | Clean + build from scratch |
| `pack:preview` | `npm run pack:preview` | Preview files that will be published |
| `publish:patch` | `npm run publish:patch` | Bump patch version and publish (0.1.0 → 0.1.1) |
| `publish:minor` | `npm run publish:minor` | Bump minor version and publish (0.1.0 → 0.2.0) |
| `publish:major` | `npm run publish:major` | Bump major version and publish (0.1.0 → 1.0.0) |

### Test locally before publishing

```bash
# Preview what files will be included in the package
npm run pack:preview

# Create a tarball and install it in another project
npm pack
cd /path/to/test-project
npm install /path/to/gqlkit-ts-0.1.0.tgz
```

## Publishing to npm

Publish with a version bump in one step:

```bash
cd gqlkit-ts
npm run publish:patch    # bug fixes
npm run publish:minor    # new features
npm run publish:major    # breaking changes
```

Or manually:

```bash
npm version patch
npm publish
```

The `prepublishOnly` script runs `tsc` automatically before publishing.

## Design Principles

- **Zero dependencies** — Uses native `fetch`, no external packages
- **Minimal surface** — Only export what generated SDKs need
- **CommonJS output** — Broadest compatibility (ES2020 target)
- **Type-safe** — Full TypeScript declarations included in the package

## Dependencies

- [typescript](https://www.typescriptlang.org/) — Build-time only (devDependency)
