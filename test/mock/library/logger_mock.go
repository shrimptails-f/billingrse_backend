package mocklibrary

import (
	"business/internal/library/logger"
	"context"
)

// CapturedLogEntry stores the log level and message emitted by a mock logger.
type CapturedLogEntry struct {
	Level   string
	Message string
}

// NopLogger is a reusable no-op logger for tests.
type NopLogger struct{}

// CapturingLogger records log messages for assertions in tests.
type CapturingLogger struct {
	Entries []CapturedLogEntry
}

var _ logger.Interface = (*NopLogger)(nil)
var _ logger.Interface = (*CapturingLogger)(nil)

// NewNopLogger returns a logger that discards all messages.
func NewNopLogger() logger.Interface {
	return &NopLogger{}
}

// NewCapturingLogger returns a logger that records emitted messages.
func NewCapturingLogger() *CapturingLogger {
	return &CapturingLogger{}
}

func (l *NopLogger) Debug(message string, fields ...logger.Field) {}
func (l *NopLogger) Info(message string, fields ...logger.Field)  {}
func (l *NopLogger) Warn(message string, fields ...logger.Field)  {}
func (l *NopLogger) Error(message string, fields ...logger.Field) {}
func (l *NopLogger) Fatal(message string, fields ...logger.Field) {}

func (l *NopLogger) With(fields ...logger.Field) logger.Interface {
	return l
}

func (l *NopLogger) WithContext(ctx context.Context) (logger.Interface, error) {
	return l, nil
}

func (l *NopLogger) Sync() error {
	return nil
}

func (l *CapturingLogger) Debug(message string, fields ...logger.Field) {}

func (l *CapturingLogger) Info(message string, fields ...logger.Field) {
	l.Entries = append(l.Entries, CapturedLogEntry{Level: "info", Message: message})
}

func (l *CapturingLogger) Warn(message string, fields ...logger.Field) {
	l.Entries = append(l.Entries, CapturedLogEntry{Level: "warn", Message: message})
}

func (l *CapturingLogger) Error(message string, fields ...logger.Field) {
	l.Entries = append(l.Entries, CapturedLogEntry{Level: "error", Message: message})
}

func (l *CapturingLogger) Fatal(message string, fields ...logger.Field) {}

func (l *CapturingLogger) With(fields ...logger.Field) logger.Interface {
	return l
}

func (l *CapturingLogger) WithContext(ctx context.Context) (logger.Interface, error) {
	return l, nil
}

func (l *CapturingLogger) Sync() error {
	return nil
}
