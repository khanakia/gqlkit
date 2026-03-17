# example-go-mockapi — Go SDK Example (Mock API)

Demonstrates generating and using a Go SDK from the [mockapi](../mockapi) test GraphQL server.

## Quick Start

```bash
# 1. Start the mock API (in another terminal)
cd ../mockapi && go run server.go

# 2. Generate the SDK
go run ./cmd/generate

# 3. Run sample queries
go run ./cmd/samples
```

Or via Taskfile:

```bash
task mockapi:run                    # Start API
task example-go-mockapi:fetch-schema  # Fetch schema from running API
task example-go-mockapi:generate      # Generate SDK
task example-go-mockapi:run           # Run sample queries
```

## Structure

```
cmd/
  generate/
    main.go              SDK generator entry point
    schema.graphql       GraphQL schema (fetched from mockapi)
    config.jsonc         Scalar type bindings
  samples/
    main.go              Sample queries and mutations demonstrating the SDK
sdk/                     Generated SDK (do not edit)
  builder/               FieldSelection + BaseBuilder runtime
  scalars/               Scalar type aliases
  enums/                 Role enum (ADMIN, USER, GUEST)
  types/                 Go structs (Todo, User, PageInfo, etc.)
  inputs/                Input structs (NewTodo, TodoFilter, etc.)
  fields/                Type-safe field selectors
  queries/               Query builders + QueryRoot
  mutations/             Mutation builders + MutationRoot
```

## SDK Usage Examples

```go
client := graphqlclient.NewClient("http://localhost:8081/query")
qr := queries.NewQueryRoot(client)
mr := mutations.NewMutationRoot(client)

// Simple scalar query
ping, _ := qr.Ping().Execute(ctx)

// Query with args
sum, _ := qr.Sum().A(10).B(20).Execute(ctx)

// Query with filter, pagination, and nested field selection
todos, _ := qr.Todos().
    Filter(&inputs.TodoFilter{TextContains: strPtr("todo")}).
    Pagination(&inputs.PaginationInput{Limit: 10}).
    Select(func(f *fields.TodoFields) {
        f.ID().Text().Done().User(func(u *fields.UserFields) {
            u.ID().Name()
        })
    }).
    Execute(ctx)

// Relay-style cursor pagination
conn, _ := qr.TodosConnection().
    Select(func(c *fields.TodoConnectionFields) {
        c.TotalCount().
            Edges(func(e *fields.TodoEdgeFields) {
                e.Cursor().Node(func(t *fields.TodoFields) {
                    t.ID().Text()
                })
            }).
            PageInfo(func(p *fields.PageInfoFields) {
                p.HasNextPage().EndCursor()
            })
    }).
    Execute(ctx)

// Mutation
created, _ := mr.CreateTodo().
    Input(inputs.NewTodo{Text: "New todo", UserID: "1"}).
    Select(func(f *fields.TodoFields) { f.ID().Text() }).
    Execute(ctx)
```

## Fetching the Schema

To re-fetch the schema from a running mockapi instance:

```bash
task example-go-mockapi:fetch-schema
# or manually:
go run ../gqlkit-sdl -url http://localhost:8081/query -output cmd/generate/schema.graphql
```
