// Package cmd implements the CLI commands for the gqlkit code generator.
// It uses cobra for command parsing and delegates to the clientgen package
// for actual SDK generation.
package cmd

import (
	"fmt"
	"gqlkit/pkg/clientgen"

	"github.com/spf13/cobra"
)

// CLI flag variables bound to the generate command's flags.
var (
	schemaPath  string
	outputDir   string
	packageName string
	modulePath  string
)

// generateCmd is the "generate" subcommand that produces a Go client SDK from
// a GraphQL SDL schema file.
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate Go SDK from GraphQL schema",
	Long:  `Generates a type-safe Go client SDK from a GraphQL SDL file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if schemaPath == "" {
			return fmt.Errorf("--schema flag is required")
		}

		config := &clientgen.Config{
			SchemaPath:  schemaPath,
			OutputDir:   outputDir,
			PackageName: packageName,
			ModulePath:  modulePath,
		}

		gen, err := clientgen.New(config)
		if err != nil {
			return fmt.Errorf("failed to create generator: %w", err)
		}

		if err := gen.Generate(); err != nil {
			return fmt.Errorf("failed to generate SDK: %w", err)
		}

		fmt.Printf("SDK generated successfully in %s\n", outputDir)
		return nil
	},
}

func init() {
	generateCmd.Flags().StringVarP(&schemaPath, "schema", "s", "", "Path to GraphQL SDL file (required)")
	generateCmd.Flags().StringVarP(&outputDir, "output", "o", "./sdk", "Output directory for generated SDK")
	generateCmd.Flags().StringVarP(&packageName, "package", "p", "sdk", "Go package name for generated SDK")
	generateCmd.Flags().StringVarP(&modulePath, "module", "m", "", "Go module path for generated SDK (e.g., github.com/user/myapi)")

	generateCmd.MarkFlagRequired("schema")
}
