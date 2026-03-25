package domain

import "errors"

var (
	// ErrInvalidCommand is returned when the billing command is invalid.
	ErrInvalidCommand = errors.New("billing command is invalid")
	// ErrInvalidListQuery is returned when the billing list query is invalid.
	ErrInvalidListQuery = errors.New("billing list query is invalid")
)
