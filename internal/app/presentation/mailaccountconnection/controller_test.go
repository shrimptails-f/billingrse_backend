package mailaccountconnection

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"business/internal/emailcredential/application"
	"business/internal/emailcredential/domain"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setUserID(c *gin.Context, uid uint) {
	c.Set("userID", uid)
}

func authorizeRouter(ctrl *Controller) *gin.Engine {
	r := gin.New()
	r.POST("/authorize", func(c *gin.Context) { setUserID(c, 1) }, ctrl.Authorize)
	return r
}

func callbackRouter(ctrl *Controller) *gin.Engine {
	r := gin.New()
	r.POST("/callback", func(c *gin.Context) { setUserID(c, 1) }, ctrl.Callback)
	return r
}

func listRouter(ctrl *Controller) *gin.Engine {
	r := gin.New()
	r.GET("/connections", func(c *gin.Context) { setUserID(c, 1) }, ctrl.List)
	return r
}

func TestAuthorize_200(t *testing.T) {
	t.Parallel()
	uc := new(mockUseCase)
	uc.On("Authorize", mock.Anything, uint(1)).Return(application.AuthorizeResult{
		AuthorizationURL: "https://accounts.google.com/o/oauth2/auth?state=abc",
		ExpiresAt:        fixedExpiresAt(),
	}, nil).Once()

	ctrl := newTestController(uc)
	r := authorizeRouter(ctrl)

	req := httptest.NewRequest(http.MethodPost, "/authorize", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "authorization_url")
	assert.Contains(t, resp.Body.String(), "expires_at")
	uc.AssertExpectations(t)
}

func TestAuthorize_401_no_user(t *testing.T) {
	t.Parallel()
	uc := new(mockUseCase)
	ctrl := newTestController(uc)

	r := gin.New()
	r.POST("/authorize", ctrl.Authorize) // no userID set

	req := httptest.NewRequest(http.MethodPost, "/authorize", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.Contains(t, resp.Body.String(), "unauthorized")
}

func TestAuthorize_500_internal(t *testing.T) {
	t.Parallel()
	uc := new(mockUseCase)
	uc.On("Authorize", mock.Anything, uint(1)).Return(application.AuthorizeResult{}, errors.New("config fail")).Once()

	ctrl := newTestController(uc)
	r := authorizeRouter(ctrl)

	req := httptest.NewRequest(http.MethodPost, "/authorize", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	uc.AssertExpectations(t)
}

func TestCallback_200(t *testing.T) {
	t.Parallel()
	uc := new(mockUseCase)
	uc.On("Callback", mock.Anything, uint(1), "auth-code", "state-value").Return(nil).Once()

	ctrl := newTestController(uc)
	r := callbackRouter(ctrl)

	body := []byte(`{"code":"auth-code","state":"state-value"}`)
	req := httptest.NewRequest(http.MethodPost, "/callback", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "Gmail連携が完了しました。")
	uc.AssertExpectations(t)
}

func TestCallback_400_invalid_request(t *testing.T) {
	t.Parallel()
	uc := new(mockUseCase)
	ctrl := newTestController(uc)
	r := callbackRouter(ctrl)

	body := []byte(`{"code":""}`)
	req := httptest.NewRequest(http.MethodPost, "/callback", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "invalid_request")
}

func TestCallback_401_no_user(t *testing.T) {
	t.Parallel()
	uc := new(mockUseCase)
	ctrl := newTestController(uc)

	r := gin.New()
	r.POST("/callback", ctrl.Callback) // no userID set

	body := []byte(`{"code":"code","state":"state"}`)
	req := httptest.NewRequest(http.MethodPost, "/callback", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestCallback_409_state_mismatch(t *testing.T) {
	t.Parallel()
	uc := new(mockUseCase)
	uc.On("Callback", mock.Anything, uint(1), "code", "bad-state").Return(domain.ErrOAuthStateMismatch).Once()

	ctrl := newTestController(uc)
	r := callbackRouter(ctrl)

	body := []byte(`{"code":"code","state":"bad-state"}`)
	req := httptest.NewRequest(http.MethodPost, "/callback", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusConflict, resp.Code)
	assert.Contains(t, resp.Body.String(), "oauth_state_mismatch")
	uc.AssertExpectations(t)
}

func TestCallback_409_state_expired(t *testing.T) {
	t.Parallel()
	uc := new(mockUseCase)
	uc.On("Callback", mock.Anything, uint(1), "code", "expired-state").Return(domain.ErrOAuthStateExpired).Once()

	ctrl := newTestController(uc)
	r := callbackRouter(ctrl)

	body := []byte(`{"code":"code","state":"expired-state"}`)
	req := httptest.NewRequest(http.MethodPost, "/callback", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusConflict, resp.Code)
	assert.Contains(t, resp.Body.String(), "oauth_state_expired")
	uc.AssertExpectations(t)
}

func TestCallback_503_exchange_failed(t *testing.T) {
	t.Parallel()
	uc := new(mockUseCase)
	uc.On("Callback", mock.Anything, uint(1), "code", "state").Return(domain.ErrOAuthExchangeFailed).Once()

	ctrl := newTestController(uc)
	r := callbackRouter(ctrl)

	body := []byte(`{"code":"code","state":"state"}`)
	req := httptest.NewRequest(http.MethodPost, "/callback", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusServiceUnavailable, resp.Code)
	assert.Contains(t, resp.Body.String(), "gmail_oauth_exchange_failed")
	uc.AssertExpectations(t)
}

func TestList_200(t *testing.T) {
	t.Parallel()
	createdAt := time.Date(2026, 3, 19, 12, 34, 56, 0, time.UTC)
	updatedAt := time.Date(2026, 3, 19, 12, 40, 12, 0, time.UTC)

	uc := new(mockUseCase)
	uc.On("ListConnections", mock.Anything, uint(1)).Return([]domain.ConnectionView{
		{
			ID:                12,
			Provider:          "gmail",
			AccountIdentifier: "user@gmail.com",
			CreatedAt:         createdAt,
			UpdatedAt:         updatedAt,
		},
	}, nil).Once()

	ctrl := newTestController(uc)
	r := listRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/connections", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(t, `{
		"items": [
			{
				"id": 12,
				"provider": "gmail",
				"account_identifier": "user@gmail.com",
				"created_at": "2026-03-19T12:34:56Z",
				"updated_at": "2026-03-19T12:40:12Z"
			}
		]
	}`, resp.Body.String())
	uc.AssertExpectations(t)
}

func TestList_401_no_user(t *testing.T) {
	t.Parallel()
	uc := new(mockUseCase)
	ctrl := newTestController(uc)

	r := gin.New()
	r.GET("/connections", ctrl.List)

	req := httptest.NewRequest(http.MethodGet, "/connections", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.Contains(t, resp.Body.String(), "unauthorized")
}

func TestList_500_internal(t *testing.T) {
	t.Parallel()
	uc := new(mockUseCase)
	uc.On("ListConnections", mock.Anything, uint(1)).Return(([]domain.ConnectionView)(nil), errors.New("db fail")).Once()

	ctrl := newTestController(uc)
	r := listRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/connections", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Contains(t, resp.Body.String(), "internal_server_error")
	uc.AssertExpectations(t)
}
