package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCsrfOriginCheck_AllowsConfiguredOrigin(t *testing.T) {
	router := gin.New()
	router.Use(CsrfOriginCheck("https://example.com"))
	router.POST("/submit", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/submit", nil)
	req.Header.Set("Origin", "https://example.com/form")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusNoContent, resp.Code)
	assert.Empty(t, resp.Body.String())
}

func TestCsrfOriginCheck_RejectsUnapprovedOriginWithStandardErrorResponse(t *testing.T) {
	router := gin.New()
	router.Use(CsrfOriginCheck("https://example.com"))
	router.POST("/submit", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/submit", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusForbidden, resp.Code)
	assert.JSONEq(t, `{"error":{"code":"csrf_origin_not_allowed","message":"オリジンまたはリファラが許可されていません。"}}`, resp.Body.String())
}
