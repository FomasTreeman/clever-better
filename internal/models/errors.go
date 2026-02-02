package models

import "errors"

// Custom errors
var (
	ErrStrategyNameRequired = errors.New("strategy name is required")
	ErrNotFound            = errors.New("record not found")
	ErrDuplicateKey        = errors.New("duplicate key violation")
	ErrInvalidID           = errors.New("invalid ID format")
)
