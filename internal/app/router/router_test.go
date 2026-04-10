package router

import (
	"context"
	"testing"
	"time"

	"business/internal/app/middleware"
	authpresentation "business/internal/app/presentation/auth"
	billingpresentation "business/internal/app/presentation/billing"
	dashboardpresentation "business/internal/app/presentation/dashboard"
	macpresentation "business/internal/app/presentation/mailaccountconnection"
	manualpresentation "business/internal/app/presentation/manualmailworkflow"
	"business/internal/auth/domain"
	billingqueryapp "business/internal/billingquery/application"
	dashboardqueryapp "business/internal/dashboardquery/application"
	"business/internal/library/logger"
	macapp "business/internal/mailaccountconnection/application"
	macdomain "business/internal/mailaccountconnection/domain"
	manualapp "business/internal/manualmailworkflow/application"
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

func (s *stubEmailCredentialUsecase) Authorize(ctx context.Context, userID uint) (macapp.AuthorizeResult, error) {
	return macapp.AuthorizeResult{
		AuthorizationURL: "https://accounts.google.com/o/oauth2/auth?state=test",
		ExpiresAt:        time.Now().Add(10 * time.Minute),
	}, nil
}

func (s *stubEmailCredentialUsecase) Callback(ctx context.Context, userID uint, code, state string) error {
	return nil
}

func (s *stubEmailCredentialUsecase) ListConnections(ctx context.Context, userID uint) ([]macdomain.ConnectionView, error) {
	return []macdomain.ConnectionView{}, nil
}

func (s *stubEmailCredentialUsecase) Disconnect(ctx context.Context, userID uint, connectionID uint) error {
	return nil
}

type stubManualMailWorkflowUseCase struct{}

func (s *stubManualMailWorkflowUseCase) Start(ctx context.Context, cmd manualapp.Command) (manualapp.StartResult, error) {
	return manualapp.StartResult{
		WorkflowID: "wf-test",
		Status:     manualapp.WorkflowStatusQueued,
	}, nil
}

type stubManualMailWorkflowListUseCase struct{}

func (s *stubManualMailWorkflowListUseCase) List(ctx context.Context, query manualapp.ListQuery) (manualapp.ListResult, error) {
	return manualapp.ListResult{Items: []manualapp.WorkflowHistoryListItem{}}, nil
}

type stubBillingListUseCase struct{}

func (s *stubBillingListUseCase) List(ctx context.Context, query billingqueryapp.ListQuery) (billingqueryapp.ListResult, error) {
	return billingqueryapp.ListResult{Items: []billingqueryapp.ListItem{}}, nil
}

type stubBillingMonthlyTrendUseCase struct{}

func (s *stubBillingMonthlyTrendUseCase) Get(ctx context.Context, query billingqueryapp.MonthlyTrendQuery) (billingqueryapp.MonthlyTrendResult, error) {
	return billingqueryapp.MonthlyTrendResult{Items: []billingqueryapp.MonthlyTrendItem{}}, nil
}

type stubBillingMonthDetailUseCase struct{}

func (s *stubBillingMonthDetailUseCase) Get(ctx context.Context, query billingqueryapp.MonthDetailQuery) (billingqueryapp.MonthDetailResult, error) {
	return billingqueryapp.MonthDetailResult{VendorItems: []billingqueryapp.MonthDetailVendorItem{}}, nil
}

type stubDashboardSummaryUseCase struct{}

func (s *stubDashboardSummaryUseCase) Get(ctx context.Context, query dashboardqueryapp.SummaryQuery) (dashboardqueryapp.SummaryResult, error) {
	return dashboardqueryapp.SummaryResult{}, nil
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
	err = container.Provide(func() *manualpresentation.Controller {
		return manualpresentation.NewController(&stubManualMailWorkflowUseCase{}, &stubManualMailWorkflowListUseCase{}, log)
	})
	assert.NoError(t, err)
	err = container.Provide(func() *billingpresentation.Controller {
		return billingpresentation.NewController(
			&stubBillingListUseCase{},
			&stubBillingMonthlyTrendUseCase{},
			&stubBillingMonthDetailUseCase{},
			log,
		)
	})
	assert.NoError(t, err)
	err = container.Provide(func() *dashboardpresentation.Controller {
		return dashboardpresentation.NewController(&stubDashboardSummaryUseCase{}, log)
	})
	assert.NoError(t, err)

	domain, _ := osw.GetEnv("DOMAIN")
	_, err = Router(g, container, log, domain)
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
		"GET /api/v1/manual-mail-workflows",
		"POST /api/v1/manual-mail-workflows",
		"GET /api/v1/billings",
		"GET /api/v1/billings/summary/monthly-trend",
		"GET /api/v1/billings/summary/monthly-detail/:year_month",
		"GET /api/v1/dashboard/summary",
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
