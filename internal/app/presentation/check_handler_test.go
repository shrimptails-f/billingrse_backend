package presentation

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCheckHandler(t *testing.T) {

	t.Run("success", func(t *testing.T) {
		usecase := new(mockAuthUseCase)
		controller := newTestAuthController(usecase, newTestLogger())
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("userID", uint(123))
			c.Next()
		})
		router.GET("/auth/check", controller.Check)

		req := httptest.NewRequest(http.MethodGet, "/auth/check", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.JSONEq(t, `{"user_id":123}`, resp.Body.String())
	})

	t.Run("userID not set", func(t *testing.T) {
		usecase := new(mockAuthUseCase)
		controller := newTestAuthController(usecase, newTestLogger())
		router := gin.New()
		router.GET("/auth/check", controller.Check)

		req := httptest.NewRequest(http.MethodGet, "/auth/check", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusInternalServerError, resp.Code)
		assert.Empty(t, resp.Body.String())
	})

	t.Run("userID wrong type", func(t *testing.T) {
		usecase := new(mockAuthUseCase)
		controller := newTestAuthController(usecase, newTestLogger())
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("userID", "not-a-uint")
			c.Next()
		})
		router.GET("/auth/check", controller.Check)

		req := httptest.NewRequest(http.MethodGet, "/auth/check", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusInternalServerError, resp.Code)
		assert.Empty(t, resp.Body.String())
	})
}
