package clientgen

import "errors"

// Sentinel errors returned by Config.Validate and other initialization paths.
var (
	ErrSchemaPathRequired = errors.New("schema path is required")
	ErrSchemaNotFound     = errors.New("schema file not found")
	ErrSchemaParseFailed  = errors.New("failed to parse schema")
)
