# gqlkit-sdl

A CLI tool that fetches a GraphQL schema from a live endpoint via introspection and converts it to SDL (Schema Definition Language) format.

## Installation

### Download binary

Download the latest release for your platform from [GitHub Releases](https://github.com/khanakia/gqlkit/releases).

```bash
# macOS (Apple Silicon)
curl -sL https://github.com/khanakia/gqlkit/releases/latest/download/gqlkit-sdl_darwin_arm64.tar.gz | tar xz

# macOS (Intel)
curl -sL https://github.com/khanakia/gqlkit/releases/latest/download/gqlkit-sdl_darwin_amd64.tar.gz | tar xz

# Linux (amd64)
curl -sL https://github.com/khanakia/gqlkit/releases/latest/download/gqlkit-sdl_linux_amd64.tar.gz | tar xz

# Linux (arm64)
curl -sL https://github.com/khanakia/gqlkit/releases/latest/download/gqlkit-sdl_linux_arm64.tar.gz | tar xz
```

### From source

```bash
go install gqlkit-sdl@latest
```

## Usage

```bash
gqlkit-sdl <command> [flags]
```

### Commands

| Command   | Description                  |
|-----------|------------------------------|
| `fetch`   | Fetch schema and save as SDL |
| `version` | Print version and exit       |

### Flags (fetch)

| Flag              | Required | Default          | Description                                  |
|-------------------|----------|------------------|----------------------------------------------|
| `--url`           | Yes      | —                | GraphQL endpoint URL                         |
| `--output`        | No       | `schema.graphql` | Output file path                             |
| `-H`, `--header`  | No       | —                | HTTP header in `Key:Value` format (repeatable) |

### Examples

```bash
# Check version
gqlkit-sdl version

# Basic usage
gqlkit-sdl fetch --url "https://api.example.com/graphql"

# With custom output file
gqlkit-sdl fetch --url "https://api.example.com/graphql" --output my-schema.graphql

# With authentication
gqlkit-sdl fetch --url "https://api.example.com/graphql" \
  -H "Authorization: Bearer your-token"

# With multiple headers
gqlkit-sdl fetch --url "http://localhost:2310/sa/query_playground" \
  -H "Authorization: Bearer your-token" \
  -H "Referer: http://localhost:2310/sa/gql?pkey=1234" \
  -H "Origin: http://localhost:2310"
```

## Go API

The `schema` package can also be used programmatically:

```go
import "gqlkit-sdl/schema"

opts := &schema.FetchOptions{
    Headers: map[string]string{
        "Authorization": "Bearer token",
    },
}

introspectionSchema, err := schema.FetchSchema("https://api.example.com/graphql", opts)
if err != nil {
    log.Fatal(err)
}

sdl := schema.ConvertToSDL(introspectionSchema)
err = schema.SaveToFile(sdl, "schema.graphql")
```
