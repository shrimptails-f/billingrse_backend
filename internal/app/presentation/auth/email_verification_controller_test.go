package auth

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"business/internal/auth/application"
	"business/internal/auth/domain"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestVerifyEmailController(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		now := time.Now()
		user := domain.User{
			ID:              1,
			Name:            domain.UserName("Test User"),
			Email:           domain.EmailAddress("test@example.com"),
			EmailVerifiedAt: &now,
		}
		usecase.On("VerifyEmail", mock.Anything, mock.MatchedBy(func(req domain.VerifyEmailRequest) bool {
			return req.Token == "valid-token-123"
		})).Return(user, nil).Once()

		controller := newTestController(usecase, newTestLogger())
		router := gin.New()
		router.GET("/auth/email/verify", controller.VerifyEmail)

		req := httptest.NewRequest(http.MethodGet, "/auth/email/verify?token=valid-token-123", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Body.String(), "メールアドレスの認証が完了しました")
		assert.Contains(t, resp.Body.String(), "test@example.com")
		assert.Contains(t, resp.Body.String(), `"email_verified":true`)
		usecase.AssertExpectations(t)
	})

	t.Run("missing token", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		controller := newTestController(usecase, newTestLogger())
		router := gin.New()
		router.GET("/auth/email/verify", controller.VerifyEmail)

		req := httptest.NewRequest(http.MethodGet, "/auth/email/verify", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "missing_token")
		assert.Contains(t, resp.Body.String(), "トークンが指定されていません")
		usecase.AssertExpectations(t)
	})

	t.Run("invalid token", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		usecase.On("VerifyEmail", mock.Anything, mock.Anything).Return(domain.User{}, application.ErrInvalidToken).Once()

		controller := newTestController(usecase, newTestLogger())
		router := gin.New()
		router.GET("/auth/email/verify", controller.VerifyEmail)

		req := httptest.NewRequest(http.MethodGet, "/auth/email/verify?token=invalid-token", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "invalid_token")
		assert.Contains(t, resp.Body.String(), "無効なトークンです")
		usecase.AssertExpectations(t)
	})

	t.Run("token expired", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		usecase.On("VerifyEmail", mock.Anything, mock.Anything).Return(domain.User{}, application.ErrTokenExpired).Once()

		controller := newTestController(usecase, newTestLogger())
		router := gin.New()
		router.GET("/auth/email/verify", controller.VerifyEmail)

		req := httptest.NewRequest(http.MethodGet, "/auth/email/verify?token=expired-token", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "token_expired")
		assert.Contains(t, resp.Body.String(), "有効期限が切れています")
		usecase.AssertExpectations(t)
	})

	t.Run("token already used", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		usecase.On("VerifyEmail", mock.Anything, mock.Anything).Return(domain.User{}, application.ErrTokenAlreadyUsed).Once()

		controller := newTestController(usecase, newTestLogger())
		router := gin.New()
		router.GET("/auth/email/verify", controller.VerifyEmail)

		req := httptest.NewRequest(http.MethodGet, "/auth/email/verify?token=used-token", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "token_already_used")
		assert.Contains(t, resp.Body.String(), "既に使用済みです")
		usecase.AssertExpectations(t)
	})
}

func TestResendVerificationEmailController(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		usecase.On("ResendVerificationEmail", mock.Anything, mock.MatchedBy(func(req domain.ResendVerificationRequest) bool {
			return req.Email == "test@example.com" && req.Password == "password123"
		})).Return(nil).Once()

		controller := newTestController(usecase, newTestLogger())
		router := gin.New()
		router.POST("/auth/email/resend", controller.ResendVerificationEmail)

		reqBody := []byte(`{"email":"test@example.com","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/email/resend", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Body.String(), "確認メールを再送信しました")
		usecase.AssertExpectations(t)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		usecase.On("ResendVerificationEmail", mock.Anything, mock.Anything).Return(application.ErrInvalidCredentials).Once()

		controller := newTestController(usecase, newTestLogger())
		router := gin.New()
		router.POST("/auth/email/resend", controller.ResendVerificationEmail)

		reqBody := []byte(`{"email":"test@example.com","password":"wrong"}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/email/resend", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusUnauthorized, resp.Code)
		assert.Contains(t, resp.Body.String(), "invalid_credentials")
		usecase.AssertExpectations(t)
	})

	t.Run("already verified", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		usecase.On("ResendVerificationEmail", mock.Anything, mock.Anything).Return(application.ErrAlreadyVerified).Once()

		controller := newTestController(usecase, newTestLogger())
		router := gin.New()
		router.POST("/auth/email/resend", controller.ResendVerificationEmail)

		reqBody := []byte(`{"email":"verified@example.com","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/email/resend", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "already_verified")
		usecase.AssertExpectations(t)
	})

	t.Run("rate limit exceeded", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		usecase.On("ResendVerificationEmail", mock.Anything, mock.Anything).Return(application.ErrRateLimitExceeded).Once()

		controller := newTestController(usecase, newTestLogger())
		router := gin.New()
		router.POST("/auth/email/resend", controller.ResendVerificationEmail)

		reqBody := []byte(`{"email":"test@example.com","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/email/resend", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusTooManyRequests, resp.Code)
		assert.Contains(t, resp.Body.String(), "rate_limit_exceeded")
		usecase.AssertExpectations(t)
	})

	t.Run("mail send failed", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		usecase.On("ResendVerificationEmail", mock.Anything, mock.Anything).Return(application.ErrMailSendFailed).Once()

		controller := newTestController(usecase, newTestLogger())
		router := gin.New()
		router.POST("/auth/email/resend", controller.ResendVerificationEmail)

		reqBody := []byte(`{"email":"test@example.com","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/email/resend", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusInternalServerError, resp.Code)
		assert.Contains(t, resp.Body.String(), "mail_send_failed")
		usecase.AssertExpectations(t)
	})
}
