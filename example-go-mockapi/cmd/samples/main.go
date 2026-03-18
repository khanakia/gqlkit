package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"example-go-mockapi/sdk/graphqlclient"

	sdkbuilder "example-go-mockapi/sdk/builder"
	"example-go-mockapi/sdk/fields"
	"example-go-mockapi/sdk/inputs"
	"example-go-mockapi/sdk/mutations"
	"example-go-mockapi/sdk/queries"
)

func main() {
	fmt.Println("Running gqlgenapi SDK sample...")

	client := graphqlclient.NewClient(
		"http://localhost:8081/query",
	)

	ctx := context.Background()
	qr := queries.NewQueryRoot(client)
	mr := mutations.NewMutationRoot(client)

	if err := runPing(ctx, qr); err != nil {
		debugPrintError(err)
		log.Fatal(err)
	}

	if err := runEchoAndSum(ctx, qr); err != nil {
		debugPrintError(err)
		log.Fatal(err)
	}

	if err := runUsers(ctx, qr); err != nil {
		debugPrintError(err)
		log.Fatal(err)
	}

	if err := runTodosList(ctx, qr); err != nil {
		debugPrintError(err)
		log.Fatal(err)
	}

	if err := runTodosConnection(ctx, qr); err != nil {
		debugPrintError(err)
		log.Fatal(err)
	}

	if err := runSearch(ctx, qr); err != nil {
		debugPrintError(err)
		log.Fatal(err)
	}

	if err := runServerInfo(ctx, qr); err != nil {
		debugPrintError(err)
		log.Fatal(err)
	}

	if err := runTodoWithScalars(ctx, qr); err != nil {
		debugPrintError(err)
		log.Fatal(err)
	}

	if err := runTodoMutations(ctx, qr, mr); err != nil {
		debugPrintError(err)
		log.Fatal(err)
	}
}

func runPing(ctx context.Context, qr *queries.QueryRoot) error {
	pingResult, err := qr.Ping().Execute(ctx)
	if err != nil {
		return err
	}

	fmt.Printf("Ping result: %v\n", pingResult)

	// Also show raw JSON data
	rawData, err := qr.Ping().ExecuteRaw(ctx)
	if err != nil {
		return err
	}
	b, _ := json.MarshalIndent(rawData, "", "  ")
	fmt.Printf("Ping raw data:\n%s\n", b)

	return nil
}

func runEchoAndSum(ctx context.Context, qr *queries.QueryRoot) error {
	fmt.Println("== Echo and Sum ==")

	echoResult, err := qr.Echo().Message("Hello from SDK").Execute(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Echo result: %s\n", echoResult)

	sumResult, err := qr.Sum().A(40).B(2).Execute(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Sum result (40 + 2): %d\n", sumResult)

	return nil
}

func runUsers(ctx context.Context, qr *queries.QueryRoot) error {
	fmt.Println("== Users ==")

	users, err := qr.Users().
		Select(func(u *fields.UserFields) {
			u.ID().Name().Email().Role()
		}).
		Execute(ctx)
	if err != nil {
		return err
	}

	b, _ := json.MarshalIndent(users, "", "  ")
	fmt.Printf("Users:\n%s\n", b)
	return nil
}

func runTodosList(ctx context.Context, qr *queries.QueryRoot) error {
	fmt.Println("== Todos (list with filter + pagination) ==")

	filter := &inputs.TodoFilter{
		TextContains: stringPtr("todo"),
	}
	pagination := &inputs.PaginationInput{
		Limit:  10,
		Offset: 0,
	}

	todos, err := qr.Todos().
		Filter(filter).
		Pagination(pagination).
		Select(func(f *fields.TodoFields) {
			f.ID().
				Text().
				Done().
				Priority().
				Tags().
				User(func(u *fields.UserFields) {
					u.ID().Name().Role()
				})
		}).
		Execute(ctx)
	if err != nil {
		return err
	}

	b, _ := json.MarshalIndent(todos, "", "  ")
	fmt.Printf("Todos list:\n%s\n", b)
	return nil
}

func runTodosConnection(ctx context.Context, qr *queries.QueryRoot) error {
	fmt.Println("== Todos connection ==")

	filter := &inputs.TodoFilter{
		Done: boolPtr(false),
	}
	pagination := &inputs.PaginationInput{
		Limit:  5,
		Offset: 0,
	}

	conn, err := qr.TodosConnection().
		Filter(filter).
		Pagination(pagination).
		Select(func(c *fields.TodoConnectionFields) {
			c.TotalCount().
				PageInfo(func(p *fields.PageInfoFields) {
					p.HasNextPage().
						HasPreviousPage().
						StartCursor().
						EndCursor()
				}).
				Edges(func(e *fields.TodoEdgeFields) {
					e.Cursor().Node(func(t *fields.TodoFields) {
						t.ID().Text().Done()
					})
				})
		}).
		Execute(ctx)
	if err != nil {
		return err
	}

	b, _ := json.MarshalIndent(conn, "", "  ")
	fmt.Printf("Todos connection:\n%s\n", b)
	return nil
}

func runSearch(ctx context.Context, qr *queries.QueryRoot) error {
	fmt.Println("== Search (union result) ==")

	builder := qr.Search().
		Term("a")

	// Use the generic builder API to select subfields on the union.
	selection := builder.GetSelection()

	// We don't have a dedicated typed selector for the union, so we
	// use the raw FieldSelection to request common fields on both types.
	todoFrag := sdkbuilder.NewFieldSelection()
	todoFrag.AddField("id")
	todoFrag.AddField("text")
	selection.AddChild("... on Todo", todoFrag)

	userFrag := sdkbuilder.NewFieldSelection()
	userFrag.AddField("id")
	userFrag.AddField("name")
	userFrag.AddField("role")
	selection.AddChild("... on User", userFrag)

	results, err := builder.Execute(ctx)
	if err != nil {
		return err
	}

	b, _ := json.MarshalIndent(results, "", "  ")
	fmt.Printf("Search results:\n%s\n", b)
	return nil
}

func runServerInfo(ctx context.Context, qr *queries.QueryRoot) error {
	fmt.Println("== ServerInfo (JSON scalar) ==")

	info, err := qr.ServerInfo().Execute(ctx)
	if err != nil {
		return err
	}

	b, _ := json.MarshalIndent(json.RawMessage(info), "", "  ")
	fmt.Printf("Server info:\n%s\n", b)
	return nil
}

func runTodoWithScalars(ctx context.Context, qr *queries.QueryRoot) error {
	fmt.Println("== Todo with custom scalars (DateTime, Metadata) ==")

	todo, err := qr.Todo().
		ID("1").
		Select(func(f *fields.TodoFields) {
			f.ID().Text().CreatedAt().Metadata()
		}).
		Execute(ctx)
	if err != nil {
		return err
	}

	b, _ := json.MarshalIndent(todo, "", "  ")
	fmt.Printf("Todo with scalars:\n%s\n", b)
	return nil
}

func runTodoMutations(ctx context.Context, qr *queries.QueryRoot, mr *mutations.MutationRoot) error {
	fmt.Println("== Todo mutations (create / update / delete / completeAll) ==")

	// Create
	newInput := inputs.NewTodo{
		Text:   "SDK-created todo",
		UserID: "1",
		Tags:   []string{"sdk", "created"},
	}

	created, err := mr.CreateTodo().
		Input(newInput).
		Select(func(f *fields.TodoFields) {
			f.ID().Text().Done().Tags()
		}).
		Execute(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Created todo: %+v\n", created)

	// Update
	updateInput := inputs.UpdateTodoInput{
		Text: stringPtr("Updated via SDK"),
		Done: boolPtr(true),
	}

	updated, err := mr.UpdateTodo().
		ID(created.ID).
		Input(updateInput).
		Select(func(f *fields.TodoFields) {
			f.ID().Text().Done().Tags()
		}).
		Execute(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Updated todo: %+v\n", updated)

	// Delete
	deleted, err := mr.DeleteTodo().
		ID(created.ID).
		Execute(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Deleted todo id=%s -> %v\n", created.ID, deleted)

	// Complete all
	completedCount, err := mr.CompleteAllTodos().Execute(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Completed %d todos via completeAllTodos\n", completedCount)

	return nil
}

func debugPrintError(err error) {
	var gqlErrs graphqlclient.GraphQLErrors
	if errors.As(err, &gqlErrs) {
		data, mErr := json.MarshalIndent(gqlErrs, "", "  ")
		if mErr != nil {
			log.Printf("GraphQL error (marshal failed: %v): %v", mErr, err)
			return
		}
		log.Printf("GraphQL errors (raw):\n%s", string(data))
		return
	}

	log.Printf("error: %+v", err)
}

// small helpers

func stringPtr(s string) *string { return &s }

func boolPtr(b bool) *bool { return &b }

