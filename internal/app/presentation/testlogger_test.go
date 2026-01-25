package presentation

import "business/internal/library/logger"

// testLogger is a no-op logger implementation for testing.
type testLogger struct{}

var _ logger.Interface = (*testLogger)(nil)

func newTestLogger() logger.Interface {
	return &testLogger{}
}

func (l *testLogger) Debug(message string, fields ...logger.Field) {}
func (l *testLogger) Info(message string, fields ...logger.Field)  {}
func (l *testLogger) Warn(message string, fields ...logger.Field)  {}
func (l *testLogger) Error(message string, fields ...logger.Field) {}
func (l *testLogger) Fatal(message string, fields ...logger.Field) {}
func (l *testLogger) With(fields ...logger.Field) logger.Interface {
	return l
}
func (l *testLogger) Sync() error {
	return nil
}
