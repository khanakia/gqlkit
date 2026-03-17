// Package schemagql provides utilities for loading and parsing GraphQL SDL
// schema files into the gqlparser AST representation. It supports glob patterns
// for multi-file schemas and handles the edge case where built-in scalar types
// may or may not be present in the source SDL.
package schemagql

import (
	"fmt"
	"gqlkit/pkg/util"
	"os"
	"path"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
	"github.com/vektah/gqlparser/v2/validator"
)

// StringList provides yaml unmarshaler to accept both `string` and `[]string` as a valid type.
// Sourced from [gqlgen].
//
// [gqlgen]: https://github.com/99designs/gqlgen/blob/1a0b19feff6f02d2af6631c9d847bc243f8ede39/codegen/config/config.go#L302-L329
type StringList []string

// GetSchema reads all files matching the given glob patterns, parses them as
// GraphQL SDL, and returns a validated schema. It automatically injects the
// GraphQL prelude (built-in scalars like String, Int, etc.) when the source
// files do not already define them.
func GetSchema(globs StringList) (*ast.Schema, error) {
	filenames, err := expandFilenames(globs)
	if err != nil {
		return nil, err
	}

	sources := make([]*ast.Source, len(filenames))
	for i, filename := range filenames {
		text, err := os.ReadFile(filename)
		if err != nil {
			return nil, util.Errorf(nil, "unreadable schema file %v: %v", filename, err)
		}
		sources[i] = &ast.Source{Name: filename, Input: string(text)}
	}

	// Ideally here we'd just call gqlparser.LoadSchema. But the schema we are
	// given may or may not contain the builtin types String, Int, etc. (The
	// spec says it shouldn't, but introspection will return those types, and
	// some introspection-to-SDL tools aren't smart enough to remove them.) So
	// we inline LoadSchema and insert some checks.
	document, graphqlError := parser.ParseSchemas(sources...)
	if graphqlError != nil {
		// Schema doesn't even parse.
		return nil, util.Errorf(nil, "invalid schema: %v", graphqlError)
	}

	// Check if we have a builtin type. (String is an arbitrary choice.)
	hasBuiltins := false
	for _, def := range document.Definitions {
		if def.Name == "String" {
			hasBuiltins = true
			break
		}
	}

	if !hasBuiltins {
		// modified from parser.ParseSchemas
		var preludeAST *ast.SchemaDocument
		preludeAST, graphqlError = parser.ParseSchema(validator.Prelude)
		if graphqlError != nil {
			return nil, util.Errorf(nil, "invalid prelude (probably a gqlparser bug): %v", graphqlError)
		}
		document.Merge(preludeAST)
	}

	schema, graphqlError := validator.ValidateSchemaDocument(document)
	if graphqlError != nil {
		return nil, util.Errorf(nil, "invalid schema: %v", graphqlError)
	}

	return schema, nil
}

// expandFilenames resolves glob patterns into a deduplicated list of file paths.
// It uses doublestar for extended glob support (e.g. "**/*.graphql").
func expandFilenames(globs []string) ([]string, error) {
	uniqFilenames := make(map[string]bool, len(globs))
	for _, glob := range globs {
		// SplitPattern in case the path is absolute or something; a valid path
		// isn't necessarily a valid glob-pattern.
		glob = filepath.Clean(glob)
		glob = filepath.ToSlash(glob)
		base, pattern := doublestar.SplitPattern(glob)
		matches, err := doublestar.Glob(os.DirFS(base), pattern, doublestar.WithFilesOnly())
		if err != nil {
			return nil, util.Errorf(nil, "can't expand file-glob %v: %v", glob, err)
		}
		if len(matches) == 0 {
			return nil, util.Errorf(nil, "%v did not match any files", glob)
		}
		for _, match := range matches {
			uniqFilenames[path.Join(base, match)] = true
		}
	}
	filenames := make([]string, 0, len(uniqFilenames))
	for filename := range uniqFilenames {
		filenames = append(filenames, filename)
	}
	return filenames, nil
}

// ParseSchemaFile reads and parses a single GraphQL schema file using
// gqlparser.LoadSchema. Unlike GetSchema it does not handle multi-file schemas
// or missing built-in types.
func ParseSchemaFile(filepath string) (*ast.Schema, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	schema, gqlErr := gqlparser.LoadSchema(&ast.Source{
		Name:  filepath,
		Input: string(content),
	})
	if gqlErr != nil {
		return nil, fmt.Errorf("failed to parse schema: %v", gqlErr)
	}

	return schema, nil
}
