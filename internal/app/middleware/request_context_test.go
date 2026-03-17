package middleware

import (
	"business/internal/library/logger"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
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
		c.Request = c.Request.WithContext(logger.ContextWithUserID(c.Request.Context(), 42))
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusNoContent, resp.Code)
	assert.NotEmpty(t, resp.Header().Get("X-Request-Id"))
}
