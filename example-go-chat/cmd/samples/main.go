package main

import (
	"context"
	"encoding/json"
	"errors"
	"example-go-chat/sdk/graphqlclient"
	"fmt"
	"log"

	"example-go-chat/sdk/fields"
	"example-go-chat/sdk/inputs"
	"example-go-chat/sdk/mutations"
	"example-go-chat/sdk/queries"
)

func main() {
	fmt.Println("Hello, World!")
	client := graphqlclient.NewClient("http://localhost:2310/api/sa/query",
		graphqlclient.WithHeaders(map[string]string{
			"x-workspace-id": "ueqempirgtj02vz4pg",
			"token":          "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJhcHBJZCI6ImEyIiwiZW1haWwiOiJraGFuYWtpYUBnbWFpbC5jb20iLCJleHAiOjE3NzIyNTYzNjMsImZpcnN0TmFtZSI6IkFtYW4iLCJpYXQiOjE3Njk0OTE1NjMsImlzcyI6InNhYXNieXRlZSIsInN1YiI6InU2aDdjcW16bHpzOG9mY2I1dSIsInVzZXJOYW1lIjoiQmFuc2FsIn0.PuVOdpQRQqMqDu5nOQbD9KzVjrxSuNMjN-q1JNHiy2oD15BPEGafjwGFH0v1PO7e0ob1vzXmuxuFo1ss43_OMYtkNLQvE9us4BvjwYjbOVwE6_UqLaxjcPvx4rJZjdgfjaxH2QEerNFxGZgFEFi-0LmZ6WwAKCZdkpuGwreEZrL6jX7FLLUxeyspH9V_Stz3LVqbYoryX58tlOmZfxjeYCauwkOA66InupjP3l6EQm7EMuS84N7qU-w1eCYzEesgfvoaxt4s3_MNxIedkllC8ChuGgJLKNiEsplGrGicXMTHOhihpLTEA2eQbhTijFZ6JyaFEYUOOuY1XUazwQOxsQ",
		}),
	)

	ctx := context.Background()

	// Create query and mutation roots
	qr := queries.NewQueryRoot(client)
	// mr := mutations.NewMutationRoot(client)

	if err := runPing(ctx, qr); err != nil {
		debugPrintError(err)
		log.Fatal(err)
	}
	if err := runChatbots(ctx, qr); err != nil {
		log.Fatal(err)
	}
	// if err := runCredentials(ctx, qr); err != nil {
	// 	log.Fatal(err)
	// }
	// if err := runUsers(ctx, qr); err != nil {
	// 	log.Fatal(err)
	// }
	// if err := runFolders(ctx, qr); err != nil {
	// 	log.Fatal(err)
	// }
	// if err := runChatbotsNested(ctx, qr); err != nil {
	// 	log.Fatal(err)
	// }
	// if err := runFoldersNested(ctx, qr); err != nil {
	// 	log.Fatal(err)
	// }
	// if err := runChannelsNested(ctx, qr); err != nil {
	// 	log.Fatal(err)
	// }
	// if err := runHTMLToMarkdown(ctx, mr); err != nil {
	// 	log.Fatal(err)
	// }
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

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}

func runPing(ctx context.Context, qr *queries.QueryRoot) error {
	pingResult, err := qr.Ping().Execute(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Ping result: %v\n", pingResult)

	// full data JSON
	rawData, err := qr.Ping().ExecuteRaw(ctx)
	if err != nil {
		return err
	}
	b, _ := json.MarshalIndent(rawData, "", "  ")
	fmt.Printf("Ping raw data:\n%s\n", b)

	return nil
}

func runChatbots(ctx context.Context, qr *queries.QueryRoot) error {
	chatbotsResult, err := qr.Chatbots().
		First(intPtr(10)).
		Select(func(conn *fields.ChatbotConnectionFields) {
			conn.TotalCount()
			conn.Edges(func(e *fields.ChatbotEdgeFields) {
				e.Cursor()
				e.Node(func(c *fields.ChatbotFields) {
					c.ID().
						Name().
						CreatedAt().
						UpdatedAt()
				})
			})
			conn.PageInfo(func(p *fields.PageInfoFields) {
				p.HasNextPage().
					HasPreviousPage().
					StartCursor().
					EndCursor()
			})
		}).
		Execute(ctx)
	if err != nil {
		return err
	}

	// goutil.PrintToJSON(chatbotsResult)

	fmt.Printf("Chatbots: %+v\n", chatbotsResult)
	return nil
}

func runCredentials(ctx context.Context, qr *queries.QueryRoot) error {
	credentialsResult, err := qr.Credentials().
		First(intPtr(5)).
		Select(func(conn *fields.CredentialConnectionFields) {
			conn.TotalCount()
			conn.Edges(func(e *fields.CredentialEdgeFields) {
				e.Cursor()
				e.Node(func(c *fields.CredentialFields) {
					c.ID().
						Name().
						CreatedAt()
				})
			})
		}).
		Execute(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Credentials: %+v\n", credentialsResult)
	return nil
}

func runUsers(ctx context.Context, qr *queries.QueryRoot) error {
	usersResult, err := qr.Users().
		First(intPtr(10)).
		Select(func(conn *fields.UserConnectionFields) {
			conn.TotalCount()
			conn.Edges(func(e *fields.UserEdgeFields) {
				e.Cursor()
				e.Node(func(u *fields.UserFields) {
					u.ID().
						Email().
						FirstName().
						LastName().
						CreatedAt()
				})
			})
		}).
		Execute(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Users: %+v\n", usersResult)
	return nil
}

func runFolders(ctx context.Context, qr *queries.QueryRoot) error {
	foldersResult, err := qr.Folders().
		First(intPtr(20)).
		Select(func(conn *fields.FolderConnectionFields) {
			conn.TotalCount()
			conn.Edges(func(e *fields.FolderEdgeFields) {
				e.Cursor()
				e.Node(func(f *fields.FolderFields) {
					f.ID().
						Name().
						Position().
						CreatedAt()
				})
			})
		}).
		Execute(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Folders: %+v\n", foldersResult)
	return nil
}

func runChatbotsNested(ctx context.Context, qr *queries.QueryRoot) error {
	chatbotsNestedResult, err := qr.Chatbots().
		First(intPtr(5)).
		Select(func(conn *fields.ChatbotConnectionFields) {
			conn.TotalCount()
			conn.Edges(func(e *fields.ChatbotEdgeFields) {
				e.Node(func(c *fields.ChatbotFields) {
					c.ID().
						Name().
						CreatedAt()
					c.AiModel(func(ai *fields.AiModelFields) {
						ai.ID().
							ModelID().
							ProviderName().
							Status()
					})
					c.Users(func(u *fields.UserFields) {
						u.ID().
							Email().
							FirstName().
							LastName()
					})
				})
			})
		}).
		Execute(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Chatbots with nested: %+v\n", chatbotsNestedResult)
	return nil
}

func runFoldersNested(ctx context.Context, qr *queries.QueryRoot) error {
	foldersNestedResult, err := qr.Folders().
		First(intPtr(10)).
		Select(func(conn *fields.FolderConnectionFields) {
			conn.TotalCount()
			conn.Edges(func(e *fields.FolderEdgeFields) {
				e.Node(func(f *fields.FolderFields) {
					f.ID().
						Name().
						Position()
					f.Owner(func(u *fields.UserFields) {
						u.ID().
							Email().
							FirstName()
					})
					f.Chatbot(func(c *fields.ChatbotFields) {
						c.ID().
							Name()
					})
					f.Channels(func(ch *fields.ChannelFields) {
						ch.ID().
							Name().
							CreatedAt()
					})
				})
			})
		}).
		Execute(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Folders with nested: %+v\n", foldersNestedResult)
	return nil
}

func runChannelsNested(ctx context.Context, qr *queries.QueryRoot) error {
	channelsResult, err := qr.Channels().
		First(intPtr(5)).
		Select(func(conn *fields.ChannelConnectionFields) {
			conn.TotalCount()
			conn.Edges(func(e *fields.ChannelEdgeFields) {
				e.Node(func(ch *fields.ChannelFields) {
					ch.ID().
						Name().
						CreatedAt()
					ch.Owner(func(u *fields.UserFields) {
						u.ID().
							Email()
					})
					ch.Chatbot(func(c *fields.ChatbotFields) {
						c.ID().
							Name()
						c.AiModel(func(ai *fields.AiModelFields) {
							ai.ID().
								ModelID().
								ProviderName()
						})
					})
					ch.Folder(func(f *fields.FolderFields) {
						f.ID().
							Name().
							Position()
					})
				})
			})
		}).
		Execute(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Channels with deep nesting: %+v\n", channelsResult)
	return nil
}

func runHTMLToMarkdown(ctx context.Context, mr *mutations.MutationRoot) error {
	markdownResult, err := mr.HTMLToMarkdown().
		Input(inputs.HtmlToMarkdownInput{
			HTML: "<h1>Hello World</h1><p>This is a test</p>",
		}).
		Execute(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("Markdown: %+v\n", markdownResult)
	return nil
}
