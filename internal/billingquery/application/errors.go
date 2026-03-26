package application

import "errors"

var (
	// ErrInvalidListQuery is returned when the billing list query is invalid.
	ErrInvalidListQuery = errors.New("billing list query is invalid")
	// ErrInvalidMonthlyTrendQuery is returned when the billing monthly trend query is invalid.
	ErrInvalidMonthlyTrendQuery = errors.New("billing monthly trend query is invalid")
	// ErrInvalidMonthDetailQuery is returned when the billing month detail query is invalid.
	ErrInvalidMonthDetailQuery = errors.New("billing month detail query is invalid")
)
