package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/khanakia/gqlkit/gqlkit-sdl/schema"

	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:   "gqlkit-sdl",
		Short: "Fetch a GraphQL schema and convert it to SDL format",
	}

	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(fetchCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version and exit",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("gqlkit-sdl", version)
		},
	}
}

func fetchCmd() *cobra.Command {
	var (
		url     string
		output  string
		headers []string
		debug   bool
	)

	cmd := &cobra.Command{
		Use:   "fetch",
		Short: "Fetch GraphQL schema and save as SDL",
		Long: `Fetch a GraphQL schema via introspection and save it as SDL.

Examples:
  gqlkit-sdl fetch --url https://graphql.anilist.co
  gqlkit-sdl fetch --url https://graphql.anilist.co --output my-schema.graphql
  gqlkit-sdl fetch --url https://graphql.anilist.co -H "Authorization: Bearer token"
  gqlkit-sdl fetch --url https://graphql.anilist.co -H "Authorization: Bearer token" -H "Origin: https://example.com"
  gqlkit-sdl fetch --url https://graphql.anilist.co --debug`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := &schema.FetchOptions{
				Headers: make(map[string]string),
				Debug:   debug,
			}

			for _, h := range headers {
				key, value, found := strings.Cut(h, ":")
				if !found {
					return fmt.Errorf("invalid header format %q, expected Key:Value", h)
				}
				opts.Headers[strings.TrimSpace(key)] = strings.TrimSpace(value)
			}

			if !debug {
				fmt.Printf("Fetching schema from %s...\n", url)
			}
			introspectionSchema, err := schema.FetchSchema(url, opts)
			if err != nil {
				return fmt.Errorf("fetching schema: %w", err)
			}
			if introspectionSchema == nil {
				return nil
			}

			fmt.Println("Converting to SDL format...")
			sdl := schema.ConvertToSDL(introspectionSchema)

			fmt.Printf("Saving to %s...\n", output)
			if err := schema.SaveToFile(sdl, output); err != nil {
				return fmt.Errorf("saving file: %w", err)
			}

			fmt.Println("Done! Schema saved successfully.")
			return nil
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "GraphQL endpoint URL (required)")
	cmd.Flags().StringVarP(&output, "output", "o", "schema.graphql", "Output file path")
	cmd.Flags().StringArrayVarP(&headers, "header", "H", nil, `HTTP header in "Key:Value" format (repeatable)`)
	cmd.Flags().BoolVar(&debug, "debug", false, "Print the curl command for debugging")
	cmd.MarkFlagRequired("url")

	return cmd
}
