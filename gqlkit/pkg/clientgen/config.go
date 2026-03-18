package clientgen

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
// When Package is empty, it is auto-detected from go.mod + OutputDir.
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
	if c.Package == "" {
		c.Package = detectPackagePath(c.OutputDir)
	}
	return nil
}

// detectPackagePath reads the module path from go.mod in the current directory
// and appends the output directory name to form the full import path for the
// generated SDK (e.g., "github.com/user/myproject/sdk").
func detectPackagePath(outputDir string) string {
	mod := readModulePath("go.mod")
	if mod == "" {
		return ""
	}
	// Clean the output dir: "./sdk" -> "sdk", "foo/bar" -> "foo/bar"
	rel := strings.TrimPrefix(filepath.ToSlash(outputDir), "./")
	return mod + "/" + rel
}

// readModulePath reads the "module" line from a go.mod file.
func readModulePath(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module"))
		}
	}
	return ""
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
