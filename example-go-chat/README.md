# example-go-chat — Go SDK Example (Production Schema)

Demonstrates generating a Go SDK from a production-scale chatbot GraphQL schema with complex types, Relay-style cursor pagination, and many operations.

## Structure

```
cmd/
  generate/
    main.go              SDK generator entry point
    schema.graphql       Production chatbot GraphQL schema
    config.jsonc         Scalar bindings (Cursor, Time, Password, JSON, etc.)
  samples/
    main.go              Sample queries demonstrating the SDK
sdk/                     Generated SDK (do not edit)
  builder/               FieldSelection + BaseBuilder runtime
  scalars/               Custom scalars (Cursor, Password, JSON, Time, etc.)
  enums/                 Order field enums (ChatbotOrderField, etc.)
  types/                 ~20 Go structs (Chatbot, Channel, User, etc.)
  inputs/                Input structs for ordering, filtering, etc.
  fields/                87 field selector files (complex nested types)
  queries/               47 query builders + QueryRoot
  mutations/             57 mutation builders + MutationRoot
```

## Generate

```bash
go run ./cmd/generate
```

## SDK Usage

```go
client := graphqlclient.NewClient("http://localhost:2310/api/sa/query",
    graphqlclient.WithHeaders(map[string]string{
        "workspace": "workspace-id",
        "Authorization": "Bearer <jwt>",
    }),
)

qr := queries.NewQueryRoot(client)

// Relay cursor pagination with ordering
chatbots, _ := qr.Chatbots().
    First(intPtr(10)).
    OrderBy(&inputs.ChatbotOrder{
        Field: enums.ChatbotOrderFieldCreatedAt,
    }).
    Select(func(conn *fields.ChatbotConnectionFields) {
        conn.TotalCount().
            Edges(func(e *fields.ChatbotEdgeFields) {
                e.Cursor().Node(func(c *fields.ChatbotFields) {
                    c.ID().Name().CreatedAt().UpdatedAt()
                })
            }).
            PageInfo(func(p *fields.PageInfoFields) {
                p.HasNextPage().HasPreviousPage().StartCursor().EndCursor()
            })
    }).
    Execute(ctx)
```

## Custom Scalars

Configured in `cmd/generate/config.jsonc`:

| GraphQL Scalar | Go Type |
|---------------|---------|
| `Cursor` | `string` |
| `Time` | `time.Time` |
| `Password` | `string` |
| `JSON` | `encoding/json.RawMessage` |
| `Uint64` | `uint64` |
