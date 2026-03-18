package auth

import (
	"bytes"
	"errors"
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

	t.Run("POST success", func(t *testing.T) {
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
		router.POST("/api/v1/auth/email/verify", controller.VerifyEmail)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/email/verify", bytes.NewBuffer([]byte(`{"token":"valid-token-123"}`)))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Body.String(), "メールアドレスの認証が完了しました")
		assert.Contains(t, resp.Body.String(), "test@example.com")
		assert.Contains(t, resp.Body.String(), `"email_verified":true`)
		usecase.AssertExpectations(t)
	})

	t.Run("POST missing token", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		controller := newTestController(usecase, newTestLogger())
		router := gin.New()
		router.POST("/api/v1/auth/email/verify", controller.VerifyEmail)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/email/verify", bytes.NewBuffer([]byte(`{}`)))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.JSONEq(t, `{"error":{"code":"missing_token","message":"トークンが指定されていません。"}}`, resp.Body.String())
		usecase.AssertExpectations(t)
	})

	t.Run("POST invalid token", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		usecase.On("VerifyEmail", mock.Anything, mock.Anything).Return(domain.User{}, application.ErrInvalidToken).Once()

		controller := newTestController(usecase, newTestLogger())
		router := gin.New()
		router.POST("/api/v1/auth/email/verify", controller.VerifyEmail)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/email/verify", bytes.NewBuffer([]byte(`{"token":"invalid-token"}`)))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.JSONEq(t, `{"error":{"code":"invalid_token","message":"無効なトークンです。"}}`, resp.Body.String())
		usecase.AssertExpectations(t)
	})

	t.Run("POST token expired", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		usecase.On("VerifyEmail", mock.Anything, mock.Anything).Return(domain.User{}, application.ErrTokenExpired).Once()

		controller := newTestController(usecase, newTestLogger())
		router := gin.New()
		router.POST("/api/v1/auth/email/verify", controller.VerifyEmail)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/email/verify", bytes.NewBuffer([]byte(`{"token":"expired-token"}`)))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusConflict, resp.Code)
		assert.JSONEq(t, `{"error":{"code":"token_expired","message":"トークンの有効期限が切れています。再送信をお試しください。"}}`, resp.Body.String())
		usecase.AssertExpectations(t)
	})

	t.Run("POST token already used", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		usecase.On("VerifyEmail", mock.Anything, mock.Anything).Return(domain.User{}, application.ErrTokenAlreadyUsed).Once()

		controller := newTestController(usecase, newTestLogger())
		router := gin.New()
		router.POST("/api/v1/auth/email/verify", controller.VerifyEmail)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/email/verify", bytes.NewBuffer([]byte(`{"token":"used-token"}`)))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusConflict, resp.Code)
		assert.JSONEq(t, `{"error":{"code":"token_already_used","message":"このトークンは既に使用済みです。"}}`, resp.Body.String())
		usecase.AssertExpectations(t)
	})

	t.Run("POST internal error", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		usecase.On("VerifyEmail", mock.Anything, mock.Anything).Return(domain.User{}, errors.New("db error")).Once()

		controller := newTestController(usecase, newTestLogger())
		router := gin.New()
		router.POST("/api/v1/auth/email/verify", controller.VerifyEmail)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/email/verify", bytes.NewBuffer([]byte(`{"token":"internal-error-token"}`)))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusInternalServerError, resp.Code)
		assert.JSONEq(t, `{"error":{"code":"internal_server_error","message":"サーバー内部でエラーが発生しました。しばらくしてから再度お試しください。"}}`, resp.Body.String())
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
		router.POST("/api/v1/auth/email/resend", controller.ResendVerificationEmail)

		reqBody := []byte(`{"email":"test@example.com","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/email/resend", bytes.NewBuffer(reqBody))
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
		router.POST("/api/v1/auth/email/resend", controller.ResendVerificationEmail)

		reqBody := []byte(`{"email":"test@example.com","password":"wrong"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/email/resend", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusUnauthorized, resp.Code)
		assert.JSONEq(t, `{"error":{"code":"invalid_credentials","message":"メールアドレスまたはパスワードが正しくありません。"}}`, resp.Body.String())
		usecase.AssertExpectations(t)
	})

	t.Run("already verified", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		usecase.On("ResendVerificationEmail", mock.Anything, mock.Anything).Return(application.ErrAlreadyVerified).Once()

		controller := newTestController(usecase, newTestLogger())
		router := gin.New()
		router.POST("/api/v1/auth/email/resend", controller.ResendVerificationEmail)

		reqBody := []byte(`{"email":"verified@example.com","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/email/resend", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusForbidden, resp.Code)
		assert.JSONEq(t, `{"error":{"code":"already_verified","message":"このメールアドレスは既に認証済みです。"}}`, resp.Body.String())
		usecase.AssertExpectations(t)
	})

	t.Run("rate limit exceeded", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		usecase.On("ResendVerificationEmail", mock.Anything, mock.Anything).Return(application.ErrRateLimitExceeded).Once()

		controller := newTestController(usecase, newTestLogger())
		router := gin.New()
		router.POST("/api/v1/auth/email/resend", controller.ResendVerificationEmail)

		reqBody := []byte(`{"email":"test@example.com","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/email/resend", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusTooManyRequests, resp.Code)
		assert.JSONEq(t, `{"error":{"code":"rate_limit_exceeded","message":"再送信の回数制限に達しました。15分後に再度お試しください。"}}`, resp.Body.String())
		usecase.AssertExpectations(t)
	})

	t.Run("mail send failed", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		usecase.On("ResendVerificationEmail", mock.Anything, mock.Anything).Return(application.ErrMailSendFailed).Once()

		controller := newTestController(usecase, newTestLogger())
		router := gin.New()
		router.POST("/api/v1/auth/email/resend", controller.ResendVerificationEmail)

		reqBody := []byte(`{"email":"test@example.com","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/email/resend", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusServiceUnavailable, resp.Code)
		assert.JSONEq(t, `{"error":{"code":"mail_send_failed","message":"メール送信に失敗しました。しばらくしてから再度お試しください。"}}`, resp.Body.String())
		usecase.AssertExpectations(t)
	})

	t.Run("internal error", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		usecase.On("ResendVerificationEmail", mock.Anything, mock.Anything).Return(errors.New("db error")).Once()

		controller := newTestController(usecase, newTestLogger())
		router := gin.New()
		router.POST("/api/v1/auth/email/resend", controller.ResendVerificationEmail)

		reqBody := []byte(`{"email":"test@example.com","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/email/resend", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusInternalServerError, resp.Code)
		assert.JSONEq(t, `{"error":{"code":"internal_server_error","message":"サーバー内部でエラーが発生しました。しばらくしてから再度お試しください。"}}`, resp.Body.String())
		usecase.AssertExpectations(t)
	})
}
