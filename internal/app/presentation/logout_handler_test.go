package presentation

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestLogoutHandlerClearsCookie(t *testing.T) {
	t.Parallel()
	controller := newTestAuthController(new(mockAuthUseCase), newTestLogger())
	router := gin.New()
	router.POST("/auth/logout", controller.Logout)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusNoContent, resp.Code)

	setCookie := resp.Header().Get("Set-Cookie")
	if assert.NotEmpty(t, setCookie) {
		assert.Contains(t, setCookie, "access_token=")
		assert.Contains(t, setCookie, "Max-Age=0") // Max-Age is set to 0 when deleting
		assert.Contains(t, setCookie, "Path=/")
		assert.Contains(t, setCookie, "Domain=localhost")
		assert.Contains(t, setCookie, "HttpOnly")
		assert.Contains(t, setCookie, "SameSite=Lax")
		assert.NotContains(t, setCookie, "Secure")
	}
}

func TestLogoutHandlerMarksSecureWhenNeeded(t *testing.T) {
	t.Parallel()
	controller := newTestAuthControllerWithVars(new(mockAuthUseCase), newTestLogger(), map[string]string{"APP": "production"})
	router := gin.New()
	router.POST("/auth/logout", controller.Logout)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	setCookie := resp.Header().Get("Set-Cookie")
	if assert.NotEmpty(t, setCookie) {
		assert.Contains(t, setCookie, "Secure")
	}
}
