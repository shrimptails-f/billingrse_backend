package middleware

import (
	"business/internal/auth/domain"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type stubOsWrapper struct {
	env map[string]string
}

func (s *stubOsWrapper) ReadFile(path string) (string, error) {
	return "", errors.New("not implemented")
}

func (s *stubOsWrapper) GetEnv(key string) (string, error) {
	if s.env == nil {
		return "", fmt.Errorf("environment variable %s not set", key)
	}
	if value, ok := s.env[key]; ok && value != "" {
		return value, nil
	}
	return "", fmt.Errorf("environment variable %s not set", key)
}

func newStubOsWrapper(secret string) *stubOsWrapper {
	return &stubOsWrapper{
		env: map[string]string{
			"JWT_SECRET_KEY": secret,
		},
	}
}

type mockUserProvider struct {
	mock.Mock
}

func (m *mockUserProvider) GetUserByID(ctx context.Context, id uint) (domain.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.User), args.Error(1)
}

func generateToken(secret string, userID uint, expiresAt time.Time) string {
	claims := &domain.AuthClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, _ := token.SignedString([]byte(secret))
	return signedToken
}

func TestAuthMiddleware_Success(t *testing.T) {

	secret := "test-secret"
	osw := newStubOsWrapper(secret)
	users := new(mockUserProvider)
	users.On("GetUserByID", mock.Anything, uint(123)).Return(domain.User{
		ID:            123,
		EmailVerified: true,
	}, nil)

	middleware := NewAuthMiddleware(osw, users)

	router := gin.New()
	router.Use(middleware.Authenticate())
	router.GET("/protected", func(c *gin.Context) {
		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "userID not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	token := generateToken(secret, 123, time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "   " + token + "   "}) // should trim whitespace
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(t, `{"user_id":123}`, resp.Body.String())
	users.AssertExpectations(t)
}

func TestAuthMiddleware_SuccessWithCookie(t *testing.T) {

	secret := "test-secret"
	osw := newStubOsWrapper(secret)
	users := new(mockUserProvider)
	users.On("GetUserByID", mock.Anything, uint(123)).Return(domain.User{
		ID:            123,
		EmailVerified: true,
	}, nil)

	middleware := NewAuthMiddleware(osw, users)

	router := gin.New()
	router.Use(middleware.Authenticate())
	router.GET("/protected", func(c *gin.Context) {
		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "userID not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	token := generateToken(secret, 123, time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(t, `{"user_id":123}`, resp.Body.String())
	users.AssertExpectations(t)
}

func TestAuthMiddleware_MissingTokenNoCredentials(t *testing.T) {

	osw := newStubOsWrapper("test-secret")
	users := new(mockUserProvider)
	middleware := NewAuthMiddleware(osw, users)

	router := gin.New()
	router.Use(middleware.Authenticate())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.JSONEq(t, `{"error":"missing token"}`, resp.Body.String())
}

func TestAuthMiddleware_IgnoresAuthorizationHeader(t *testing.T) {

	secret := "test-secret"
	osw := newStubOsWrapper(secret)
	users := new(mockUserProvider)
	middleware := NewAuthMiddleware(osw, users)

	router := gin.New()
	router.Use(middleware.Authenticate())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	token := generateToken(secret, 123, time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token) // Authorization header should be ignored
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.JSONEq(t, `{"error":"missing token"}`, resp.Body.String())
}

func TestAuthMiddleware_EmptyCookieValue(t *testing.T) {

	osw := newStubOsWrapper("test-secret")
	users := new(mockUserProvider)
	middleware := NewAuthMiddleware(osw, users)

	router := gin.New()
	router.Use(middleware.Authenticate())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "   "})
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.JSONEq(t, `{"error":"missing token"}`, resp.Body.String())
}

func TestAuthMiddleware_SecretKeyNotSet(t *testing.T) {

	osw := &stubOsWrapper{env: map[string]string{}} // No JWT_SECRET_KEY
	users := new(mockUserProvider)
	middleware := NewAuthMiddleware(osw, users)

	router := gin.New()
	router.Use(middleware.Authenticate())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "some-token"})
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.JSONEq(t, `{"error":"internal server error"}`, resp.Body.String())
}

func TestAuthMiddleware_InvalidSigningMethod(t *testing.T) {

	secret := "test-secret"
	osw := newStubOsWrapper(secret)
	users := new(mockUserProvider)
	middleware := NewAuthMiddleware(osw, users)

	router := gin.New()
	router.Use(middleware.Authenticate())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	claims := &domain.AuthClaims{
		UserID: 123,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	signedToken, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: signedToken})
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.JSONEq(t, `{"error":"invalid token"}`, resp.Body.String())
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {

	secret := "test-secret"
	osw := newStubOsWrapper(secret)
	users := new(mockUserProvider)
	middleware := NewAuthMiddleware(osw, users)

	router := gin.New()
	router.Use(middleware.Authenticate())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	token := generateToken(secret, 123, time.Now().Add(-time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.JSONEq(t, `{"error":"invalid token"}`, resp.Body.String())
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {

	osw := newStubOsWrapper("test-secret")
	users := new(mockUserProvider)
	middleware := NewAuthMiddleware(osw, users)

	router := gin.New()
	router.Use(middleware.Authenticate())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "invalid.token.here"})
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.JSONEq(t, `{"error":"invalid token"}`, resp.Body.String())
}

func TestAuthMiddleware_EmailVerificationRequired(t *testing.T) {

	secret := "test-secret"
	osw := newStubOsWrapper(secret)
	users := new(mockUserProvider)
	users.On("GetUserByID", mock.Anything, uint(123)).Return(domain.User{
		ID:            123,
		EmailVerified: false, // Not verified
	}, nil)

	middleware := NewAuthMiddleware(osw, users)

	router := gin.New()
	router.Use(middleware.Authenticate())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	token := generateToken(secret, 123, time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.Contains(t, resp.Body.String(), "email_verification_required")
	assert.Contains(t, resp.Body.String(), "メールアドレスの認証が完了していません")
	users.AssertExpectations(t)
}

func TestAuthMiddleware_SkipEmailVerificationForAuthPaths(t *testing.T) {

	secret := "test-secret"
	osw := newStubOsWrapper(secret)
	users := new(mockUserProvider)

	middleware := NewAuthMiddleware(osw, users)

	testCases := []string{
		"/auth/register",
		"/auth/login",
		"/auth/email/verify",
		"/auth/email/resend",
	}

	for _, path := range testCases {
		t.Run(path, func(t *testing.T) {
			router := gin.New()
			router.Use(middleware.Authenticate())
			router.POST(path, func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"success": true})
			})

			token := generateToken(secret, 123, time.Now().Add(time.Hour))
			req := httptest.NewRequest(http.MethodPost, path, nil)
			req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			// Should not check email verification for these paths
			assert.Equal(t, http.StatusOK, resp.Code)
		})
	}

	// users.GetUserByID should not have been called for any of these paths
	users.AssertNotCalled(t, "GetUserByID", mock.Anything, mock.Anything)
}

func TestAuthMiddleware_UserNotFoundDuringEmailVerificationCheck(t *testing.T) {

	secret := "test-secret"
	osw := newStubOsWrapper(secret)
	users := new(mockUserProvider)
	users.On("GetUserByID", mock.Anything, uint(123)).Return(domain.User{}, errors.New("user not found"))

	middleware := NewAuthMiddleware(osw, users)

	router := gin.New()
	router.Use(middleware.Authenticate())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	token := generateToken(secret, 123, time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.JSONEq(t, `{"error":"user not found"}`, resp.Body.String())
	users.AssertExpectations(t)
}
