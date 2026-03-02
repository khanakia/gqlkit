package util

import (
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/ubgo/goutil"
	"github.com/vektah/gqlparser/v2/ast"
)

func Errorf(pos *ast.Position, msg string, args ...interface{}) error {
	return fmt.Errorf(msg, args...)
}

func DumpStructToFile(v interface{}, filename string) error {
	content, err := goutil.ToJSONIndent(v)
	if err != nil {
		return fmt.Errorf("failed to dump struct to file: %w", err)
	}
	return SaveToFile(filename, content)
}

func SaveToFile(filename string, content string) error {
	return os.WriteFile(filename, []byte(content), 0644)
}

// acronyms holds known initialisms that should stay fully upper-cased,
// following the logic used by ent's codegen helpers.
var acronyms = map[string]struct{}{
	"ACL": {}, "API": {}, "ASCII": {}, "AWS": {}, "CPU": {}, "CSS": {}, "DNS": {}, "EOF": {},
	"GB": {}, "GUID": {}, "HCL": {}, "HTML": {}, "HTTP": {}, "HTTPS": {}, "ID": {}, "IP": {},
	"JSON": {}, "KB": {}, "LHS": {}, "MAC": {}, "MB": {}, "QPS": {}, "RAM": {}, "RHS": {},
	"RPC": {}, "SLA": {}, "SMTP": {}, "SQL": {}, "SSH": {}, "SSO": {}, "TCP": {}, "TLS": {},
	"TTL": {}, "UDP": {}, "UI": {}, "UID": {}, "URI": {}, "URL": {}, "UTF8": {}, "UUID": {},
	"VM": {}, "XML": {}, "XMPP": {}, "XSRF": {}, "XSS": {},
}

// isSeparator matches ent's separator logic (underscore, dash, or whitespace).
func isSeparator(r rune) bool {
	return r == '_' || r == '-' || unicode.IsSpace(r)
}

// splitCamel inserts separators between lower-to-upper transitions so that
// "createdAt" becomes "created_At" before applying Pascal case rules.
func splitCamel(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	var prev rune
	for i, r := range s {
		if i > 0 && unicode.IsLower(prev) && unicode.IsUpper(r) {
			b.WriteRune('_')
		}
		b.WriteRune(r)
		prev = r
	}
	return b.String()
}

// pascalWords upper-cases each word, keeping known acronyms fully upper-case.
// This mirrors the behavior of ent's pascalWords helper.
func pascalWords(words []string) string {
	for i, w := range words {
		upper := strings.ToUpper(w)
		if _, ok := acronyms[upper]; ok {
			words[i] = upper
		} else {
			if w == "" {
				continue
			}
			lower := strings.ToLower(w)
			runes := []rune(lower)
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		}
	}
	return strings.Join(words, "")
}

// toPascalCase converts a string to PascalCase (for field names), borrowing
// the logic from ent's pascal helper and extending it to also handle camelCase.
//
// Examples:
//
//	"created_at"  => "CreatedAt"
//	"createdAt"   => "CreatedAt"
//	"user_id"     => "UserID"
//	"http_server" => "HTTPServer"
func ToPascalCase(s string) string {
	if s == "" {
		return s
	}

	// First normalize camelCase to snake-style with underscores, then split.
	s = splitCamel(s)
	words := strings.FieldsFunc(s, isSeparator)
	return pascalWords(words)
}

// toCamelCase converts a string to lower camelCase, matching ent's camel logic.
//
// Examples:
//
//	"user_name"  => "userName"
//	"UserName"   => "userName"
//	"HTTPServer" => "httpServer"
func ToCamelCase(s string) string {
	if s == "" {
		return s
	}
	// Reuse pascalization on separators, then just lower-case the first rune.
	p := ToPascalCase(s)
	runes := []rune(p)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

// toSnakeCase converts a string to snake_case, following ent's snake helper.
//
// Examples:
//
//	"UserName"   => "user_name"
//	"HTTPServer" => "http_server"
//	"userID"     => "user_id"
func ToSnakeCase(s string) string {
	if s == "" {
		return s
	}
	var (
		j int
		b strings.Builder
	)
	for i := 0; i < len(s); i++ {
		r := rune(s[i])
		// Insert '_' at word boundaries:
		// - not at the start or end
		// - current is uppercase
		// - previous is lowercase (UserInfo)
		//   or next is lowercase and previous is a letter (HTTPServer).
		if i > 0 && i < len(s)-1 && unicode.IsUpper(r) {
			if unicode.IsLower(rune(s[i-1])) ||
				j != i-1 && unicode.IsLower(rune(s[i+1])) && unicode.IsLetter(rune(s[i-1])) {
				j = i
				b.WriteString("_")
			}
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}
