package logger

import (
	"context"
	"fmt"
	"runtime/debug"
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
	WithContext(ctx context.Context) Interface
	Sync() error
}

// Logger is a thin wrapper around zap.Logger that satisfies Interface.
type Logger struct {
	base *zap.Logger
}

type contextKey string

const (
	requestIDContextKey contextKey = "request_id"
	userIDContextKey    contextKey = "user_id"
)

var _ Interface = (*Logger)(nil)

// New creates a zap-backed logger that emits JSON logs to stdout.
func New(level string) (*Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.Encoding = "json"
	cfg.DisableStacktrace = true
	cfg.EncoderConfig = zapcore.EncoderConfig{
		TimeKey:        "",
		LevelKey:       "level",
		NameKey:        "",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stack_trace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
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

// WithContext enriches a logger with common request-scoped fields from context.Context.
func (l *Logger) WithContext(ctx context.Context) Interface {
	return withContextFields(l, ctx)
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

func withContextFields(base Interface, ctx context.Context) Interface {
	if base == nil {
		base = NewNop()
	}

	if ctx == nil {
		return base
	}

	log := base

	if requestID, ok := RequestIDFromContext(ctx); ok {
		log = log.With(RequestID(requestID))
	}

	if userID, ok := UserIDFromContext(ctx); ok {
		log = log.With(UserID(userID))
	}

	return log
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

// Component attaches a component name following the common log schema.
func Component(value string) Field {
	return String("component", value)
}

// RequestID attaches an HTTP request correlation ID.
func RequestID(value string) Field {
	return String("request_id", value)
}

// JobID attaches a batch or async job correlation ID.
func JobID(value string) Field {
	return String("job_id", value)
}

// String attaches a string value to the log entry.
func String(key string, value string) Field {
	return zap.String(key, sanitizeStringValue(key, value))
}

// Bool attaches a boolean value to the log entry.
func Bool(key string, value bool) Field {
	return zap.Bool(key, value)
}

// Int attaches an integer value to the log entry.
func Int(key string, value int) Field {
	return zap.Int(key, value)
}

// HTTPStatusCode attaches an HTTP status code following the common log schema.
func HTTPStatusCode(value int) Field {
	return Int("http_status_code", value)
}

// Uint attaches an unsigned integer value to the log entry.
func Uint(key string, value uint) Field {
	return zap.Uint(key, value)
}

// UserID attaches a user ID following the common log schema.
func UserID(value uint) Field {
	return Uint("user_id", value)
}

// Recovered attaches a recovered panic value.
func Recovered(value interface{}) Field {
	return Any("recovered", value)
}

// Err attaches an error to the log entry.
func Err(err error) Field {
	return zap.Error(err)
}

// StackTrace captures the current goroutine stack trace under the common schema key.
func StackTrace() Field {
	return zap.String("stack_trace", string(debug.Stack()))
}

// ContextWithRequestID stores request_id in context.Context.
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey, requestID)
}

// ContextWithUserID stores user_id in context.Context.
func ContextWithUserID(ctx context.Context, userID uint) context.Context {
	return context.WithValue(ctx, userIDContextKey, userID)
}

// RequestIDFromContext retrieves request_id from context.Context.
func RequestIDFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}

	requestID, ok := ctx.Value(requestIDContextKey).(string)
	if !ok || strings.TrimSpace(requestID) == "" {
		return "", false
	}

	return requestID, true
}

// UserIDFromContext retrieves user_id from context.Context.
func UserIDFromContext(ctx context.Context) (uint, bool) {
	if ctx == nil {
		return 0, false
	}

	userID, ok := ctx.Value(userIDContextKey).(uint)
	if !ok {
		return 0, false
	}

	return userID, true
}

func sanitizeStringValue(key string, value string) string {
	if !isSensitiveKey(key) {
		return value
	}

	return "[REDACTED]"
}

func isSensitiveKey(key string) bool {
	normalized := strings.NewReplacer("-", "_", " ", "_").Replace(strings.ToLower(strings.TrimSpace(key)))

	switch normalized {
	case "email",
		"mail",
		"phone",
		"address",
		"password",
		"token",
		"access_token",
		"refresh_token",
		"api_key",
		"apikey",
		"cookie",
		"set_cookie",
		"authorization",
		"session_id":
		return true
	default:
		return false
	}
}
