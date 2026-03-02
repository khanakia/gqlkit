# Builder

Go package for building GraphQL queries and mutations with a fluent API. It produces operation strings and a variables map suitable for sending to a GraphQL endpoint.

## Overview

* **FieldSelection** – Builds field selection sets (scalar fields and nested selections). Child keys are emitted in sorted order for deterministic output.
* **BaseBuilder** – Builds full operations: `query` or `mutation`, with optional variables and a field selection.

## Installation

```go
import "your-module/gqlcustom/pkg/builder"
```

***

## FieldSelection

Use `FieldSelection` to describe which fields to request. You can add scalar fields and nested selections (children).

### Scalar fields only

```go
sel := builder.NewFieldSelection()
sel.AddField("id")
sel.AddField("name")
sel.AddField("email")
fmt.Println(sel.Build(0))
```

**Output:**

```
id
name
email
```

### With indent

```go
sel := builder.NewFieldSelection()
sel.AddField("id")
sel.AddField("name")
fmt.Println(sel.Build(1))
```

**Output:**

```
  id
  name
```

### Nested selection (child)

```go
sel := builder.NewFieldSelection()
sel.AddField("id")
sel.AddField("name")
addr := builder.NewFieldSelection()
addr.AddField("street")
addr.AddField("city")
addr.AddField("zip")
sel.AddChild("address", addr)
fmt.Println(sel.Build(0))
```

**Output:**

```
id
name
address {
  street
  city
  zip
}
```

### Deep nesting

```go
sel := builder.NewFieldSelection()
sel.AddField("id")
country := builder.NewFieldSelection()
country.AddField("code")
country.AddField("name")
region := builder.NewFieldSelection()
region.AddField("name")
region.AddChild("country", country)
sel.AddChild("address", region)
fmt.Println(sel.Build(0))
```

**Output:**

```
id
address {
  country {
    code
    name
  }
  name
}
```

***

## BaseBuilder: Query

### Query with no arguments and no selection

```go
b := builder.NewBaseBuilder("query", "GetViewer", "viewer")
query := b.BuildQuery()
fmt.Println(query)
```

**Full query output:**

```graphql
query GetViewer {
  viewer

}
```

### Query with arguments

```go
b := builder.NewBaseBuilder("query", "GetUser", "user")
b.SetArg("id", "user-123", "ID!")
query := b.BuildQuery()
vars := b.GetVariables()
fmt.Println(query)
fmt.Println("Variables:", vars)
```

**Full query output:**

```graphql
query GetUser($id: ID!) {
  user(id: $id)

}
```

**Variables:** `map[id:user-123]`

### Query with field selection

```go
b := builder.NewBaseBuilder("query", "GetUser", "user")
b.SetArg("id", "user-123", "ID!")
sel := b.GetSelection()
sel.AddField("id")
sel.AddField("name")
sel.AddField("email")
query := b.BuildQuery()
fmt.Println(query)
```

**Full query output:**

```graphql
query GetUser($id: ID!) {
  user(id: $id) {
  id
  name
  email
  }
}
```

### Query with nested selection

```go
b := builder.NewBaseBuilder("query", "GetUser", "user")
b.SetArg("id", "user-123", "ID!")
sel := b.GetSelection()
sel.AddField("id")
sel.AddField("name")
addr := builder.NewFieldSelection()
addr.AddField("street")
addr.AddField("city")
addr.AddField("zip")
sel.AddChild("address", addr)
query := b.BuildQuery()
fmt.Println(query)
```

**Full query output:**

```graphql
query GetUser($id: ID!) {
  user(id: $id) {
  id
  name
  address {
    street
    city
    zip
  }
  }
}
```

### Query with multiple variables

```go
b := builder.NewBaseBuilder("query", "ListProducts", "products")
b.SetArg("first", 10, "Int")
b.SetArg("filter", "active", "String")
sel := b.GetSelection()
sel.AddField("id")
sel.AddField("name")
query := b.BuildQuery()
vars := b.GetVariables()
fmt.Println(query)
fmt.Println("Variables:", vars)
```

**Full query output:**

```graphql
query ListProducts($filter: String, $first: Int) {
  products(filter: $filter, first: $first) {
  id
  name
  }
}
```

**Variables:** `map[filter:active first:10]`

***

## BaseBuilder: Mutation

### Mutation with input and selection

```go
b := builder.NewBaseBuilder("mutation", "CreateUser", "createUser")
b.SetArg("input", map[string]interface{}{
	"name":  "Jane",
	"email": "jane@example.com",
}, "CreateUserInput!")
sel := b.GetSelection()
sel.AddField("id")
sel.AddField("name")
sel.AddField("email")
query := b.BuildQuery()
vars := b.GetVariables()
fmt.Println(query)
fmt.Println("Variables:", vars)
```

**Full mutation output:**

```graphql
mutation CreateUser($input: CreateUserInput!) {
  createUser(input: $input) {
  id
  name
  email
  }
}
```

**Variables:** `map[input:map[email:jane@example.com name:Jane]]`

### Mutation with multiple arguments

```go
b := builder.NewBaseBuilder("mutation", "UpdateProduct", "updateProduct")
b.SetArg("id", "prod-1", "ID!")
b.SetArg("input", map[string]interface{}{"name": "New Name"}, "UpdateProductInput!")
sel := b.GetSelection()
sel.AddField("id")
sel.AddField("name")
query := b.BuildQuery()
fmt.Println(query)
```

**Full mutation output:**

```graphql
mutation UpdateProduct($id: ID!, $input: UpdateProductInput!) {
  updateProduct(id: $id, input: $input) {
  id
  name
  }
}
```

***

## Sending the request

Use `BuildQuery()` and `GetVariables()` with your HTTP client or GraphQL client:

```go
b := builder.NewBaseBuilder("query", "GetUser", "user")
b.SetArg("id", "user-123", "ID!")
b.GetSelection().AddField("id")
b.GetSelection().AddField("name")

query := b.BuildQuery()
variables := b.GetVariables()

// Example payload for a GraphQL HTTP request
payload := map[string]interface{}{
	"query":     query,
	"variables": variables,
}
// Send payload as JSON to your GraphQL endpoint
```

***

## API summary

| API | Description |
|-----|-------------|
| `NewFieldSelection()` | New empty field selection |
| `AddField(name)` | Add a scalar field |
| `AddChild(name, child)` | Add a nested selection |
| `Build(indent int)` | Build selection string with given indent level |
| `NewBaseBuilder(opType, opName, fieldName)` | New query/mutation builder (`opType`: `"query"` or `"mutation"`) |
| `SetArg(name, value, graphqlType)` | Set a variable and its GraphQL type |
| `GetSelection()` | Get the root field selection to add fields/children |
| `BuildQuery()` | Build the full operation string |
| `GetVariables()` | Get the variables map for the request body |
