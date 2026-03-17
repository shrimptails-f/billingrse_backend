package logger

import (
	"context"

	"go.uber.org/zap"
)

// Field re-exports zap.Field so callers do not depend on zap directly.
type Field = zap.Field

// Interface represents the logger contract used across the app layers.
type Interface interface {
	Debug(message string, fields ...Field)
	Info(message string, fields ...Field)
	Warn(message string, fields ...Field)
	Error(message string, fields ...Field)
	Fatal(message string, fields ...Field)
	With(fields ...Field) Interface
	WithContext(ctx context.Context) Interface
	Sync() error
}
