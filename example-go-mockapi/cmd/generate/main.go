package main

import (
	"fmt"

	"github.com/khanakia/gqlkit/gqlkit/pkg/clientgen"
)

func main() {
	fmt.Println("Generating SDK for gqlgenapi...")

	config := &clientgen.Config{
		SchemaPath: "cmd/generate/schema.graphql",
		OutputDir:  "./sdk",
		PackageName: "sdk",
		ModulePath:  "github.com/example/sdk",
		ConfigPath:  "cmd/generate/config.jsonc",
		Package:     "example-go-mockapi/sdk",
	}

	gen, err := clientgen.New(config)
	if err != nil {
		fmt.Printf("failed to create generator: %v\n", err)
		return
	}

	if err := gen.Generate(); err != nil {
		fmt.Printf("failed to generate SDK: %v\n", err)
		return
	}

	fmt.Println("SDK generation completed.")
}

