package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd is the top-level cobra command for the gqlsdk CLI tool. Subcommands
// (like "generate") are registered in init().
var rootCmd = &cobra.Command{
	Use:   "gqlsdk",
	Short: "GraphQL SDK Generator",
	Long:  `A CLI tool that generates type-safe Go client SDKs from GraphQL SDL files.`,
}

// Execute runs the root command. Called from main.main().
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(generateCmd)
}
