package clientgents

import (
	"fmt"
	"os"
	"path/filepath"
)

// TSWriter handles writing generated TypeScript files to disk. Unlike the Go
// writer, it does not apply any formatting since TypeScript formatting is
// typically handled by external tools (prettier, etc.).
type TSWriter struct {
	outputDir string // root directory for all generated TS output
}

// NewTSWriter creates a new TSWriter
func NewTSWriter(outputDir string) *TSWriter {
	return &TSWriter{outputDir: outputDir}
}

// WriteFile writes content to a file in the output directory, automatically
// creating intermediate directories. No formatting is applied.
func (w *TSWriter) WriteFile(filename, content string) error {
	fullPath := filepath.Join(w.outputDir, filename)

	parentDir := filepath.Dir(fullPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", parentDir, err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filename, err)
	}

	return nil
}

// EnsureDir ensures the output directory exists
func (w *TSWriter) EnsureDir() error {
	return os.MkdirAll(w.outputDir, 0755)
}
