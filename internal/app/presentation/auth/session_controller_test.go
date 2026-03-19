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
		usecase.On("LoginTokens", mock.Anything, mock.MatchedBy(func(req domain.LoginRequest) bool {
			return req.Email == "user@example.com" && req.Password == "password123"
		})).Return(domain.AuthTokens{
			AccessToken:           "access-token",
			TokenType:             "Bearer",
			ExpiresIn:             900,
			RefreshToken:          "refresh-token",
			RefreshTokenExpiresIn: 2592000,
		}, nil).Once()

		controller := newTestControllerWithVars(usecase, log, map[string]string{"APP": "local"})
		router := gin.New()
		router.POST("/api/v1/auth/login", controller.Login)

		reqBody := []byte(`{"email":"user@example.com","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.JSONEq(t, `{"access_token":"access-token","token_type":"Bearer","expires_in":900}`, resp.Body.String())

		setCookie := resp.Header().Get("Set-Cookie")
		if assert.NotEmpty(t, setCookie) {
			assert.Contains(t, setCookie, "refresh_token=refresh-token")
			assert.Contains(t, setCookie, "Max-Age=2592000")
			assert.Contains(t, setCookie, "Path=/api/v1/auth")
			assert.Contains(t, setCookie, "Domain=localhost")
			assert.Contains(t, setCookie, "HttpOnly")
			assert.Contains(t, setCookie, "SameSite=Strict")
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
			On("LoginTokens", mock.Anything, mock.MatchedBy(func(req domain.LoginRequest) bool { return true })).
			Return(domain.AuthTokens{}, application.ErrInvalidCredentials).
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
			On("LoginTokens", mock.Anything, mock.MatchedBy(func(req domain.LoginRequest) bool { return true })).
			Return(domain.AuthTokens{}, errors.New("unexpected error")).
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

func TestRefreshController(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		usecase := new(mockAuthUseCase)
		log := newCapturingTestLogger()
		usecase.On("Refresh", mock.Anything, mock.MatchedBy(func(req domain.RefreshRequest) bool {
			return req.RefreshToken == "refresh-token"
		})).Return(domain.AuthTokens{
			AccessToken:           "new-access-token",
			TokenType:             "Bearer",
			ExpiresIn:             900,
			RefreshToken:          "new-refresh-token",
			RefreshTokenExpiresIn: 2592000,
		}, nil).Once()

		controller := newTestControllerWithVars(usecase, log, map[string]string{"APP": "local"})
		router := gin.New()
		router.POST("/api/v1/auth/refresh", controller.Refresh)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
		req.AddCookie(&http.Cookie{Name: refreshTokenCookieName, Value: "refresh-token"})
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.JSONEq(t, `{"access_token":"new-access-token","token_type":"Bearer","expires_in":900}`, resp.Body.String())
		setCookie := resp.Header().Get("Set-Cookie")
		if assert.NotEmpty(t, setCookie) {
			assert.Contains(t, setCookie, "refresh_token=new-refresh-token")
			assert.Contains(t, setCookie, "Max-Age=2592000")
			assert.Contains(t, setCookie, "SameSite=Strict")
		}
		usecase.AssertExpectations(t)
		if assert.Len(t, log.Entries, 1) {
			assert.Equal(t, "info", log.Entries[0].Level)
			assert.Equal(t, "refresh_succeeded", log.Entries[0].Message)
		}
	})

	t.Run("missing refresh token", func(t *testing.T) {
		usecase := new(mockAuthUseCase)
		controller := newTestController(usecase, newTestLogger())
		router := gin.New()
		router.POST("/api/v1/auth/refresh", controller.Refresh)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusUnauthorized, resp.Code)
		assert.JSONEq(t, `{"error":{"code":"missing_refresh_token","message":"リフレッシュトークンがありません。"}}`, resp.Body.String())
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		usecase := new(mockAuthUseCase)
		usecase.On("Refresh", mock.Anything, mock.MatchedBy(func(req domain.RefreshRequest) bool {
			return req.RefreshToken == "bad-token"
		})).Return(domain.AuthTokens{}, application.ErrRefreshTokenInvalid).Once()

		controller := newTestController(usecase, newTestLogger())
		router := gin.New()
		router.POST("/api/v1/auth/refresh", controller.Refresh)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
		req.AddCookie(&http.Cookie{Name: refreshTokenCookieName, Value: "bad-token"})
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusUnauthorized, resp.Code)
		assert.JSONEq(t, `{"error":{"code":"invalid_refresh_token","message":"リフレッシュトークンが無効です。"}}`, resp.Body.String())
		usecase.AssertExpectations(t)
	})
}

func TestLogoutController(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		usecase := new(mockAuthUseCase)
		usecase.On("Logout", mock.Anything, mock.MatchedBy(func(req domain.LogoutRequest) bool {
			return req.RefreshToken == "refresh-token"
		})).Return(nil).Once()

		controller := newTestControllerWithVars(usecase, newTestLogger(), map[string]string{"APP": "production"})
		router := gin.New()
		router.POST("/api/v1/auth/logout", controller.Logout)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		req.AddCookie(&http.Cookie{Name: refreshTokenCookieName, Value: "refresh-token"})
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNoContent, resp.Code)
		setCookie := resp.Header().Get("Set-Cookie")
		if assert.NotEmpty(t, setCookie) {
			assert.Contains(t, setCookie, "refresh_token=")
			assert.Contains(t, setCookie, "Max-Age=0")
			assert.Contains(t, setCookie, "SameSite=Strict")
			assert.Contains(t, setCookie, "Secure")
		}
		usecase.AssertExpectations(t)
	})

	t.Run("missing refresh token still clears cookie", func(t *testing.T) {
		controller := newTestController(new(mockAuthUseCase), newTestLogger())
		router := gin.New()
		router.POST("/api/v1/auth/logout", controller.Logout)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNoContent, resp.Code)
	})
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
