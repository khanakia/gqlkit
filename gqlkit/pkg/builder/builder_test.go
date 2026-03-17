package builder

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- FieldSelection tests ---

func TestNewFieldSelection(t *testing.T) {
	fs := NewFieldSelection()
	assert.NotNil(t, fs)
	assert.Empty(t, fs.Build(0))
}

func TestFieldSelection_AddField(t *testing.T) {
	fs := NewFieldSelection()
	fs.AddField("id")
	fs.AddField("name")
	out := fs.Build(0)
	assert.Contains(t, out, "id")
	assert.Contains(t, out, "name")
}

func TestFieldSelection_AddField_EmptyIndent(t *testing.T) {
	fs := NewFieldSelection()
	fs.AddField("id")
	assert.Equal(t, "id\n", fs.Build(0))
}

func TestFieldSelection_AddField_WithIndent(t *testing.T) {
	fs := NewFieldSelection()
	fs.AddField("id")
	fs.AddField("name")
	out := fs.Build(1)
	assert.Contains(t, out, "  id")
	assert.Contains(t, out, "  name")
}

func TestFieldSelection_AddChild(t *testing.T) {
	fs := NewFieldSelection()
	fs.AddField("id")
	child := NewFieldSelection()
	child.AddField("name")
	child.AddField("slug")
	fs.AddChild("owner", child)
	out := fs.Build(0)
	assert.Contains(t, out, "id")
	assert.Contains(t, out, "owner {")
	assert.Contains(t, out, "name")
	assert.Contains(t, out, "slug")
	assert.Contains(t, out, "}")
}

func TestFieldSelection_AddChild_EmptyChild(t *testing.T) {
	fs := NewFieldSelection()
	fs.AddField("id")
	fs.AddChild("empty", NewFieldSelection())
	out := fs.Build(0)
	// Empty child should not emit a block
	assert.Contains(t, out, "id")
	assert.NotContains(t, out, "empty")
}

func TestFieldSelection_Build_FieldOrder(t *testing.T) {
	fs := NewFieldSelection()
	fs.AddField("z")
	fs.AddField("a")
	fs.AddField("m")
	out := fs.Build(0)
	// Scalar fields are emitted in insertion order (builder does not sort them)
	assert.Contains(t, out, "z")
	assert.Contains(t, out, "a")
	assert.Contains(t, out, "m")
	assert.Len(t, strings.Split(strings.TrimSpace(out), "\n"), 3)
}

func TestFieldSelection_Build_NestedDeterministic(t *testing.T) {
	fs := NewFieldSelection()
	c1 := NewFieldSelection()
	c1.AddField("id")
	c2 := NewFieldSelection()
	c2.AddField("id")
	fs.AddChild("zebra", c1)
	fs.AddChild("alpha", c2)
	out := fs.Build(0)
	// Children keys sorted: alpha before zebra
	alphaPos := strings.Index(out, "alpha")
	zebraPos := strings.Index(out, "zebra")
	assert.Less(t, alphaPos, zebraPos)
}

func TestFieldSelection_Build_DeepNesting(t *testing.T) {
	fs := NewFieldSelection()
	fs.AddField("id")
	level1 := NewFieldSelection()
	level1.AddField("name")
	level2 := NewFieldSelection()
	level2.AddField("code")
	level1.AddChild("country", level2)
	fs.AddChild("address", level1)
	out := fs.Build(1)
	assert.Contains(t, out, "address {")
	assert.Contains(t, out, "country {")
	assert.Contains(t, out, "code")
	assert.Contains(t, out, "name")
}

// --- BaseBuilder tests ---

func TestNewBaseBuilder(t *testing.T) {
	b := NewBaseBuilder(nil, "query", "aiModels", "aiModels")
	assert.NotNil(t, b)
	assert.Equal(t, "query", b.opType)
	assert.Equal(t, "aiModels", b.opName)
	assert.Equal(t, "aiModels", b.fieldName)
	assert.NotNil(t, b.GetSelection())
	assert.Empty(t, b.GetVariables())
}

func TestBaseBuilder_SetArg(t *testing.T) {
	b := NewBaseBuilder(nil, "query", "Q", "items")
	b.SetArg("first", 10, "Int")
	b.SetArg("filter", "active", "String")
	v := b.GetVariables()
	assert.Equal(t, 10, v["first"])
	assert.Equal(t, "active", v["filter"])
}

func TestBaseBuilder_GetSelection(t *testing.T) {
	b := NewBaseBuilder(nil, "query", "Q", "node")
	sel := b.GetSelection()
	assert.NotNil(t, sel)
	sel.AddField("id")
	assert.Contains(t, sel.Build(0), "id")
}

func TestBaseBuilder_BuildQuery_NoArgsNoSelection(t *testing.T) {
	b := NewBaseBuilder(nil, "query", "GetUser", "user")
	q := b.BuildQuery()
	assert.Contains(t, q, "query GetUser")
	assert.Contains(t, q, "user")
	assert.NotContains(t, q, "(")
	assert.NotContains(t, q, "$")
}

func TestBaseBuilder_BuildQuery_WithArgs(t *testing.T) {
	b := NewBaseBuilder(nil, "query", "GetUser", "user")
	b.SetArg("id", "x-123", "ID!")
	q := b.BuildQuery()
	assert.Contains(t, q, "query GetUser($id: ID!)")
	assert.Contains(t, q, "user(id: $id)")
	assert.NotContains(t, q, "user\n}")
}

func TestBaseBuilder_BuildQuery_WithSelection(t *testing.T) {
	b := NewBaseBuilder(nil, "query", "GetUser", "user")
	b.GetSelection().AddField("id")
	b.GetSelection().AddField("name")
	q := b.BuildQuery()
	assert.Contains(t, q, "user {")
	assert.Contains(t, q, "id")
	assert.Contains(t, q, "name")
	assert.Contains(t, q, "}")
}

func TestBaseBuilder_BuildQuery_WithArgsAndSelection(t *testing.T) {
	b := NewBaseBuilder(nil, "query", "GetUser", "user")
	b.SetArg("id", "x-123", "ID!")
	b.GetSelection().AddField("id")
	b.GetSelection().AddField("email")
	q := b.BuildQuery()
	assert.Contains(t, q, "$id: ID!")
	assert.Contains(t, q, "user(id: $id)")
	assert.Contains(t, q, "id")
	assert.Contains(t, q, "email")
}

func TestBaseBuilder_BuildQuery_WithNestedSelection(t *testing.T) {
	b := NewBaseBuilder(nil, "query", "GetUser", "user")
	b.GetSelection().AddField("id")
	addr := NewFieldSelection()
	addr.AddField("city")
	addr.AddField("zip")
	b.GetSelection().AddChild("address", addr)
	q := b.BuildQuery()
	assert.Contains(t, q, "address {")
	assert.Contains(t, q, "city")
	assert.Contains(t, q, "zip")
}

func TestBaseBuilder_BuildQuery_Mutation(t *testing.T) {
	b := NewBaseBuilder(nil, "mutation", "CreateUser", "createUser")
	b.SetArg("input", nil, "CreateUserInput!")
	b.GetSelection().AddField("id")
	b.GetSelection().AddField("email")
	q := b.BuildQuery()
	assert.Contains(t, q, "mutation CreateUser")
	assert.Contains(t, q, "$input: CreateUserInput!")
	assert.Contains(t, q, "createUser(input: $input)")
}

func TestBaseBuilder_GetVariables(t *testing.T) {
	b := NewBaseBuilder(nil, "query", "Q", "f")
	b.SetArg("a", 1, "Int")
	b.SetArg("b", "two", "String")
	v := b.GetVariables()
	assert.Len(t, v, 2)
	assert.Equal(t, 1, v["a"])
	assert.Equal(t, "two", v["b"])
}
