package main

import (
	"fmt"
	"github.com/khanakia/gqlkit/gqlkit/pkg/clientgents"

	"github.com/spf13/cobra"
)

// CLI flag variables bound to the generate-ts command's flags.
var (
	tsSchemaPath string
	tsOutputDir  string
	tsConfigPath string
)

// generateTSCmd is the "generate-ts" subcommand that produces a TypeScript
// client SDK from a GraphQL SDL schema file.
var generateTSCmd = &cobra.Command{
	Use:   "generate-ts",
	Short: "Generate TypeScript SDK from GraphQL schema",
	Long:  `Generates a type-safe TypeScript client SDK from a GraphQL SDL file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		config := &clientgents.Config{
			SchemaPath: tsSchemaPath,
			OutputDir:  tsOutputDir,
			ConfigPath: tsConfigPath,
		}

		gen, err := clientgents.New(config)
		if err != nil {
			return fmt.Errorf("failed to create TypeScript generator: %w", err)
		}

		if err := gen.Generate(); err != nil {
			return fmt.Errorf("failed to generate TypeScript SDK: %w", err)
		}

		fmt.Printf("TypeScript SDK generated successfully in %s\n", tsOutputDir)
		return nil
	},
}

func init() {
	generateTSCmd.Flags().StringVarP(&tsSchemaPath, "schema", "s", "", "Path to GraphQL SDL file (required)")
	generateTSCmd.Flags().StringVarP(&tsOutputDir, "output", "o", "./sdk", "Output directory for generated SDK")
	generateTSCmd.Flags().StringVarP(&tsConfigPath, "config", "c", "", "Path to config.jsonc file (optional)")

	generateTSCmd.MarkFlagRequired("schema")
}
