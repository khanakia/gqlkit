// Package templater provides a Template wrapper around Go's text/template with
// pre-registered helper functions (pascalCase, camelCase, formatDesc, jsonTag,
// etc.) and embedded template files used by the Go code generator. It follows a
// pattern similar to the ent framework's template management.
package templater

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"text/template/parse"
)

// templateDir embeds all .tmpl files under the template/ subdirectory. These
// are the Go code generation templates used by the clientgen package.
//
//go:embed template/*
var templateDir embed.FS

// var templateFileNames = []string{
// 	"scalars.tmpl",
// 	"types.tmpl",
// 	"enums.tmpl",
// 	"inputs.tmpl",
// 	"client.tmpl",
// 	"builder.tmpl",
// }

// pascalCase converts a string like "CREATED_AT" or "user_id" to PascalCase,
// with special handling for common acronyms (ID, URL, API, etc.).
// Used as a template function.
func pascalCase(s string) string {
	if s == "" {
		return s
	}

	// Handle common acronyms
	acronyms := map[string]string{
		"id":   "ID",
		"url":  "URL",
		"api":  "API",
		"http": "HTTP",
		"json": "JSON",
		"xml":  "XML",
		"sql":  "SQL",
		"html": "HTML",
		"css":  "CSS",
		"uri":  "URI",
		"uuid": "UUID",
	}

	// Split by underscore, hyphen, or space
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})

	var result strings.Builder
	for _, word := range words {
		lower := strings.ToLower(word)
		if acronym, ok := acronyms[lower]; ok {
			result.WriteString(acronym)
		} else {
			// Capitalize first letter, lowercase the rest
			if len(word) > 0 {
				result.WriteString(strings.ToUpper(string(word[0])))
				if len(word) > 1 {
					result.WriteString(strings.ToLower(word[1:]))
				}
			}
		}
	}

	return result.String()
}

// formatDesc formats a description for Go comments
// First line includes the name prefix, subsequent lines are continuation
func formatDesc(name, desc string) string {
	if desc == "" {
		return ""
	}

	lines := strings.Split(desc, "\n")
	var result strings.Builder

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if i == 0 {
			result.WriteString(fmt.Sprintf("// %s %s\n", name, line))
		} else {
			result.WriteString(fmt.Sprintf("// %s\n", line))
		}
	}

	return result.String()
}

// splitLines splits a string by newlines
func splitLines(s string) []string {
	return strings.Split(s, "\n")
}

// trimSpace trims whitespace from a string
func trimSpace(s string) string {
	return strings.TrimSpace(s)
}

// toCamelCase converts a string to lowerCamelCase. It first applies pascalCase
// then lowercases the leading character(s), preserving acronym boundaries.
func toCamelCase(s string) string {
	pascal := pascalCase(s)
	if pascal == "" {
		return pascal
	}

	// Find the first lowercase letter position
	for i, r := range pascal {
		if r >= 'a' && r <= 'z' {
			if i == 0 {
				return pascal
			}
			if i == 1 {
				return strings.ToLower(string(pascal[0])) + pascal[1:]
			}
			// Handle acronyms like "ID" -> "id", "URL" -> "url"
			return strings.ToLower(pascal[:i-1]) + pascal[i-1:]
		}
	}

	// All uppercase (acronym)
	return strings.ToLower(pascal)
}

// jsonTag generates a Go struct field JSON tag from a GraphQL field name.
// The field name is converted to camelCase; omitempty is appended when true.
func jsonTag(fieldName string, omitempty bool) string {
	jsonName := toCamelCase(fieldName)
	if omitempty {
		return fmt.Sprintf("`json:\"%s,omitempty\"`", jsonName)
	}
	return fmt.Sprintf("`json:\"%s\"`", jsonName)
}

// Funcs is the default set of template helper functions registered on every
// Template created via NewTemplate. These are available inside .tmpl files.
var Funcs = template.FuncMap{
	"lower":      strings.ToLower,
	"upper":      strings.ToUpper,
	"base":       filepath.Base,
	"pascalCase": pascalCase,
	"camelCase":  toCamelCase,
	"formatDesc": formatDesc,
	"splitLines": splitLines,
	"trimSpace":  trimSpace,
	"jsonTag":    jsonTag,
}

// Template wraps Go's standard template.Template to provide chainable methods
// and an accumulated FuncMap. It mirrors the pattern used by ent's codegen.
type Template struct {
	*template.Template
	FuncMap template.FuncMap // accumulated function map across Funcs() calls
}

// NewTemplate creates an empty template with the standard codegen functions.
func NewTemplate(name string) *Template {
	t := &Template{Template: template.New(name)}
	return t.Funcs(Funcs)
}

// Funcs merges the given funcMap into the template. New functions are added;
// existing functions are not overwritten in the accumulated FuncMap.
func (t *Template) Funcs(funcMap template.FuncMap) *Template {
	t.Template.Funcs(funcMap)
	if t.FuncMap == nil {
		t.FuncMap = template.FuncMap{}
	}
	for name, f := range funcMap {
		if _, ok := t.FuncMap[name]; !ok {
			t.FuncMap[name] = f
		}
	}
	return t
}

// SkipIf allows registering a function to determine if the template needs to be skipped or not.
// func (t *Template) SkipIf(cond func(*Graph) bool) *Template {
// 	t.condition = cond
// 	return t
// }

// Parse parses text as a template body for t.
func (t *Template) Parse(text string) (*Template, error) {
	if _, err := t.Template.Parse(text); err != nil {
		return nil, err
	}
	return t, nil
}

// ParseFiles parses a list of files as templates and associate them with t.
// Each file can be a standalone template.
func (t *Template) ParseFiles(filenames ...string) (*Template, error) {
	if _, err := t.Template.ParseFiles(filenames...); err != nil {
		return nil, err
	}
	return t, nil
}

// ParseGlob parses the files that match the given pattern as templates and
// associate them with t.
func (t *Template) ParseGlob(pattern string) (*Template, error) {
	if _, err := t.Template.ParseGlob(pattern); err != nil {
		return nil, err
	}
	return t, nil
}

// ParseDir walks on the given dir path and parses the given matches with aren't Go files.
func (t *Template) ParseDir(path string) (*Template, error) {
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("walk path %s: %w", path, err)
		}
		if info.IsDir() || strings.HasSuffix(path, ".go") {
			return nil
		}
		_, err = t.ParseFiles(path)
		return err
	})
	return t, err
}

// ParseFS is like ParseFiles or ParseGlob but reads from the file system fsys
// instead of the host operating system's file system.
func (t *Template) ParseFS(fsys fs.FS, patterns ...string) (*Template, error) {
	if _, err := t.Template.ParseFS(fsys, patterns...); err != nil {
		return nil, err
	}
	return t, nil
}

// AddParseTree adds the given parse tree to the template.
func (t *Template) AddParseTree(name string, tree *parse.Tree) (*Template, error) {
	if _, err := t.Template.AddParseTree(name, tree); err != nil {
		return nil, err
	}
	return t, nil
}

// MustParse is a helper that wraps a call to a function returning (*Template, error)
// and panics if the error is non-nil.
func MustParse(t *Template, err error) *Template {
	if err != nil {
		panic(err)
	}
	return t
}

// type Builder struct {
// 	config       *Config
// 	clientConfig *ClientConfig
// 	schema       *ast.Schema
// 	writer       *Writer
// }

// func NewBuilder(config *Config, clientConfig *ClientConfig, schema *ast.Schema) *Builder {
// 	return &Builder{config: config, clientConfig: clientConfig, schema: schema, writer: writer}
// }

// TemplateDir returns the embedded filesystem containing the Go code generation
// templates. Callers typically pass this to Template.ParseFS.
func TemplateDir() fs.FS {
	return templateDir
}
