package clientgents

import "errors"

// Sentinel errors returned by Config.Validate.
var (
	ErrSchemaPathRequired = errors.New("schema path is required")
	ErrConfigPathRequired = errors.New("config path is required")
)
