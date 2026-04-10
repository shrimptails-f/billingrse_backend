package infrastructure

import (
	dashboardqueryapp "business/internal/dashboardquery/application"
	"business/internal/library/logger"
	"business/internal/library/mysql"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type dashboardSummaryRepoTestEnv struct {
	repo  *DashboardSummaryRepository
	db    *gorm.DB
	clean func() error
}

func newDashboardSummaryRepoTestEnv(t *testing.T) *dashboardSummaryRepoTestEnv {
	t.Helper()

	mysqlConn, cleanup, err := mysql.CreateNewTestDB()
	if err != nil {
		skipIfDashboardSummaryRepoDBUnavailable(t, err)
	}
	require.NoError(t, err)
	require.NoError(t, mysqlConn.DB.AutoMigrate(
		&parsedEmailSummaryRecord{},
		&billingSummaryRecord{},
	))

	return &dashboardSummaryRepoTestEnv{
		repo:  NewDashboardSummaryRepository(mysqlConn.DB, logger.NewNop()),
		db:    mysqlConn.DB,
		clean: cleanup,
	}
}

func skipIfDashboardSummaryRepoDBUnavailable(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "dial tcp") || strings.Contains(err.Error(), "lookup mysql") {
		t.Skipf("Skipping repository integration test: %v", err)
	}
}

func seedDashboardSummaryFixtures(t *testing.T, db *gorm.DB) {
	t.Helper()

	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	march1 := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	march15 := time.Date(2026, 3, 15, 9, 30, 0, 0, time.UTC)
	march31 := time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC)
	april1 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	april3 := time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)
	feb28 := time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC)
	billingDate := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)

	parsedEmails := []parsedEmailSummaryRecord{
		{ID: 1, UserID: 1, EmailID: 101, ExtractedAt: march1, CreatedAt: now, UpdatedAt: now},
		{ID: 2, UserID: 1, EmailID: 102, ExtractedAt: march15, CreatedAt: now, UpdatedAt: now},
		{ID: 3, UserID: 1, EmailID: 103, ExtractedAt: march31, CreatedAt: now, UpdatedAt: now},
		{ID: 4, UserID: 1, EmailID: 104, ExtractedAt: feb28, CreatedAt: now, UpdatedAt: now},
		{ID: 5, UserID: 1, EmailID: 105, ExtractedAt: april1, CreatedAt: now, UpdatedAt: now},
		{ID: 6, UserID: 2, EmailID: 201, ExtractedAt: march15, CreatedAt: now, UpdatedAt: now},
	}
	require.NoError(t, db.Create(&parsedEmails).Error)

	billings := []billingSummaryRecord{
		{ID: 201, UserID: 1, BillingDate: &billingDate, BillingSummaryDate: billingDate, CreatedAt: now, UpdatedAt: now},
		{ID: 202, UserID: 1, BillingDate: nil, BillingSummaryDate: march15, CreatedAt: now, UpdatedAt: now},
		{ID: 203, UserID: 1, BillingDate: nil, BillingSummaryDate: april3, CreatedAt: now, UpdatedAt: now},
		{ID: 204, UserID: 2, BillingDate: nil, BillingSummaryDate: march31, CreatedAt: now, UpdatedAt: now},
	}
	require.NoError(t, db.Create(&billings).Error)
}

func TestDashboardSummaryRepository_CountCurrentMonthAnalysisSuccess_AppliesUserScopeAndMonthRange(t *testing.T) {
	t.Parallel()

	env := newDashboardSummaryRepoTestEnv(t)
	defer env.clean()
	seedDashboardSummaryFixtures(t, env.db)

	count, err := env.repo.CountCurrentMonthAnalysisSuccess(
		context.Background(),
		1,
		time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
	)
	require.NoError(t, err)
	require.Equal(t, 3, count)
}

func TestDashboardSummaryRepository_GetBillingCounts_AppliesUserScope(t *testing.T) {
	t.Parallel()

	env := newDashboardSummaryRepoTestEnv(t)
	defer env.clean()
	seedDashboardSummaryFixtures(t, env.db)

	result, err := env.repo.GetBillingCounts(
		context.Background(),
		1,
		time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
	)
	require.NoError(t, err)
	require.Equal(t, dashboardqueryapp.BillingCounts{
		TotalSavedBillingCount:           3,
		CurrentMonthFallbackBillingCount: 1,
	}, result)
}

func TestDashboardSummaryRepository_ReturnsZeroCountsWithoutData(t *testing.T) {
	t.Parallel()

	env := newDashboardSummaryRepoTestEnv(t)
	defer env.clean()

	count, err := env.repo.CountCurrentMonthAnalysisSuccess(
		context.Background(),
		1,
		time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
	)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	result, err := env.repo.GetBillingCounts(
		context.Background(),
		1,
		time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
	)
	require.NoError(t, err)
	require.Equal(t, dashboardqueryapp.BillingCounts{}, result)
}
