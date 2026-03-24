package domain

import "errors"

var (
	// ErrInvalidCommand is returned when the billing eligibility command is invalid.
	ErrInvalidCommand = errors.New("billing eligibility command is invalid")
)
