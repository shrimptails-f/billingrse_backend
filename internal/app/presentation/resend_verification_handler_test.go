package presentation

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"business/internal/auth/application"
	"business/internal/auth/domain"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestResendVerificationEmailHandler(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		usecase.On("ResendVerificationEmail", mock.Anything, mock.MatchedBy(func(req domain.ResendVerificationRequest) bool {
			return req.Email == "test@example.com" && req.Password == "password123"
		})).Return(nil).Once()

		controller := newTestAuthController(usecase, newTestLogger())
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

		controller := newTestAuthController(usecase, newTestLogger())
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

		controller := newTestAuthController(usecase, newTestLogger())
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

		controller := newTestAuthController(usecase, newTestLogger())
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

		controller := newTestAuthController(usecase, newTestLogger())
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
