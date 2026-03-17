package main

import (
	"fmt"

	"gqlkit/pkg/clientgents"
)

func main() {
	fmt.Println("Generating TypeScript SDK...")

	config := &clientgents.Config{
		SchemaPath: "cmd/generate/schema.graphql",
		OutputDir:  "./sdk",
		ConfigPath: "cmd/generate/config.jsonc",
	}

	gen, err := clientgents.New(config)
	if err != nil {
		fmt.Printf("failed to create generator: %v\n", err)
		return
	}

	if err := gen.Generate(); err != nil {
		fmt.Printf("failed to generate SDK: %v\n", err)
		return
	}

	fmt.Println("TypeScript SDK generation completed.")
}
