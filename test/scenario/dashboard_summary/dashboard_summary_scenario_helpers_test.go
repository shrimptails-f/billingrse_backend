package test

import (
	"business/internal/app/middleware"
	authpresentation "business/internal/app/presentation/auth"
	billingpresentation "business/internal/app/presentation/billing"
	dashboardpresentation "business/internal/app/presentation/dashboard"
	macpresentation "business/internal/app/presentation/mailaccountconnection"
	manualpresentation "business/internal/app/presentation/manualmailworkflow"
	v1 "business/internal/app/router"
	authdomain "business/internal/auth/domain"
	authinfra "business/internal/auth/infrastructure"
	dashboardqueryapp "business/internal/dashboardquery/application"
	dashboardqueryinfra "business/internal/dashboardquery/infrastructure"
	"business/internal/library/logger"
	"business/internal/library/mysql"
	mocklibrary "business/test/mock/library"
	model "business/tools/migrations/models"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/dig"
	"gorm.io/gorm"
)

const (
	dashboardSummaryScenarioJWTSecret     = "scenario-jwt-secret"
	dashboardSummaryScenarioAllowedOrigin = "http://localhost:3000"
)

type dashboardSummaryScenarioEnv struct {
	t           *testing.T
	db          *gorm.DB
	router      *gin.Engine
	clock       *dashboardSummaryScenarioClock
	userID      uint
	otherUserID uint
}

type dashboardSummaryScenarioClock struct {
	now time.Time
}

type dashboardSummaryResponse struct {
	CurrentMonthAnalysisSuccessCount int `json:"current_month_analysis_success_count"`
	TotalSavedBillingCount           int `json:"total_saved_billing_count"`
	CurrentMonthFallbackBillingCount int `json:"current_month_fallback_billing_count"`
}

type dashboardSummaryErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *dashboardSummaryScenarioClock) Now() time.Time {
	return c.now
}

func (c *dashboardSummaryScenarioClock) After(d time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	ch <- c.now.Add(d)
	return ch
}

func newDashboardSummaryScenarioEnv(t *testing.T) *dashboardSummaryScenarioEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	mysqlConn, cleanup, err := mysql.CreateNewTestDB()
	if err != nil {
		skipDashboardSummaryScenarioDBUnavailable(t, err)
	}
	require.NoError(t, err)

	t.Cleanup(func() {
		if cleanup != nil {
			require.NoError(t, cleanup())
		}
	})

	require.NoError(t, mysqlConn.DB.AutoMigrate(
		&model.User{},
		&model.Email{},
		&model.ParsedEmail{},
		&model.Billing{},
	))

	log := logger.NewNop()
	osw := mocklibrary.NewOsWrapperMock(map[string]string{
		"APP":            "test",
		"DOMAIN":         "localhost",
		"JWT_SECRET_KEY": dashboardSummaryScenarioJWTSecret,
	})

	authRepo := authinfra.NewRepository(mysqlConn.DB, log)
	authMiddleware := middleware.NewAuthMiddleware(osw, authRepo, log)
	authController := authpresentation.NewController(nil, log, osw)

	clock := &dashboardSummaryScenarioClock{
		now: time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
	}
	dashboardRepository := dashboardqueryinfra.NewDashboardSummaryRepository(mysqlConn.DB, log)
	dashboardUseCase := dashboardqueryapp.NewSummaryUseCase(dashboardRepository, clock, log)
	dashboardController := dashboardpresentation.NewController(dashboardUseCase, log)

	router := gin.New()
	container := dig.New()
	require.NoError(t, container.Provide(func() *authpresentation.Controller { return authController }))
	require.NoError(t, container.Provide(func() *middleware.AuthMiddleware { return authMiddleware }))
	require.NoError(t, container.Provide(func() *macpresentation.Controller {
		return macpresentation.NewController(nil, log)
	}))
	require.NoError(t, container.Provide(func() *manualpresentation.Controller {
		return manualpresentation.NewController(nil, nil, log)
	}))
	require.NoError(t, container.Provide(func() *billingpresentation.Controller {
		return billingpresentation.NewController(nil, nil, nil, log)
	}))
	require.NoError(t, container.Provide(func() *dashboardpresentation.Controller { return dashboardController }))

	_, err = v1.Router(router, container, log, dashboardSummaryScenarioAllowedOrigin)
	require.NoError(t, err)

	env := &dashboardSummaryScenarioEnv{
		t:      t,
		db:     mysqlConn.DB,
		router: router,
		clock:  clock,
	}
	env.userID = env.mustCreateVerifiedUser("dashboard-scenario-user-1", "dashboard-scenario-user-1@example.com")
	env.otherUserID = env.mustCreateVerifiedUser("dashboard-scenario-user-2", "dashboard-scenario-user-2@example.com")
	return env
}

func skipDashboardSummaryScenarioDBUnavailable(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}

	msg := err.Error()
	skipPatterns := []string{
		"dial tcp",
		"connect: connection refused",
		"lookup mysql",
		"access denied",
		"environment variable MYSQL_",
		"environment variable DB_HOST",
	}
	for _, pattern := range skipPatterns {
		if strings.Contains(msg, pattern) {
			t.Skipf("Skipping DashboardSummary scenario test: %v", err)
		}
	}
}

func (e *dashboardSummaryScenarioEnv) mustCreateVerifiedUser(name, email string) uint {
	e.t.Helper()

	now := e.clock.Now().UTC()
	user := model.User{
		Name:            name,
		Email:           email,
		Password:        "$2a$10$abcdefghijklmnopqrstuv",
		EmailVerified:   true,
		EmailVerifiedAt: &now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	require.NoError(e.t, e.db.Create(&user).Error)
	return user.ID
}

func (e *dashboardSummaryScenarioEnv) mustInsertParsedEmails(rows []model.ParsedEmail) {
	e.t.Helper()
	require.NoError(e.t, e.db.Create(&rows).Error)
}

func (e *dashboardSummaryScenarioEnv) mustInsertBillings(rows []model.Billing) {
	e.t.Helper()
	require.NoError(e.t, e.db.Create(&rows).Error)
}

func (e *dashboardSummaryScenarioEnv) getSummary(userID uint) *httptest.ResponseRecorder {
	e.t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
	req.Header.Set("Authorization", "Bearer "+e.mustIssueJWT(userID))

	resp := httptest.NewRecorder()
	e.router.ServeHTTP(resp, req)
	return resp
}

func (e *dashboardSummaryScenarioEnv) getSummaryWithoutAuth() *httptest.ResponseRecorder {
	e.t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
	resp := httptest.NewRecorder()
	e.router.ServeHTTP(resp, req)
	return resp
}

func (e *dashboardSummaryScenarioEnv) mustIssueJWT(userID uint) string {
	e.t.Helper()

	claims := authdomain.AuthClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(e.clock.Now().Add(1 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(dashboardSummaryScenarioJWTSecret))
	require.NoError(e.t, err)
	return signed
}

func (e *dashboardSummaryScenarioEnv) mustDecodeSummaryResponse(resp *httptest.ResponseRecorder) dashboardSummaryResponse {
	e.t.Helper()
	var out dashboardSummaryResponse
	e.mustDecodeResponse(resp, &out)
	return out
}

func (e *dashboardSummaryScenarioEnv) mustDecodeErrorResponse(resp *httptest.ResponseRecorder) dashboardSummaryErrorResponse {
	e.t.Helper()
	var out dashboardSummaryErrorResponse
	e.mustDecodeResponse(resp, &out)
	return out
}

func (e *dashboardSummaryScenarioEnv) mustDecodeResponse(resp *httptest.ResponseRecorder, out any) {
	e.t.Helper()
	require.NoError(e.t, json.Unmarshal(resp.Body.Bytes(), out))
}
