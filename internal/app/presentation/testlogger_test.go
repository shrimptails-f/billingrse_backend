package presentation

import (
	"business/internal/library/logger"
	"context"
)

// testLogger is a no-op logger implementation for testing.
type testLogger struct{}

type capturedLogEntry struct {
	level   string
	message string
}

type capturingTestLogger struct {
	entries []capturedLogEntry
}

var _ logger.Interface = (*testLogger)(nil)

func newTestLogger() logger.Interface {
	return &testLogger{}
}

func newCapturingTestLogger() *capturingTestLogger {
	return &capturingTestLogger{}
}

func (l *testLogger) Debug(message string, fields ...logger.Field) {}
func (l *testLogger) Info(message string, fields ...logger.Field)  {}
func (l *testLogger) Warn(message string, fields ...logger.Field)  {}
func (l *testLogger) Error(message string, fields ...logger.Field) {}
func (l *testLogger) Fatal(message string, fields ...logger.Field) {}
func (l *testLogger) With(fields ...logger.Field) logger.Interface {
	return l
}
func (l *testLogger) WithContext(ctx context.Context) (logger.Interface, error) {
	return l, nil
}
func (l *testLogger) Sync() error {
	return nil
}

func (l *capturingTestLogger) Debug(message string, fields ...logger.Field) {}

func (l *capturingTestLogger) Info(message string, fields ...logger.Field) {
	l.entries = append(l.entries, capturedLogEntry{level: "info", message: message})
}

func (l *capturingTestLogger) Warn(message string, fields ...logger.Field) {
	l.entries = append(l.entries, capturedLogEntry{level: "warn", message: message})
}

func (l *capturingTestLogger) Error(message string, fields ...logger.Field) {
	l.entries = append(l.entries, capturedLogEntry{level: "error", message: message})
}

func (l *capturingTestLogger) Fatal(message string, fields ...logger.Field) {}

func (l *capturingTestLogger) With(fields ...logger.Field) logger.Interface {
	return l
}

func (l *capturingTestLogger) WithContext(ctx context.Context) (logger.Interface, error) {
	return l, nil
}

func (l *capturingTestLogger) Sync() error {
	return nil
}
