package v1_test

import (
	"context"
	"testing"
	"time"

	"business/internal/app/middleware"
	authpresentation "business/internal/app/presentation/auth"
	macpresentation "business/internal/app/presentation/mailaccountconnection"
	v1 "business/internal/app/router"
	"business/internal/auth/domain"
	ecapp "business/internal/emailcredential/application"
	ecdomain "business/internal/emailcredential/domain"
	"business/internal/library/logger"
	mocklibrary "business/test/mock/library"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/dig"
)

type stubAuthUserProvider struct{}

func (s *stubAuthUserProvider) GetUserByID(ctx context.Context, id uint) (domain.User, error) {
	verifiedAt := time.Now()
	return domain.User{
		ID:              id,
		EmailVerifiedAt: &verifiedAt,
	}, nil
}

type stubAuthUseCase struct{}

func (s *stubAuthUseCase) Login(ctx context.Context, req domain.LoginRequest) (string, error) {
	return "dummy-token", nil
}

func (s *stubAuthUseCase) LoginTokens(ctx context.Context, req domain.LoginRequest) (domain.AuthTokens, error) {
	return domain.AuthTokens{
		AccessToken:           "dummy-token",
		TokenType:             "Bearer",
		ExpiresIn:             900,
		RefreshToken:          "dummy-refresh-token",
		RefreshTokenExpiresIn: 2592000,
	}, nil
}

func (s *stubAuthUseCase) Register(ctx context.Context, req domain.RegisterRequest) (domain.User, error) {
	return domain.User{}, nil
}

func (s *stubAuthUseCase) VerifyEmail(ctx context.Context, req domain.VerifyEmailRequest) (domain.User, error) {
	return domain.User{}, nil
}

func (s *stubAuthUseCase) ResendVerificationEmail(ctx context.Context, req domain.ResendVerificationRequest) error {
	return nil
}

func (s *stubAuthUseCase) Refresh(ctx context.Context, req domain.RefreshRequest) (domain.AuthTokens, error) {
	return domain.AuthTokens{
		AccessToken:           "dummy-token",
		TokenType:             "Bearer",
		ExpiresIn:             900,
		RefreshToken:          "dummy-refresh-token",
		RefreshTokenExpiresIn: 2592000,
	}, nil
}

func (s *stubAuthUseCase) Logout(ctx context.Context, req domain.LogoutRequest) error {
	return nil
}

type stubEmailCredentialUsecase struct{}

func (s *stubEmailCredentialUsecase) Authorize(ctx context.Context, userID uint) (ecapp.AuthorizeResult, error) {
	return ecapp.AuthorizeResult{
		AuthorizationURL: "https://accounts.google.com/o/oauth2/auth?state=test",
		ExpiresAt:        time.Now().Add(10 * time.Minute),
	}, nil
}

func (s *stubEmailCredentialUsecase) Callback(ctx context.Context, userID uint, code, state string) error {
	return nil
}

func (s *stubEmailCredentialUsecase) ListConnections(ctx context.Context, userID uint) ([]ecdomain.ConnectionView, error) {
	return []ecdomain.ConnectionView{}, nil
}

func (s *stubEmailCredentialUsecase) Disconnect(ctx context.Context, userID uint, connectionID uint) error {
	return nil
}

func TestNewRouterRegistersVersionedAndLegacyRoutes(t *testing.T) {
	t.Parallel()

	g := gin.New()
	container := dig.New()
	osw := mocklibrary.NewOsWrapperMock(map[string]string{
		"APP":            "local",
		"DOMAIN":         "localhost",
		"JWT_SECRET_KEY": "test-secret",
	})
	log := logger.NewNop()

	err := container.Provide(func() *authpresentation.Controller {
		return authpresentation.NewController(&stubAuthUseCase{}, log, osw)
	})
	assert.NoError(t, err)
	err = container.Provide(func() *middleware.AuthMiddleware {
		return middleware.NewAuthMiddleware(osw, &stubAuthUserProvider{}, log)
	})
	assert.NoError(t, err)
	err = container.Provide(func() *macpresentation.Controller {
		return macpresentation.NewController(&stubEmailCredentialUsecase{}, log)
	})
	assert.NoError(t, err)

	domain, _ := osw.GetEnv("DOMAIN")
	_, err = v1.NewRouter(g, container, log, domain)
	assert.NoError(t, err)

	routes := map[string]struct{}{}
	for _, route := range g.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}

	expectedRoutes := []string{
		"GET /api/v1",
		"GET /",
		"POST /api/v1/auth/login",
		"POST /api/v1/auth/refresh",
		"POST /api/v1/auth/logout",
		"POST /api/v1/auth/register",
		"POST /api/v1/auth/email/verify",
		"POST /api/v1/auth/email/resend",
		"GET /api/v1/auth/check",
		"GET /api/v1/mail-account-connections",
		"DELETE /api/v1/mail-account-connections/:connection_id",
		"POST /api/v1/mail-account-connections/gmail/authorize",
		"POST /api/v1/mail-account-connections/gmail/callback",
	}
	for _, route := range expectedRoutes {
		assert.Contains(t, routes, route)
	}

	unexpectedRoutes := []string{
		"POST /auth/login",
		"POST /auth/logout",
		"POST /auth/register",
		"POST /auth/email/verify",
		"POST /auth/email/resend",
		"GET /auth/check",
	}
	for _, route := range unexpectedRoutes {
		assert.NotContains(t, routes, route)
	}
}
