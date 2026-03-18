package middleware

import (
	"business/internal/app/httpresponse"
	"business/internal/library/logger"
	"business/internal/library/timewrapper"
	"io"
	"net/http"
	"strings"

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
		requestCtx, err := logger.ContextWithRequestID(c.Request.Context(), requestID)
		if err != nil {
			httpresponse.AbortInternalServerError(c)
			return
		}

		c.Request = c.Request.WithContext(requestCtx)
		c.Next()
	}
}

// RequestSummary writes an HTTP request summary log after the handler chain completes.
func RequestSummary(log logger.Interface, clock timewrapper.ClockInterface) gin.HandlerFunc {
	if log == nil {
		log = logger.NewNop()
	}
	if clock == nil {
		clock = timewrapper.NewClock()
	}

	summaryLogger := log.With(logger.Component(requestLogComponent))

	return func(c *gin.Context) {
		start := clock.Now()
		reqLog, err := summaryLogger.WithContext(c.Request.Context())
		if err != nil {
			reqLog = summaryLogger
		}

		reqLog.Info(
			"http_request_started",
			logger.String("method", c.Request.Method),
			logger.String("path", resolvedPath(c)),
		)

		c.Next()

		reqLog, err = summaryLogger.WithContext(c.Request.Context())
		if err != nil {
			reqLog = summaryLogger
		}

		fields := []logger.Field{
			logger.String("method", c.Request.Method),
			logger.String("path", resolvedPath(c)),
			logger.HTTPStatusCode(c.Writer.Status()),
			logger.Int("latency_ms", int(clock.Now().Sub(start).Milliseconds())),
		}

		switch status := c.Writer.Status(); {
		case status >= http.StatusInternalServerError:
			reqLog.Error("http_request_failed", fields...)
		case status == http.StatusUnauthorized:
			// 認証失敗系は別のミドルウェアで記録する。
		case status >= http.StatusBadRequest:
			reqLog.Info("http_request_rejected", fields...)
		default:
			reqLog.Info("http_request_succeeded", fields...)
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
		reqLog, err := recoveryLogger.WithContext(c.Request.Context())
		if err != nil {
			reqLog = recoveryLogger
		}

		reqLog.Error(
			"panic recovered",
			logger.Recovered(recovered),
			logger.String("method", c.Request.Method),
			logger.String("path", resolvedPath(c)),
			logger.HTTPStatusCode(http.StatusInternalServerError),
			logger.StackTrace(),
		)
		httpresponse.AbortInternalServerError(c)
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
