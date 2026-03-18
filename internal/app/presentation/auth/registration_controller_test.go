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

func TestRegistrationController(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		user := domain.User{
			ID:    1,
			Name:  domain.UserName("New User"),
			Email: domain.EmailAddress("new@example.com"),
		}
		usecase.On("Register", mock.Anything, mock.MatchedBy(func(req domain.RegisterRequest) bool {
			return req.Email == "new@example.com" && req.Name == "New User" && req.Password == "password123"
		})).Return(user, nil).Once()

		controller := newTestController(usecase, newTestLogger())
		router := gin.New()
		router.POST("/api/v1/auth/register", controller.Register)

		reqBody := []byte(`{"email":"new@example.com","name":"New User","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)
		assert.Contains(t, resp.Body.String(), "登録が完了しました")
		assert.Contains(t, resp.Body.String(), "new@example.com")
		assert.Contains(t, resp.Body.String(), `"email_verified":false`)
		usecase.AssertExpectations(t)
	})

	t.Run("validation error", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		controller := newTestController(usecase, newTestLogger())
		router := gin.New()
		router.POST("/api/v1/auth/register", controller.Register)

		reqBody := []byte(`{"email":"invalid","name":"","password":""}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.JSONEq(t, `{"error":{"code":"invalid_request","message":"入力値が不正です。"}}`, resp.Body.String())
		usecase.AssertExpectations(t)
	})

	t.Run("email conflict", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		usecase.On("Register", mock.Anything, mock.Anything).Return(domain.User{}, application.ErrEmailAlreadyExists).Once()

		controller := newTestController(usecase, newTestLogger())
		router := gin.New()
		router.POST("/api/v1/auth/register", controller.Register)

		reqBody := []byte(`{"email":"dup@example.com","name":"Dup","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusConflict, resp.Code)
		assert.JSONEq(t, `{"error":{"code":"email_already_exists","message":"このメールアドレスは既に登録されています。"}}`, resp.Body.String())
		usecase.AssertExpectations(t)
	})

	t.Run("mail send failed", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		usecase.On("Register", mock.Anything, mock.Anything).Return(domain.User{}, application.ErrMailSendFailed).Once()

		controller := newTestController(usecase, newTestLogger())
		router := gin.New()
		router.POST("/api/v1/auth/register", controller.Register)

		reqBody := []byte(`{"email":"user@example.com","name":"User","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(reqBody))
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
		usecase.On("Register", mock.Anything, mock.Anything).Return(domain.User{}, errors.New("db error")).Once()

		controller := newTestController(usecase, newTestLogger())
		router := gin.New()
		router.POST("/api/v1/auth/register", controller.Register)

		reqBody := []byte(`{"email":"user@example.com","name":"User","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusInternalServerError, resp.Code)
		assert.JSONEq(t, `{"error":{"code":"internal_server_error","message":"サーバー内部でエラーが発生しました。しばらくしてから再度お試しください。"}}`, resp.Body.String())
		usecase.AssertExpectations(t)
	})
}
