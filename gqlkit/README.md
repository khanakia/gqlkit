# gqlkit

A CLI tool that generates type-safe Go and TypeScript client SDKs from GraphQL SDL schema files.

## Installation

### Quick install (macOS / Linux)

```bash
curl -sL https://raw.githubusercontent.com/khanakia/gqlkit/main/gqlkit/install.sh | sh
```

### Manual download

Download the latest release for your platform from [GitHub Releases](https://github.com/khanakia/gqlkit/releases).

### From source

```bash
go install gqlkit/cmd/cli@latest
```

## Usage

```bash
gqlkit <command> [flags]
```

### Commands

| Command    | Description                              |
|------------|------------------------------------------|
| `generate` | Generate Go/TypeScript SDK from a schema |
| `version`  | Print version and exit                   |

### Flags (generate)

| Flag                  | Short | Required | Default | Description                             |
|-----------------------|-------|----------|---------|-----------------------------------------|
| `--schema`            | `-s`  | Yes      | —       | Path to GraphQL SDL file                |
| `--output`            | `-o`  | No       | `./sdk` | Output directory for generated SDK      |
| `--package`           | `-p`  | No       | `sdk`   | Go package name for generated SDK       |
| `--module`            | `-m`  | No       | —       | Go module path for generated SDK        |

### Examples

```bash
# Check version
gqlkit version

# Generate Go SDK
gqlkit generate --schema schema.graphql

# Generate with custom output and package
gqlkit generate --schema schema.graphql --output ./client --package client

# Generate with full module path
gqlkit generate \
  --schema schema.graphql \
  --output ./sdk \
  --package sdk \
  --module github.com/myorg/myproject/sdk
```

## Generated SDK Usage

### Go

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

### TypeScript

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

## Configuration

Each SDK project can have a `config.jsonc` file for custom scalar bindings:

```jsonc
{
  "bindings": {
    "Time": { "model": "time.Time" },
    "JSON": { "model": "encoding/json.RawMessage" }
  }
}
```
