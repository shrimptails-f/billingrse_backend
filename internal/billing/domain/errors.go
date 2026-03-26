package domain

import "errors"

var (
	// ErrInvalidCommand is returned when the billing command is invalid.
	ErrInvalidCommand = errors.New("billing command is invalid")
)
