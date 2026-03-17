// Package main provides the CLI entry point for gqlkit-sdl, a tool that fetches
// a GraphQL schema from a live endpoint via the introspection query and converts
// it to SDL (.graphql) format. The pipeline is:
// HTTP POST introspection query -> parse JSON response -> convert to SDL text -> save to file.
package main

import (
	"flag"
	"fmt"
	"os"

	"gqlkit-sdl/schema"
)

// main parses CLI flags, fetches the GraphQL introspection schema from the
// specified endpoint, converts it to SDL format, and writes it to a file.
func main() {
	// Define CLI flags for endpoint URL, output path, and optional HTTP headers.
	url := flag.String("url", "", "GraphQL endpoint URL")
	output := flag.String("output", "schema.graphql", "Output file path")
	authHeader := flag.String("auth", "", "Authorization header value (optional)")
	referer := flag.String("referer", "", "Referer header value (optional)")
	origin := flag.String("origin", "", "Origin header value (optional)")

	flag.Parse()

	// URL is required; print usage and exit if not provided.
	if *url == "" {
		fmt.Println("Usage: gqlsdl -url <graphql-endpoint> [-output <file>] [-auth <token>]")
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Build FetchOptions with any user-supplied HTTP headers.
	opts := &schema.FetchOptions{
		Headers: make(map[string]string),
	}

	if *authHeader != "" {
		opts.Headers["Authorization"] = *authHeader
	}
	if *referer != "" {
		opts.Headers["Referer"] = *referer
	}
	if *origin != "" {
		opts.Headers["Origin"] = *origin
	}

	// Step 1: Fetch the introspection schema from the remote GraphQL endpoint.
	fmt.Printf("Fetching schema from %s...\n", *url)

	introspectionSchema, err := schema.FetchSchema(*url, opts)
	if err != nil {
		fmt.Printf("Error fetching schema: %v\n", err)
		os.Exit(1)
	}

	// Step 2: Convert the parsed introspection JSON into SDL text.
	fmt.Println("Converting to SDL format...")
	sdl := schema.ConvertToSDL(introspectionSchema)

	// Step 3: Write the SDL output to disk.
	fmt.Printf("Saving to %s...\n", *output)
	if err := schema.SaveToFile(sdl, *output); err != nil {
		fmt.Printf("Error saving file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Done! Schema saved successfully.")
}
