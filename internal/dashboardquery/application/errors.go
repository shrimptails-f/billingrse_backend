package application

import "errors"

var (
	// ErrInvalidSummaryQuery is returned when the dashboard summary query is invalid.
	ErrInvalidSummaryQuery = errors.New("dashboard summary query is invalid")
)
