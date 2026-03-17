// Package schema provides functionality to fetch a GraphQL schema via the
// introspection query, parse the JSON response into Go types, and convert
// the result into GraphQL SDL (Schema Definition Language) text.
package schema

// GraphQLError represents a single error object returned in the "errors" array
// of a GraphQL response.
type GraphQLError struct {
	Message string `json:"message"`
}

// IntrospectionResponse is the top-level JSON structure returned by a GraphQL
// endpoint when executing an introspection query. It contains either data or errors.
type IntrospectionResponse struct {
	Data   IntrospectionData `json:"data"`
	Errors []GraphQLError    `json:"errors"`
}

// IntrospectionData wraps the __schema field that contains the full schema
// definition returned by introspection.
type IntrospectionData struct {
	Schema IntrospectionSchema `json:"__schema"`
}

// IntrospectionSchema represents the complete GraphQL schema as returned by
// introspection, including root operation types, all type definitions, and
// directive declarations.
type IntrospectionSchema struct {
	QueryType        *TypeRef    `json:"queryType"`        // Root query type reference (e.g., "Query")
	MutationType     *TypeRef    `json:"mutationType"`     // Root mutation type reference, nil if not defined
	SubscriptionType *TypeRef    `json:"subscriptionType"` // Root subscription type reference, nil if not defined
	Types            []FullType  `json:"types"`            // All types in the schema, including built-in introspection types
	Directives       []Directive `json:"directives"`       // All directives, including built-in ones like @skip and @include
}

// TypeRef is a lightweight reference to a named type, used by the schema to
// identify root operation types (query, mutation, subscription).
type TypeRef struct {
	Name string `json:"name"`
}

// FullType represents a complete GraphQL type definition from introspection.
// Depending on the Kind, different fields are populated:
//   - SCALAR: only Name and Description
//   - OBJECT: Fields and Interfaces
//   - INTERFACE: Fields, Interfaces (parent interfaces), and PossibleTypes
//   - UNION: PossibleTypes
//   - ENUM: EnumValues
//   - INPUT_OBJECT: InputFields
type FullType struct {
	Kind          string       `json:"kind"`
	Name          string       `json:"name"`
	Description   *string      `json:"description"`
	Fields        []Field      `json:"fields"`        // Non-nil for OBJECT and INTERFACE kinds
	InputFields   []InputValue `json:"inputFields"`   // Non-nil for INPUT_OBJECT kind
	Interfaces    []TypeInfo   `json:"interfaces"`    // Interfaces implemented by this type
	EnumValues    []EnumValue  `json:"enumValues"`    // Non-nil for ENUM kind
	PossibleTypes []TypeInfo   `json:"possibleTypes"` // Member types for UNION and INTERFACE kinds
}

// Field represents a single field on an OBJECT or INTERFACE type, including
// its arguments, return type, and deprecation status.
type Field struct {
	Name              string       `json:"name"`
	Description       *string      `json:"description"`
	Args              []InputValue `json:"args"`              // Field arguments (may be empty)
	Type              TypeInfo     `json:"type"`              // Return type of the field
	IsDeprecated      bool         `json:"isDeprecated"`
	DeprecationReason *string      `json:"deprecationReason"`
}

// InputValue represents a field argument or an input object field. It carries
// type information and an optional default value as a JSON-encoded string.
type InputValue struct {
	Name         string   `json:"name"`
	Description  *string  `json:"description"`
	Type         TypeInfo `json:"type"`         // The type of this input value
	DefaultValue *string  `json:"defaultValue"` // Default value as a GraphQL literal string, nil if none
}

// TypeInfo represents a type reference in the introspection schema. Types are
// represented as a recursive wrapper structure:
//   - Named types (SCALAR, OBJECT, etc.) have Kind and Name set, OfType is nil.
//   - NON_NULL and LIST are wrapper types where OfType points to the inner type.
//
// For example, [String!]! is NON_NULL -> LIST -> NON_NULL -> SCALAR(String).
type TypeInfo struct {
	Kind   string    `json:"kind"`
	Name   *string   `json:"name"`   // Non-nil only for named (non-wrapper) types
	OfType *TypeInfo `json:"ofType"` // Non-nil only for NON_NULL and LIST wrapper types
}

// EnumValue represents a single value within a GraphQL ENUM type, including
// its deprecation status.
type EnumValue struct {
	Name              string  `json:"name"`
	Description       *string `json:"description"`
	IsDeprecated      bool    `json:"isDeprecated"`
	DeprecationReason *string `json:"deprecationReason"`
}

// Directive represents a GraphQL directive declaration, including the schema
// locations where it can be applied and its accepted arguments.
type Directive struct {
	Name        string       `json:"name"`
	Description *string      `json:"description"`
	Locations   []string     `json:"locations"` // Valid locations, e.g., "FIELD", "OBJECT", "ARGUMENT_DEFINITION"
	Args        []InputValue `json:"args"`
}
