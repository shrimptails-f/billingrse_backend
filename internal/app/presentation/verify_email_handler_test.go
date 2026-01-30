package presentation

import (
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

func TestVerifyEmailHandler(t *testing.T) {
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

		controller := newTestAuthController(usecase, newTestLogger())
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
		controller := newTestAuthController(usecase, newTestLogger())
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

		controller := newTestAuthController(usecase, newTestLogger())
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

		controller := newTestAuthController(usecase, newTestLogger())
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

		controller := newTestAuthController(usecase, newTestLogger())
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
