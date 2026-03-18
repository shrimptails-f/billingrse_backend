package auth

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

func TestLoginController(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		log := newCapturingTestLogger()
		usecase.On("Login", mock.Anything, mock.MatchedBy(func(req domain.LoginRequest) bool {
			return req.Email == "user@example.com" && req.Password == "password123"
		})).Return("token", nil).Once()

		controller := newTestControllerWithVars(usecase, log, map[string]string{"APP": "local"})
		router := gin.New()
		router.POST("/api/v1/auth/login", controller.Login)

		reqBody := []byte(`{"email":"user@example.com","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(reqBody))
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
		if assert.Len(t, log.Entries, 1) {
			assert.Equal(t, "info", log.Entries[0].Level)
			assert.Equal(t, "login_succeeded", log.Entries[0].Message)
		}
	})

	t.Run("validation error", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		controller := newTestController(usecase, newTestLogger())
		router := gin.New()
		router.POST("/api/v1/auth/login", controller.Login)

		reqBody := []byte(`{"email":"","password":""}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.JSONEq(t, `{"error":{"code":"invalid_request","message":"入力値が不正です。"}}`, resp.Body.String())
		usecase.AssertExpectations(t)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		log := newCapturingTestLogger()
		usecase.
			On("Login", mock.Anything, mock.MatchedBy(func(req domain.LoginRequest) bool { return true })).
			Return("", application.ErrInvalidCredentials).
			Once()

		controller := newTestController(usecase, log)
		router := gin.New()
		router.POST("/api/v1/auth/login", controller.Login)

		reqBody := []byte(`{"email":"user@example.com","password":"wrong"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusUnauthorized, resp.Code)
		assert.JSONEq(t, `{"error":{"code":"invalid_credentials","message":"メールアドレスまたはパスワードが正しくありません。"}}`, resp.Body.String())
		usecase.AssertExpectations(t)
		if assert.Len(t, log.Entries, 1) {
			assert.Equal(t, "info", log.Entries[0].Level)
			assert.Equal(t, "login_failed", log.Entries[0].Message)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		usecase.
			On("Login", mock.Anything, mock.MatchedBy(func(req domain.LoginRequest) bool { return true })).
			Return("", errors.New("unexpected error")).
			Once()

		controller := newTestController(usecase, newTestLogger())
		router := gin.New()
		router.POST("/api/v1/auth/login", controller.Login)

		reqBody := []byte(`{"email":"user@example.com","password":"password"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusInternalServerError, resp.Code)
		assert.JSONEq(t, `{"error":{"code":"internal_server_error","message":"サーバー内部でエラーが発生しました。しばらくしてから再度お試しください。"}}`, resp.Body.String())
		usecase.AssertExpectations(t)
	})
}

func TestLogoutControllerClearsCookie(t *testing.T) {
	t.Parallel()
	controller := newTestController(new(mockAuthUseCase), newTestLogger())
	router := gin.New()
	router.POST("/api/v1/auth/logout", controller.Logout)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
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

func TestLogoutControllerMarksSecureWhenNeeded(t *testing.T) {
	t.Parallel()
	controller := newTestControllerWithVars(new(mockAuthUseCase), newTestLogger(), map[string]string{"APP": "production"})
	router := gin.New()
	router.POST("/api/v1/auth/logout", controller.Logout)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	setCookie := resp.Header().Get("Set-Cookie")
	if assert.NotEmpty(t, setCookie) {
		assert.Contains(t, setCookie, "Secure")
	}
}

func TestCheckController(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		usecase := new(mockAuthUseCase)
		controller := newTestController(usecase, newTestLogger())
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("userID", uint(123))
			c.Next()
		})
		router.GET("/api/v1/auth/check", controller.Check)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/check", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.JSONEq(t, `{"user_id":123}`, resp.Body.String())
	})

	t.Run("userID not set", func(t *testing.T) {
		usecase := new(mockAuthUseCase)
		controller := newTestController(usecase, newTestLogger())
		router := gin.New()
		router.GET("/api/v1/auth/check", controller.Check)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/check", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusInternalServerError, resp.Code)
		assert.JSONEq(t, `{"error":{"code":"internal_server_error","message":"サーバー内部でエラーが発生しました。しばらくしてから再度お試しください。"}}`, resp.Body.String())
	})

	t.Run("userID wrong type", func(t *testing.T) {
		usecase := new(mockAuthUseCase)
		controller := newTestController(usecase, newTestLogger())
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set("userID", "not-a-uint")
			c.Next()
		})
		router.GET("/api/v1/auth/check", controller.Check)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/check", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusInternalServerError, resp.Code)
		assert.JSONEq(t, `{"error":{"code":"internal_server_error","message":"サーバー内部でエラーが発生しました。しばらくしてから再度お試しください。"}}`, resp.Body.String())
	})
}
