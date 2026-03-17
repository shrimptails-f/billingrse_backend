package middleware

import (
	"business/internal/library/logger"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestID_GeneratesRequestIDAndPropagatesToContext(t *testing.T) {
	router := gin.New()
	router.Use(RequestID())
	router.GET("/ping", func(c *gin.Context) {
		requestID, exists := c.Get("request_id")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "request_id not found"})
			return
		}

		contextRequestID, ok := logger.RequestIDFromContext(c.Request.Context())
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "context request_id not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"request_id":         requestID,
			"context_request_id": contextRequestID,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.NotEmpty(t, resp.Header().Get("X-Request-Id"))
	assert.Contains(t, resp.Body.String(), resp.Header().Get("X-Request-Id"))
}

func TestRequestID_UsesInboundRequestID(t *testing.T) {
	router := gin.New()
	router.Use(RequestID())
	router.GET("/ping", func(c *gin.Context) {
		requestID, _ := c.Get("request_id")
		contextRequestID, _ := logger.RequestIDFromContext(c.Request.Context())

		c.JSON(http.StatusOK, gin.H{
			"request_id":         requestID,
			"context_request_id": contextRequestID,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("X-Request-Id", "req-123")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "req-123", resp.Header().Get("X-Request-Id"))
	assert.JSONEq(t, `{"request_id":"req-123","context_request_id":"req-123"}`, resp.Body.String())
}

func TestRequestSummary_UsesContextFields(t *testing.T) {
	router := gin.New()
	router.Use(RequestID())
	router.Use(RequestSummary(logger.NewNop()))
	router.GET("/ping", func(c *gin.Context) {
		c.Set("userID", uint(42))
		requestCtx, err := logger.ContextWithUserID(c.Request.Context(), 42)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		c.Request = c.Request.WithContext(requestCtx)
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusNoContent, resp.Code)
	assert.NotEmpty(t, resp.Header().Get("X-Request-Id"))
}

func TestRequestSummary_DoesNotLogUnauthorizedRequest(t *testing.T) {
	spy := &capturingLogger{}

	router := gin.New()
	router.Use(RequestID())
	router.Use(RequestSummary(spy))
	router.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusUnauthorized)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	require.Len(t, spy.entries, 1)
	assert.Equal(t, "info", spy.entries[0].level)
	assert.Equal(t, "http_request_started", spy.entries[0].message)
}

func TestRequestSummary_LogsRejectedRequestAsInfo(t *testing.T) {
	spy := &capturingLogger{}

	router := gin.New()
	router.Use(RequestID())
	router.Use(RequestSummary(spy))
	router.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusForbidden)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusForbidden, resp.Code)
	require.Len(t, spy.entries, 2)
	assert.Equal(t, "info", spy.entries[0].level)
	assert.Equal(t, "http_request_started", spy.entries[0].message)
	assert.Equal(t, "info", spy.entries[1].level)
	assert.Equal(t, "http_request_rejected", spy.entries[1].message)
}

func TestRequestSummary_LogsStartedAndSucceeded(t *testing.T) {
	spy := &capturingLogger{}

	router := gin.New()
	router.Use(RequestID())
	router.Use(RequestSummary(spy))
	router.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusNoContent, resp.Code)
	require.Len(t, spy.entries, 2)
	assert.Equal(t, "info", spy.entries[0].level)
	assert.Equal(t, "http_request_started", spy.entries[0].message)
	assert.Equal(t, "info", spy.entries[1].level)
	assert.Equal(t, "http_request_succeeded", spy.entries[1].message)
}

type capturingLogger struct {
	entries []capturedEntry
}

type capturedEntry struct {
	level   string
	message string
}

func (l *capturingLogger) Debug(message string, fields ...logger.Field) {}

func (l *capturingLogger) Info(message string, fields ...logger.Field) {
	l.entries = append(l.entries, capturedEntry{level: "info", message: message})
}

func (l *capturingLogger) Warn(message string, fields ...logger.Field) {
	l.entries = append(l.entries, capturedEntry{level: "warn", message: message})
}

func (l *capturingLogger) Error(message string, fields ...logger.Field) {
	l.entries = append(l.entries, capturedEntry{level: "error", message: message})
}

func (l *capturingLogger) Fatal(message string, fields ...logger.Field) {}

func (l *capturingLogger) With(fields ...logger.Field) logger.Interface {
	return l
}

func (l *capturingLogger) WithContext(ctx context.Context) (logger.Interface, error) {
	return l, nil
}

func (l *capturingLogger) Sync() error {
	return nil
}
