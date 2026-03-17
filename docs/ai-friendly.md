# Why GQLKit Is AI-Friendly

AI coding assistants (Copilot, Cursor, Claude) are increasingly writing the majority of application code. GQLKit's architecture is a natural fit for this workflow, while query-first tools like genqlient and GraphQL Code Generator create friction that breaks AI-assisted development.

---

## The problem with query-first tools + AI

With genqlient or GraphQL Code Generator, writing a GraphQL query is a multi-step process:

```
1. AI writes a .graphql query string       →  raw text, no type feedback
2. User manually runs codegen              →  AI can't do this step
3. Types now exist in generated files      →  AI can finally reference them
4. AI writes Go/TS code using those types  →  working code
5. Want to change a field?                 →  back to step 1
```

**Step 2 is the bottleneck.** The AI assistant cannot run `go generate` or `npx graphql-codegen` for you. This means:

- The AI writes code that **references types that don't exist yet** — red squiggles everywhere until you run codegen
- The AI is writing **two languages** (GraphQL query strings + Go/TypeScript) across **two separate files** — more context switching, more room for mistakes
- Every field change requires a **round trip** through codegen — the AI can't iterate quickly
- The AI has **no way to discover** what fields are available when writing a raw GraphQL string. It has to know the schema from memory or guess

This workflow was designed for humans who run codegen as a build step. It was never designed for AI assistants that generate code in a single pass.

## How GQLKit solves this

With GQLKit, the entire query is just code — no separate files, no codegen step:

```go
todos, err := qr.Todos().
    Filter(&inputs.TodoFilter{Done: boolPtr(false)}).
    Select(func(f *fields.TodoFields) {
        f.ID().Text().Done().User(func(u *fields.UserFields) {
            u.ID().Name()
        })
    }).
    Execute(ctx)
```

```typescript
const todos = await qr
  .todos()
  .filter({ done: false })
  .select((t) => t.id().text().done().user((u) => u.id().name()))
  .execute();
```

The AI generates this in one shot. No intermediate steps.

---

## Why the builder pattern works well for AI

### 1. Discoverable API surface

The generated SDK exposes every query, mutation, field, and argument as typed methods. An AI assistant can see:

- `qr.Todos()` → returns a `TodosBuilder`
- `TodosBuilder` has `.Filter()`, `.Pagination()`, `.Select()`, `.Execute()`
- `.Select()` takes a function with `TodoFields`
- `TodoFields` has `.ID()`, `.Text()`, `.Done()`, `.User()`, `.Priority()`, `.Tags()`

The AI doesn't need to memorize the GraphQL schema. The **method signatures are the schema**. Each method is a concrete signal for what's available and what types are expected.

### 2. Fluent chains give strong sequential context

Builder patterns produce highly predictable code. After seeing `qr.Todos()`, the AI knows the next steps are arguments (`.Filter()`, `.Pagination()`), then field selection (`.Select()`), then execution (`.Execute()`). This sequential pattern is exactly what language models are good at — predicting the next token in a chain.

Compare this to writing a raw GraphQL string where the AI has to produce correct syntax, remember field names, and match argument types — all inside a string with no type checking.

### 3. One type per GraphQL type

genqlient generates names like `GetUserUser`, `GetViewerViewerUser`, `ListUsersUsersUser` for the same underlying `User` type. An AI has to figure out which query-specific variant to use in each context.

GQLKit generates one `User` type. The AI just uses `User` everywhere. Less ambiguity, fewer mistakes.

### 4. No context switching between files

With query-first tools, the AI has to:
- Open/create a `.graphql` file
- Write the query string
- Switch back to the Go/TypeScript file
- Import the generated types
- Write the actual code

With GQLKit, everything is in one place. The AI writes the query, selects fields, and handles the result — all in the same file, in the same language.

### 5. Changing fields is a code edit, not a rebuild

When an AI (or human) wants to add a field to a query:

**genqlient / GraphQL Code Generator:**
- Edit the `.graphql` file to add the field
- Re-run codegen to regenerate types
- Update the consuming code to use the new field

**GQLKit:**
- Add `.Email()` to the `.Select()` chain

One line change, no rebuild, immediate type safety.

---

## Side-by-side: AI writing the same query

**Task:** "Fetch all todos that aren't done, with their user's name"

### genqlient (AI needs two steps + manual codegen)

Step 1 — AI writes `queries/todos.graphql`:
```graphql
query GetTodos($done: Boolean) {
  todos(filter: { done: $done }) {
    id
    text
    done
    user {
      id
      name
    }
  }
}
```

Step 2 — **User manually runs** `go generate ./...`

Step 3 — AI writes Go code:
```go
resp, err := GetTodos(ctx, client, false)
for _, t := range resp.Todos {
    fmt.Println(t.Text, t.User.Name)
}
```

If the AI writes step 3 before step 2, the code won't compile — `GetTodos`, `GetTodosResponse`, and all field types don't exist yet.

### GQLKit (AI writes once, done)

```go
todos, err := qr.Todos().
    Filter(&inputs.TodoFilter{Done: boolPtr(false)}).
    Select(func(f *fields.TodoFields) {
        f.ID().Text().Done().User(func(u *fields.UserFields) {
            u.ID().Name()
        })
    }).
    Execute(ctx)

for _, t := range todos {
    fmt.Println(t.Text, t.User.Name)
}
```

Compiles immediately. No intermediate step.

---

## Summary

| | GQLKit | Query-first tools (genqlient, GraphQL Code Generator) |
|---|---|---|
| **AI writes query as** | Typed method chains (code) | Raw GraphQL strings (text) |
| **Types available** | Immediately — already generated from schema | Only after manual codegen step |
| **Field discovery** | Method signatures on builder types | AI must know schema from memory |
| **Changing a field** | Add/remove a method call | Edit .graphql file + re-run codegen |
| **Files involved** | One (your Go/TS file) | Two+ (.graphql file + consuming code) |
| **AI iterations** | Instant — change code, it compiles | Blocked by codegen step each time |

GQLKit's builder pattern turns GraphQL queries into regular code that AI assistants can write, modify, and reason about — with no build steps in between.
