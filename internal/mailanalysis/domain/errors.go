package domain

import "errors"

var (
	// ErrInvalidCommand is returned when the use case command is malformed.
	ErrInvalidCommand = errors.New("email analysis command is invalid")
	// ErrEmailForAnalysisInvalid is returned when an analysis target email is malformed.
	ErrEmailForAnalysisInvalid = errors.New("email for analysis is invalid")
	// ErrAnalysisResponseInvalid is returned when the analyzer response cannot be parsed.
	ErrAnalysisResponseInvalid = errors.New("analysis response is invalid")
)
