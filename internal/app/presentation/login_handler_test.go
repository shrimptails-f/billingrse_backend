package presentation

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"business/internal/auth/application"
	"business/internal/auth/domain"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestLoginHandler(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		usecase.On("Login", mock.Anything, mock.MatchedBy(func(req domain.LoginRequest) bool {
			return req.Email == "user@example.com" && req.Password == "password123"
		})).Return("token", nil).Once()

		controller := newTestAuthControllerWithVars(usecase, newTestLogger(), map[string]string{"APP": "local"})
		router := gin.New()
		router.POST("/auth/login", controller.Login)

		reqBody := []byte(`{"email":"user@example.com","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNoContent, resp.Code)
		assert.Empty(t, resp.Body.String())

		setCookie := resp.Header().Get("Set-Cookie")
		if assert.NotEmpty(t, setCookie) {
			assert.Contains(t, setCookie, "access_token=token")
			assert.Contains(t, setCookie, "Max-Age=86400")
			assert.Contains(t, setCookie, "Path=/")
			assert.Contains(t, setCookie, "Domain=localhost")
			assert.Contains(t, setCookie, "HttpOnly")
			assert.Contains(t, setCookie, "SameSite=Lax")
			assert.NotContains(t, setCookie, "Secure")
		}

		usecase.AssertExpectations(t)
	})

	t.Run("validation error", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		controller := newTestAuthController(usecase, newTestLogger())
		router := gin.New()
		router.POST("/auth/login", controller.Login)

		reqBody := []byte(`{"email":"","password":""}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Empty(t, resp.Body.String())
		usecase.AssertExpectations(t)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		usecase.
			On("Login", mock.Anything, mock.MatchedBy(func(req domain.LoginRequest) bool { return true })).
			Return("", application.ErrInvalidCredentials).
			Once()

		controller := newTestAuthController(usecase, newTestLogger())
		router := gin.New()
		router.POST("/auth/login", controller.Login)

		reqBody := []byte(`{"email":"user@example.com","password":"wrong"}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusUnauthorized, resp.Code)
		usecase.AssertExpectations(t)
	})

	t.Run("internal error", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		usecase.
			On("Login", mock.Anything, mock.MatchedBy(func(req domain.LoginRequest) bool { return true })).
			Return("", errors.New("unexpected error")).
			Once()

		controller := newTestAuthController(usecase, newTestLogger())
		router := gin.New()
		router.POST("/auth/login", controller.Login)

		reqBody := []byte(`{"email":"user@example.com","password":"password"}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusInternalServerError, resp.Code)
		usecase.AssertExpectations(t)
	})
}
