// Package main is the entry point for the mock GraphQL API server.
// It sets up a gqlgen-based HTTP server with an in-memory data store,
// providing a todo/user CRUD API for testing GraphQL client libraries.
// The server exposes a GraphQL playground at "/" and the query endpoint at "/query".
package main

import (
	"mockapi/graph"
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/vektah/gqlparser/v2/ast"
)

// defaultPort is the fallback HTTP port when the PORT env var is not set.
const defaultPort = "8081"

func main() {
	// Allow port override via environment variable for flexible deployment.
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	// Initialize the gqlgen handler with the generated schema and resolver.
	// The Resolver struct holds all in-memory state (todos, users) and seeds
	// data lazily on first query.
	srv := handler.New(graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{}}))

	// Register supported HTTP transports: OPTIONS (CORS preflight), GET, and POST.
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})

	// Cache up to 1000 parsed query documents to avoid re-parsing identical queries.
	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	// Enable schema introspection so clients can discover the API.
	srv.Use(extension.Introspection{})
	// Enable Automatic Persisted Queries (APQ) with a 100-entry hash cache,
	// allowing clients to send a query hash instead of the full query string.
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	// Serve the interactive GraphQL playground UI at the root path.
	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	// Serve the GraphQL query endpoint.
	http.Handle("/query", srv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
