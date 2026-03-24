package infrastructure

import (
	commondomain "business/internal/common/domain"
	"business/internal/library/logger"
	"business/internal/library/mysql"
	"business/internal/mailanalysis/domain"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type parsedEmailRepoTestEnv struct {
	repo  *GormParsedEmailRepositoryAdapter
	db    *gorm.DB
	clean func() error
}

func newParsedEmailRepoTestEnv(t *testing.T) *parsedEmailRepoTestEnv {
	t.Helper()

	mysqlConn, cleanup, err := mysql.CreateNewTestDB()
	if err != nil {
		skipIfParsedEmailRepoDBUnavailable(t, err)
	}
	require.NoError(t, err)
	require.NoError(t, mysqlConn.DB.AutoMigrate(&parsedEmailRecord{}))

	return &parsedEmailRepoTestEnv{
		repo:  NewGormParsedEmailRepositoryAdapter(mysqlConn.DB, logger.NewNop()),
		db:    mysqlConn.DB,
		clean: cleanup,
	}
}

func skipIfParsedEmailRepoDBUnavailable(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "dial tcp") || strings.Contains(err.Error(), "lookup mysql") {
		t.Skipf("Skipping repository integration test: %v", err)
	}
}

func TestGormParsedEmailRepositoryAdapter_SaveAll(t *testing.T) {
	t.Parallel()

	env := newParsedEmailRepoTestEnv(t)
	defer env.clean()

	ctx := context.Background()
	extractedAt := time.Date(2026, 3, 24, 10, 0, 0, 0, time.UTC)

	records, err := env.repo.SaveAll(ctx, domain.SaveInput{
		UserID:        1,
		EmailID:       10,
		AnalysisRunID: "run-1",
		PositionBase:  5,
		ExtractedAt:   extractedAt,
		PromptVersion: "emailanalysis_v1",
		ParsedEmails: []commondomain.ParsedEmail{
			{
				ProductNameRaw:     stringPtr(" Example Product Full Name "),
				ProductNameDisplay: stringPtr(" Example Product "),
				VendorName:         stringPtr(" Example Vendor "),
				BillingNumber:      stringPtr(" INV-001 "),
				InvoiceNumber:      stringPtr(" INV-RAW-001 "),
				Amount:             float64Ptr(123.456),
				Currency:           stringPtr(" jpy "),
				PaymentCycle:       stringPtr("one time"),
			},
			{
				Amount:       float64Ptr(9.99),
				Currency:     stringPtr("usd"),
				PaymentCycle: stringPtr(" recurring "),
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, records, 2)
	require.NotZero(t, records[0].ID)
	require.NotZero(t, records[1].ID)

	var stored []parsedEmailRecord
	require.NoError(t, env.db.WithContext(ctx).Order("position asc").Find(&stored).Error)
	require.Len(t, stored, 2)

	require.Equal(t, uint(1), stored[0].UserID)
	require.Equal(t, uint(10), stored[0].EmailID)
	require.Equal(t, "run-1", stored[0].AnalysisRunID)
	require.Equal(t, 5, stored[0].Position)
	require.Equal(t, "Example Product Full Name", *stored[0].ProductNameRaw)
	require.Equal(t, "Example Product", *stored[0].ProductNameDisplay)
	require.Equal(t, "Example Vendor", *stored[0].VendorName)
	require.Equal(t, "INV-001", *stored[0].BillingNumber)
	require.Equal(t, "INV-RAW-001", *stored[0].InvoiceNumber)
	require.Equal(t, 123.456, *stored[0].Amount)
	require.Equal(t, "JPY", *stored[0].Currency)
	require.Equal(t, "one_time", *stored[0].PaymentCycle)
	require.True(t, stored[0].ExtractedAt.Equal(extractedAt))
	require.Equal(t, "emailanalysis_v1", stored[0].PromptVersion)
	require.False(t, stored[0].CreatedAt.IsZero())
	require.False(t, stored[0].UpdatedAt.IsZero())

	require.Equal(t, 6, stored[1].Position)
	require.Equal(t, "USD", *stored[1].Currency)
	require.Equal(t, "recurring", *stored[1].PaymentCycle)
}

func TestGormParsedEmailRepositoryAdapter_SaveAll_NormalizedEmptyDraftsNoop(t *testing.T) {
	t.Parallel()

	env := newParsedEmailRepoTestEnv(t)
	defer env.clean()

	ctx := context.Background()
	records, err := env.repo.SaveAll(ctx, domain.SaveInput{
		UserID:        1,
		EmailID:       10,
		AnalysisRunID: "run-1",
		ExtractedAt:   time.Date(2026, 3, 24, 10, 0, 0, 0, time.UTC),
		PromptVersion: "emailanalysis_v1",
		ParsedEmails: []commondomain.ParsedEmail{
			{VendorName: stringPtr("   ")},
		},
	})
	require.NoError(t, err)
	require.Len(t, records, 0)

	var count int64
	require.NoError(t, env.db.WithContext(ctx).Model(&parsedEmailRecord{}).Count(&count).Error)
	require.Zero(t, count)
}

func stringPtr(value string) *string {
	return &value
}

func float64Ptr(value float64) *float64 {
	return &value
}
