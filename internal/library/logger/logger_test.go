package logger

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestParseLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  zapcore.Level
	}{
		{name: "debug", input: "debug", want: zapcore.DebugLevel},
		{name: "warn", input: "warn", want: zapcore.WarnLevel},
		{name: "error", input: "error", want: zapcore.ErrorLevel},
		{name: "panic", input: "panic", want: zapcore.PanicLevel},
		{name: "fatal", input: "fatal", want: zapcore.FatalLevel},
		{name: "info", input: "info", want: zapcore.InfoLevel},
		{name: "mixed case", input: "DEBUG", want: zapcore.DebugLevel},
		{name: "unknown", input: "unexpected", want: zapcore.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, parseLevel(tt.input))
		})
	}
}

func TestRequestIDFromContext(t *testing.T) {
	t.Parallel()

	t.Run("round trip", func(t *testing.T) {
		t.Parallel()

		ctx := mustContextWithRequestID(t, context.Background(), "req-123")

		got, ok := RequestIDFromContext(ctx)

		assert.True(t, ok)
		assert.Equal(t, "req-123", got)
	})

	t.Run("nil context", func(t *testing.T) {
		t.Parallel()

		got, ok := RequestIDFromContext(nil)

		assert.False(t, ok)
		assert.Equal(t, "", got)
	})

	t.Run("missing value", func(t *testing.T) {
		t.Parallel()

		got, ok := RequestIDFromContext(context.Background())

		assert.False(t, ok)
		assert.Equal(t, "", got)
	})

	t.Run("empty value", func(t *testing.T) {
		t.Parallel()

		ctx := context.WithValue(context.Background(), requestIDContextKey, "")

		got, ok := RequestIDFromContext(ctx)

		assert.False(t, ok)
		assert.Equal(t, "", got)
	})

	t.Run("whitespace only value", func(t *testing.T) {
		t.Parallel()

		ctx := context.WithValue(context.Background(), requestIDContextKey, "   ")

		got, ok := RequestIDFromContext(ctx)

		assert.False(t, ok)
		assert.Equal(t, "", got)
	})

	t.Run("wrong type", func(t *testing.T) {
		t.Parallel()

		ctx := context.WithValue(context.Background(), requestIDContextKey, 123)

		got, ok := RequestIDFromContext(ctx)

		assert.False(t, ok)
		assert.Equal(t, "", got)
	})

	t.Run("context with request id rejects nil context", func(t *testing.T) {
		t.Parallel()

		ctx, err := ContextWithRequestID(nil, "req-123")

		require.ErrorIs(t, err, ErrNilContext)
		assert.Nil(t, ctx)
	})
}

func TestUserIDFromContext(t *testing.T) {
	t.Parallel()

	t.Run("round trip", func(t *testing.T) {
		t.Parallel()

		ctx := mustContextWithUserID(t, context.Background(), 42)

		got, ok := UserIDFromContext(ctx)

		assert.True(t, ok)
		assert.EqualValues(t, 42, got)
	})

	t.Run("nil context", func(t *testing.T) {
		t.Parallel()

		got, ok := UserIDFromContext(nil)

		assert.False(t, ok)
		assert.EqualValues(t, 0, got)
	})

	t.Run("missing value", func(t *testing.T) {
		t.Parallel()

		got, ok := UserIDFromContext(context.Background())

		assert.False(t, ok)
		assert.EqualValues(t, 0, got)
	})

	t.Run("wrong type", func(t *testing.T) {
		t.Parallel()

		ctx := context.WithValue(context.Background(), userIDContextKey, int(42))

		got, ok := UserIDFromContext(ctx)

		assert.False(t, ok)
		assert.EqualValues(t, 0, got)
	})

	t.Run("context with user id rejects nil context", func(t *testing.T) {
		t.Parallel()

		ctx, err := ContextWithUserID(nil, 42)

		require.ErrorIs(t, err, ErrNilContext)
		assert.Nil(t, ctx)
	})
}

func TestWithContextAddsRequestScopedFields(t *testing.T) {
	t.Parallel()

	base, logs := newObservedLogger(zapcore.DebugLevel)
	ctx := mustContextWithRequestID(t, context.Background(), "req-123")
	ctx = mustContextWithUserID(t, ctx, 42)

	reqLog, err := base.WithContext(ctx)
	require.NoError(t, err)

	reqLog.Info("hello")

	entry := singleEntry(t, logs)
	fields := entry.ContextMap()

	assert.Equal(t, "hello", entry.Message)
	assert.Equal(t, "req-123", fields["request_id"])
	assert.EqualValues(t, 42, fields["user_id"])
}

func TestWithContextIgnoresBlankRequestID(t *testing.T) {
	t.Parallel()

	base, logs := newObservedLogger(zapcore.DebugLevel)
	ctx := context.WithValue(context.Background(), requestIDContextKey, "   ")
	ctx = mustContextWithUserID(t, ctx, 7)

	reqLog, err := base.WithContext(ctx)
	require.NoError(t, err)

	reqLog.Info("hello")

	entry := singleEntry(t, logs)
	fields := entry.ContextMap()

	_, hasRequestID := fields["request_id"]
	assert.False(t, hasRequestID)
	assert.EqualValues(t, 7, fields["user_id"])
}

func TestWithContextReturnsBaseLoggerWhenContextHasNoFields(t *testing.T) {
	t.Parallel()

	base, logs := newObservedLogger(zapcore.DebugLevel)

	got, err := base.WithContext(context.TODO())

	require.NoError(t, err)
	assert.Same(t, base, got)

	got.Info("hello")

	entry := singleEntry(t, logs)
	assert.Equal(t, "hello", entry.Message)
	assert.Empty(t, entry.ContextMap())
}

func TestWithContextReturnsErrorForNilContext(t *testing.T) {
	t.Parallel()

	base, _ := newObservedLogger(zapcore.DebugLevel)

	log, err := base.WithContext(nil)

	require.ErrorIs(t, err, ErrNilContext)
	assert.Nil(t, log)
}

func TestWithContextWithNilBaseDoesNotPanic(t *testing.T) {
	t.Parallel()

	ctx := mustContextWithRequestID(t, context.Background(), "req-123")

	log, err := withContextFields(nil, ctx)

	require.NotNil(t, log)
	require.NoError(t, err)
	assert.NotPanics(t, func() {
		log.Info("hello")
		_ = log.Sync()
	})
}

func TestStringRedactsSensitiveKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		key      string
		value    string
		expected string
	}{
		{name: "non sensitive", key: "name", value: "Alice", expected: "Alice"},
		{name: "email", key: "email", value: "alice@example.com", expected: "[REDACTED]"},
		{name: "token", key: "token", value: "secret", expected: "[REDACTED]"},
		{name: "authorization", key: "authorization", value: "Bearer secret", expected: "[REDACTED]"},
		{name: "access token", key: "access_token", value: "secret", expected: "[REDACTED]"},
		{name: "set cookie", key: "set-cookie", value: "secret", expected: "[REDACTED]"},
		{name: "normalized key", key: " Access Token ", value: "secret", expected: "[REDACTED]"},
		{name: "similar but allowed", key: "token_type", value: "Bearer", expected: "Bearer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			field := String(tt.key, tt.value)
			encoded := fieldToMap(t, field)

			assert.Equal(t, tt.key, field.Key)
			assert.Equal(t, tt.expected, encoded[tt.key])
		})
	}
}

func TestSchemaHelpersUseExpectedKeys(t *testing.T) {
	t.Parallel()

	t.Run("component", func(t *testing.T) {
		t.Parallel()

		field := Component("auth_controller")
		encoded := fieldToMap(t, field)

		assert.Equal(t, "component", field.Key)
		assert.Equal(t, "auth_controller", encoded["component"])
	})

	t.Run("request id", func(t *testing.T) {
		t.Parallel()

		field := RequestID("req-1")
		encoded := fieldToMap(t, field)

		assert.Equal(t, "request_id", field.Key)
		assert.Equal(t, "req-1", encoded["request_id"])
	})

	t.Run("job id", func(t *testing.T) {
		t.Parallel()

		field := JobID("job-1")
		encoded := fieldToMap(t, field)

		assert.Equal(t, "job_id", field.Key)
		assert.Equal(t, "job-1", encoded["job_id"])
	})

	t.Run("http status code", func(t *testing.T) {
		t.Parallel()

		field := HTTPStatusCode(500)
		encoded := fieldToMap(t, field)

		assert.Equal(t, "http_status_code", field.Key)
		assert.EqualValues(t, 500, encoded["http_status_code"])
	})

	t.Run("user id", func(t *testing.T) {
		t.Parallel()

		field := UserID(42)
		encoded := fieldToMap(t, field)

		assert.Equal(t, "user_id", field.Key)
		assert.EqualValues(t, 42, encoded["user_id"])
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		err := errors.New("boom")
		field := Err(err)
		encoded := fieldToMap(t, field)

		assert.Equal(t, "error", field.Key)
		assert.Equal(t, zapcore.ErrorType, field.Type)
		assert.Equal(t, err, field.Interface)
		assert.Equal(t, "boom", encoded["error"])
	})
}

func TestStackTraceReturnsNonEmptyField(t *testing.T) {
	t.Parallel()

	field := StackTrace()
	encoded := fieldToMap(t, field)

	value, ok := encoded["stack_trace"].(string)
	require.True(t, ok)

	assert.Equal(t, "stack_trace", field.Key)
	assert.NotEmpty(t, value)
	assert.Contains(t, value, "runtime/debug.Stack")
}

func TestWithAddsStaticFields(t *testing.T) {
	t.Parallel()

	base, logs := newObservedLogger(zapcore.DebugLevel)

	base.With(Component("auth_controller")).Info("hello")

	entry := singleEntry(t, logs)
	fields := entry.ContextMap()

	assert.Equal(t, "auth_controller", fields["component"])
}

func TestWithAndWithContextComposeFields(t *testing.T) {
	t.Parallel()

	base, logs := newObservedLogger(zapcore.DebugLevel)
	ctx := mustContextWithRequestID(t, context.Background(), "req-123")

	child, err := base.With(Component("auth_controller")).WithContext(ctx)
	require.NoError(t, err)

	child.Info("hello")

	entry := singleEntry(t, logs)
	fields := entry.ContextMap()

	assert.Equal(t, "auth_controller", fields["component"])
	assert.Equal(t, "req-123", fields["request_id"])
}

func TestNewNopMethodsDoNotPanic(t *testing.T) {
	t.Parallel()

	log := NewNop()
	ctx := mustContextWithRequestID(t, context.Background(), "req-123")
	ctx = mustContextWithUserID(t, ctx, 42)

	assert.NotPanics(t, func() {
		log.Debug("debug")
		log.Info("info")
		log.Warn("warn")
		log.Error("error", Err(errors.New("boom")))

		child, err := log.With(Component("test_component")).WithContext(ctx)
		require.NoError(t, err)
		child.Info("child")
		_ = child.Sync()
	})
	assert.NoError(t, log.Sync())
}

func TestNewCreatesUsableLogger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		level string
	}{
		{name: "debug", level: "debug"},
		{name: "unknown falls back to info", level: "unexpected"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			log, err := New(tt.level)

			require.NoError(t, err)
			require.NotNil(t, log)

			ctx := mustContextWithRequestID(t, context.Background(), "req-123")
			child, err := log.With(Component("test_component")).WithContext(ctx)

			require.NoError(t, err)
			require.NotNil(t, child)

			_ = child.Sync()
			_ = log.Sync()
		})
	}
}

func newObservedLogger(level zapcore.Level) (*Logger, *observer.ObservedLogs) {
	core, logs := observer.New(level)
	return &Logger{base: zap.New(core)}, logs
}

func singleEntry(t *testing.T, logs *observer.ObservedLogs) observer.LoggedEntry {
	t.Helper()

	entries := logs.AllUntimed()
	require.Len(t, entries, 1)

	return entries[0]
}

func fieldToMap(t *testing.T, field Field) map[string]interface{} {
	t.Helper()

	encoder := zapcore.NewMapObjectEncoder()
	field.AddTo(encoder)

	return encoder.Fields
}

func mustContextWithRequestID(t *testing.T, ctx context.Context, requestID string) context.Context {
	t.Helper()

	next, err := ContextWithRequestID(ctx, requestID)
	require.NoError(t, err)

	return next
}

func mustContextWithUserID(t *testing.T, ctx context.Context, userID uint) context.Context {
	t.Helper()

	next, err := ContextWithUserID(ctx, userID)
	require.NoError(t, err)

	return next
}
