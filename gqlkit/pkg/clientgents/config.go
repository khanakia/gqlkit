package clientgents

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/tidwall/jsonc"
)

// Config holds all settings needed to run the TypeScript SDK generator,
// typically populated from CLI flags.
type Config struct {
	// SchemaPath is the path to the GraphQL SDL file
	SchemaPath string
	// OutputDir is the directory where the generated SDK will be written
	OutputDir string
	// ConfigPath is the path to config.jsonc
	ConfigPath string
}

// ClientConfig is loaded from config.jsonc and contains user-specified
// scalar-to-TypeScript type overrides.
type ClientConfig struct {
	Bindings ConfigTSBindings `json:"bindings"`
}

// Validate checks that required fields are set and applies defaults.
func (c *Config) Validate() error {
	if c.SchemaPath == "" {
		return ErrSchemaPathRequired
	}
	if c.OutputDir == "" {
		c.OutputDir = "./sdk"
	}
	if c.ConfigPath == "" {
		return ErrConfigPathRequired
	}
	return nil
}

// loadClientConfig reads and parses a JSONC config file into a ClientConfig.
func loadClientConfig(path string) (*ClientConfig, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	var config ClientConfig
	err = json.Unmarshal(jsonc.ToJSON(content), &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return &config, nil
}
