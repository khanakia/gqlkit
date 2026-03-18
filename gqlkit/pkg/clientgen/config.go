package clientgen

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/tidwall/jsonc"
)

// Config holds all settings needed to run the Go SDK generator, typically
// populated from CLI flags. SchemaPath and ConfigPath are required.
type Config struct {
	// SchemaPath is the path to the GraphQL SDL file
	SchemaPath string
	// OutputDir is the directory where the generated SDK will be written
	OutputDir string
	// PackageName is the Go package name for the generated SDK
	PackageName string
	// ModulePath is the Go module path for the generated SDK
	ModulePath string

	// Config is the configuration for the generator
	ConfigPath string

	// Package is the Go package name for the generated SDK
	Package string
}

// Validate checks that required fields are set and applies defaults for
// optional fields (OutputDir defaults to "./sdk", PackageName defaults to "sdk").
func (c *Config) Validate() error {
	if c.SchemaPath == "" {
		return ErrSchemaPathRequired
	}
	if c.OutputDir == "" {
		c.OutputDir = "./sdk"
	}
	if c.PackageName == "" {
		c.PackageName = "sdk"
	}
	return nil
}

// loadClientConfig reads and parses a JSONC config file (supports comments)
// into a ClientConfig struct containing scalar-to-Go type bindings.
// If path is empty or the file does not exist, returns an empty config.
func loadClientConfig(path string) (*ClientConfig, error) {
	if path == "" {
		return &ClientConfig{}, nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ClientConfig{}, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	var config ClientConfig
	err = json.Unmarshal(jsonc.ToJSON(content), &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return &config, nil
}
