package logger

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
	Sync() error
}

// Logger is a thin wrapper around zap.Logger that satisfies Interface.
type Logger struct {
	base *zap.Logger
}

var _ Interface = (*Logger)(nil)

// New creates a zap-backed logger that emits JSON logs to stdout.
func New(level string) (*Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.Encoding = "json"
	cfg.DisableStacktrace = true
	cfg.EncoderConfig = zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	cfg.Level = zap.NewAtomicLevelAt(parseLevel(level))

	base, err := cfg.Build(zap.AddCaller(), zap.AddCallerSkip(1))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize zap logger: %w", err)
	}

	return &Logger{base: base}, nil
}

// NewNop creates a no-op logger for testing that discards all log entries.
func NewNop() Interface {
	return &Logger{base: zap.NewNop()}
}

// Debug writes a debug level log entry.
func (l *Logger) Debug(message string, fields ...Field) {
	l.base.Debug(message, fields...)
}

// Info writes an info level log entry.
func (l *Logger) Info(message string, fields ...Field) {
	l.base.Info(message, fields...)
}

// Warn writes a warning log entry.
func (l *Logger) Warn(message string, fields ...Field) {
	l.base.Warn(message, fields...)
}

// Error writes an error level log entry.
func (l *Logger) Error(message string, fields ...Field) {
	l.base.Error(message, fields...)
}

// Fatal writes a fatal log entry and exits the process.
func (l *Logger) Fatal(message string, fields ...Field) {
	l.base.Fatal(message, fields...)
}

// With returns a child logger that always includes the provided fields.
func (l *Logger) With(fields ...Field) Interface {
	return &Logger{base: l.base.With(fields...)}
}

// Sync flushes any buffered log entries.
func (l *Logger) Sync() error {
	return l.base.Sync()
}

func parseLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// Any attaches an arbitrary value to the log entry.
func Any(key string, value interface{}) Field {
	return zap.Any(key, value)
}

// String attaches a string value to the log entry.
func String(key string, value string) Field {
	return zap.String(key, value)
}

// Bool attaches a boolean value to the log entry.
func Bool(key string, value bool) Field {
	return zap.Bool(key, value)
}

// Int attaches an integer value to the log entry.
func Int(key string, value int) Field {
	return zap.Int(key, value)
}

// Uint attaches an unsigned integer value to the log entry.
func Uint(key string, value uint) Field {
	return zap.Uint(key, value)
}

// Err attaches an error to the log entry.
func Err(err error) Field {
	return zap.Error(err)
}
