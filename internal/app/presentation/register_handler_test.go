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

func TestRegisterHandler(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		user := domain.User{
			ID:            1,
			Name:          domain.UserName("New User"),
			Email:         domain.EmailAddress("new@example.com"),
			EmailVerified: false,
		}
		usecase.On("Register", mock.Anything, mock.MatchedBy(func(req domain.RegisterRequest) bool {
			return req.Email == "new@example.com" && req.Name == "New User" && req.Password == "password123"
		})).Return(user, nil).Once()

		controller := newTestAuthController(usecase, newTestLogger())
		router := gin.New()
		router.POST("/auth/register", controller.Register)

		reqBody := []byte(`{"email":"new@example.com","name":"New User","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(reqBody))
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
		controller := newTestAuthController(usecase, newTestLogger())
		router := gin.New()
		router.POST("/auth/register", controller.Register)

		reqBody := []byte(`{"email":"invalid","name":"","password":""}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		usecase.AssertExpectations(t)
	})

	t.Run("email conflict", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		usecase.On("Register", mock.Anything, mock.Anything).Return(domain.User{}, application.ErrEmailAlreadyExists).Once()

		controller := newTestAuthController(usecase, newTestLogger())
		router := gin.New()
		router.POST("/auth/register", controller.Register)

		reqBody := []byte(`{"email":"dup@example.com","name":"Dup","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "email_already_exists")
		usecase.AssertExpectations(t)
	})

	t.Run("mail send failed", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		usecase.On("Register", mock.Anything, mock.Anything).Return(domain.User{}, application.ErrMailSendFailed).Once()

		controller := newTestAuthController(usecase, newTestLogger())
		router := gin.New()
		router.POST("/auth/register", controller.Register)

		reqBody := []byte(`{"email":"user@example.com","name":"User","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusInternalServerError, resp.Code)
		assert.Contains(t, resp.Body.String(), "mail_send_failed")
		usecase.AssertExpectations(t)
	})

	t.Run("internal error", func(t *testing.T) {
		t.Parallel()
		usecase := new(mockAuthUseCase)
		usecase.On("Register", mock.Anything, mock.Anything).Return(domain.User{}, errors.New("db error")).Once()

		controller := newTestAuthController(usecase, newTestLogger())
		router := gin.New()
		router.POST("/auth/register", controller.Register)

		reqBody := []byte(`{"email":"user@example.com","name":"User","password":"password123"}`)
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusInternalServerError, resp.Code)
		usecase.AssertExpectations(t)
	})
}
