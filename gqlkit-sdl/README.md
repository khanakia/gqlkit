# gqlkit-sdl

A CLI tool that fetches a GraphQL schema from a live endpoint via introspection and converts it to SDL (Schema Definition Language) format.

## Installation

### Quick install (macOS / Linux)

```bash
curl -sL https://raw.githubusercontent.com/khanakia/gqlkit/main/gqlkit-sdl/install.sh | sh
```

### Manual download

Download the latest release for your platform from [GitHub Releases](https://github.com/khanakia/gqlkit/releases).

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
gqlkit-sdl fetch --url "https://graphql.anilist.co"

# With custom output file
gqlkit-sdl fetch --url "https://graphql.anilist.co" --output my-schema.graphql

# With authentication
gqlkit-sdl fetch --url "https://graphql.anilist.co" \
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

introspectionSchema, err := schema.FetchSchema("https://graphql.anilist.co", opts)
if err != nil {
    log.Fatal(err)
}

sdl := schema.ConvertToSDL(introspectionSchema)
err = schema.SaveToFile(sdl, "schema.graphql")
```
