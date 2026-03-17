package middleware

import (
	"business/internal/library/logger"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	requestIDHeader      = "X-Request-Id"
	ginContextRequestID  = "request_id"
	ginContextUserID     = "userID"
	requestLogComponent  = "http_request"
	recoveryLogComponent = "panic_recovery"
)

// RequestID attaches a request_id to gin.Context and request.Context.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader(requestIDHeader))
		if requestID == "" {
			requestID = uuid.NewString()
		}

		c.Set(ginContextRequestID, requestID)
		c.Header(requestIDHeader, requestID)
		c.Request = c.Request.WithContext(logger.ContextWithRequestID(c.Request.Context(), requestID))
		c.Next()
	}
}

// RequestSummary writes an HTTP request summary log after the handler chain completes.
func RequestSummary(log logger.Interface) gin.HandlerFunc {
	if log == nil {
		log = logger.NewNop()
	}

	summaryLogger := log.With(logger.Component(requestLogComponent))

	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		reqLog := summaryLogger.WithContext(c.Request.Context())
		fields := []logger.Field{
			logger.String("method", c.Request.Method),
			logger.String("path", resolvedPath(c)),
			logger.HTTPStatusCode(c.Writer.Status()),
			logger.Int("latency_ms", int(time.Since(start).Milliseconds())),
		}

		switch status := c.Writer.Status(); {
		case status >= http.StatusInternalServerError:
			reqLog.Error("http request failed", fields...)
		case status >= http.StatusBadRequest:
			reqLog.Warn("http request rejected", fields...)
		default:
			reqLog.Info("http request succeeded", fields...)
		}
	}
}

// Recovery recovers panics and emits a structured panic log.
func Recovery(log logger.Interface) gin.HandlerFunc {
	if log == nil {
		log = logger.NewNop()
	}

	recoveryLogger := log.With(logger.Component(recoveryLogComponent))

	return gin.CustomRecoveryWithWriter(io.Discard, func(c *gin.Context, recovered interface{}) {
		reqLog := recoveryLogger.WithContext(c.Request.Context())
		reqLog.Error(
			"panic recovered",
			logger.Recovered(recovered),
			logger.String("method", c.Request.Method),
			logger.String("path", resolvedPath(c)),
			logger.HTTPStatusCode(http.StatusInternalServerError),
			logger.StackTrace(),
		)
		c.AbortWithStatus(http.StatusInternalServerError)
	})
}

func resolvedPath(c *gin.Context) string {
	if c == nil {
		return ""
	}

	if path := strings.TrimSpace(c.FullPath()); path != "" {
		return path
	}

	return c.Request.URL.Path
}
