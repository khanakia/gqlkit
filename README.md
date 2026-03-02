# GraphQL Client Go Workspace

A Go workspace containing tools for working with GraphQL schemas.

## Modules

| Module | Description |
|--------|-------------|
| [gqlsdl](./gqlsdl) | CLI tool to fetch GraphQL schema via introspection and save as SDL |
| [gqlsdk](./gqlsdk) | GraphQL SDK (coming soon) |

## Requirements

* Go 1.21+
* [Task](https://taskfile.dev) (optional, for task runner)

## Quick Start

```bash
# Build all modules
task build

# Or build individually
task gqlsdl:build
task gqlsdk:build
```

## Available Tasks

```bash
task gqlsdl:run      # Run gqlsdl with go run .
task gqlsdl:build    # Build gqlsdl to bin/

task gqlsdk:run      # Run gqlsdk with go run .
task gqlsdk:build    # Build gqlsdk to bin/

task build           # Build all modules to bin/
task tidy            # Run go mod tidy on all modules
task upgrade         # Update all dependencies
```

## Project Structure

```
.
├── Taskfile.yml     # Task runner configuration
├── go.work          # Go workspace file
├── gqlsdl/          # Schema fetcher module
│   ├── go.mod
│   ├── main.go
│   └── schema/
└── gqlsdk/          # SDK module
    └── go.mod
```

## Graphql Types

* Types
* Scalars
* Enum
* Input Types
* Field Selector
* Query Builder
* Mutation Builder

## SDK EXAMPLE

This is the example module to test the sdk

Generate the sdk in `sdkexample/sdk`

```sh
cd sdkexample
go run ./cmd/generate
```

Run Test Queries

```sh
cd sdkexample
go run ./cmd/samples
```
