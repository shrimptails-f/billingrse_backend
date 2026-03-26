package infrastructure

import (
	billingqueryapp "business/internal/billingquery/application"
	"business/internal/library/logger"
	"business/internal/library/mysql"
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type billingListRepoTestEnv struct {
	repo  *BillingQueryRepository
	db    *gorm.DB
	clean func() error
}

func newBillingListRepoTestEnv(t *testing.T) *billingListRepoTestEnv {
	t.Helper()

	mysqlConn, cleanup, err := mysql.CreateNewTestDB()
	if err != nil {
		skipIfBillingRepoDBUnavailable(t, err)
	}
	require.NoError(t, err)
	require.NoError(t, mysqlConn.DB.AutoMigrate(
		&billingListVendorRecord{},
		&billingListEmailRecord{},
		&billingRecord{},
	))

	return &billingListRepoTestEnv{
		repo:  NewBillingQueryRepository(mysqlConn.DB, &billingRepoFixedClock{now: time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)}, logger.NewNop()),
		db:    mysqlConn.DB,
		clean: cleanup,
	}
}

func seedBillingListFixtures(t *testing.T, db *gorm.DB) {
	t.Helper()

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)

	vendors := []billingListVendorRecord{
		{ID: 1, Name: "AWS", NormalizedName: "aws", CreatedAt: now, UpdatedAt: now},
		{ID: 2, Name: "Google Workspace", NormalizedName: "google workspace", CreatedAt: now, UpdatedAt: now},
	}
	require.NoError(t, db.Create(&vendors).Error)

	emails := []billingListEmailRecord{
		{
			ID:                101,
			UserID:            1,
			Provider:          "gmail",
			AccountIdentifier: "user1@example.com",
			ExternalMessageID: "msg-aws-001",
			Subject:           "AWS billing",
			FromRaw:           "AWS Billing <billing@aws.amazon.com>",
			ToJSON:            `["user1@example.com"]`,
			BodyDigest:        "digest-101",
			ReceivedAt:        time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                102,
			UserID:            1,
			Provider:          "gmail",
			AccountIdentifier: "user1@example.com",
			ExternalMessageID: "msg-google-001",
			Subject:           "Google billing",
			FromRaw:           "Google Workspace <billing@google.com>",
			ToJSON:            `["user1@example.com"]`,
			BodyDigest:        "digest-102",
			ReceivedAt:        time.Date(2026, 3, 22, 12, 0, 0, 0, time.UTC),
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                103,
			UserID:            1,
			Provider:          "gmail",
			AccountIdentifier: "user1@example.com",
			ExternalMessageID: "msg-aws-002",
			Subject:           "AWS billing missing billing date",
			FromRaw:           "AWS Billing <billing@aws.amazon.com>",
			ToJSON:            `["user1@example.com"]`,
			BodyDigest:        "digest-103",
			ReceivedAt:        time.Date(2026, 3, 21, 15, 0, 0, 0, time.UTC),
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                104,
			UserID:            2,
			Provider:          "gmail",
			AccountIdentifier: "user2@example.com",
			ExternalMessageID: "msg-other-user",
			Subject:           "Other user billing",
			FromRaw:           "AWS Billing <billing@aws.amazon.com>",
			ToJSON:            `["user2@example.com"]`,
			BodyDigest:        "digest-104",
			ReceivedAt:        time.Date(2026, 3, 23, 9, 0, 0, 0, time.UTC),
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                105,
			UserID:            1,
			Provider:          "gmail",
			AccountIdentifier: "user1@example.com",
			ExternalMessageID: "msg-google-002",
			Subject:           "Google billing second invoice",
			FromRaw:           "Google Workspace <billing@google.com>",
			ToJSON:            `["user1@example.com"]`,
			BodyDigest:        "digest-105",
			ReceivedAt:        time.Date(2026, 3, 22, 12, 0, 0, 0, time.UTC),
			CreatedAt:         now,
			UpdatedAt:         now,
		},
	}
	require.NoError(t, db.Create(&emails).Error)

	billingDateMarch1 := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	billingDateMarch25 := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)
	summaryDateMarch21 := time.Date(2026, 3, 21, 15, 0, 0, 0, time.UTC)
	productAWS1 := "AWS Support Enterprise"
	productAWS2 := "AWS Support Business"
	productGoogle1 := "Google Workspace Business"
	productGoogle2 := "Google Workspace Enterprise"

	billings := []billingRecord{
		{
			ID:                 201,
			UserID:             1,
			VendorID:           1,
			EmailID:            101,
			ProductNameDisplay: &productAWS1,
			BillingNumber:      "INV-AWS-001",
			Amount:             decimal.RequireFromString("1200.500"),
			Currency:           "JPY",
			BillingDate:        &billingDateMarch1,
			BillingSummaryDate: billingDateMarch1,
			PaymentCycle:       "recurring",
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		{
			ID:                 202,
			UserID:             1,
			VendorID:           1,
			EmailID:            103,
			ProductNameDisplay: &productAWS2,
			BillingNumber:      "INV-AWS-002",
			Amount:             decimal.RequireFromString("800.000"),
			Currency:           "JPY",
			BillingDate:        nil,
			BillingSummaryDate: summaryDateMarch21,
			PaymentCycle:       "recurring",
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		{
			ID:                 203,
			UserID:             1,
			VendorID:           2,
			EmailID:            102,
			ProductNameDisplay: &productGoogle1,
			BillingNumber:      "INV-GWS-001",
			Amount:             decimal.RequireFromString("99.990"),
			Currency:           "USD",
			BillingDate:        &billingDateMarch25,
			BillingSummaryDate: billingDateMarch25,
			PaymentCycle:       "recurring",
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		{
			ID:                 204,
			UserID:             2,
			VendorID:           1,
			EmailID:            104,
			ProductNameDisplay: &productAWS1,
			BillingNumber:      "INV-OTHER-001",
			Amount:             decimal.RequireFromString("500.000"),
			Currency:           "JPY",
			BillingDate:        &billingDateMarch25,
			BillingSummaryDate: billingDateMarch25,
			PaymentCycle:       "one_time",
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		{
			ID:                 205,
			UserID:             1,
			VendorID:           2,
			EmailID:            105,
			ProductNameDisplay: &productGoogle2,
			BillingNumber:      "INV-GWS-002",
			Amount:             decimal.RequireFromString("199.990"),
			Currency:           "USD",
			BillingDate:        &billingDateMarch25,
			BillingSummaryDate: billingDateMarch25,
			PaymentCycle:       "recurring",
			CreatedAt:          now,
			UpdatedAt:          now,
		},
	}
	require.NoError(t, db.Create(&billings).Error)
}

func TestBillingQueryRepository_List_AppliesUserScopeAndFilters(t *testing.T) {
	t.Parallel()

	env := newBillingListRepoTestEnv(t)
	defer env.clean()
	seedBillingListFixtures(t, env.db)

	emailID := uint(101)
	limit := 10
	offset := 0
	fallback := true

	result, err := env.repo.List(context.Background(), billingqueryapp.ListQuery{
		UserID:                1,
		Q:                     "aws",
		EmailID:               &emailID,
		UseReceivedAtFallback: &fallback,
		Limit:                 &limit,
		Offset:                &offset,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), result.TotalCount)
	require.Len(t, result.Items, 1)
	require.Equal(t, uint(101), result.Items[0].EmailID)
	require.Equal(t, "msg-aws-001", result.Items[0].ExternalMessageID)
	require.Equal(t, "AWS", result.Items[0].VendorName)

	result, err = env.repo.List(context.Background(), billingqueryapp.ListQuery{
		UserID:            1,
		ExternalMessageID: "msg-google-001",
		Limit:             &limit,
		Offset:            &offset,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), result.TotalCount)
	require.Len(t, result.Items, 1)
	require.Equal(t, uint(102), result.Items[0].EmailID)
	require.Equal(t, "msg-google-001", result.Items[0].ExternalMessageID)
}

func TestBillingQueryRepository_List_DateFallbackChangesResults(t *testing.T) {
	t.Parallel()

	env := newBillingListRepoTestEnv(t)
	defer env.clean()
	seedBillingListFixtures(t, env.db)

	limit := 10
	offset := 0
	dateFrom := time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(2026, 3, 22, 23, 59, 59, 0, time.UTC)
	fallbackTrue := true
	fallbackFalse := false

	withFallback, err := env.repo.List(context.Background(), billingqueryapp.ListQuery{
		UserID:                1,
		DateFrom:              &dateFrom,
		DateTo:                &dateTo,
		UseReceivedAtFallback: &fallbackTrue,
		Limit:                 &limit,
		Offset:                &offset,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), withFallback.TotalCount)
	require.Len(t, withFallback.Items, 1)
	require.Equal(t, uint(103), withFallback.Items[0].EmailID)
	require.Nil(t, withFallback.Items[0].BillingDate)

	withoutFallback, err := env.repo.List(context.Background(), billingqueryapp.ListQuery{
		UserID:                1,
		DateFrom:              &dateFrom,
		DateTo:                &dateTo,
		UseReceivedAtFallback: &fallbackFalse,
		Limit:                 &limit,
		Offset:                &offset,
	})
	require.NoError(t, err)
	require.Equal(t, int64(0), withoutFallback.TotalCount)
	require.Empty(t, withoutFallback.Items)
}

func TestBillingQueryRepository_List_UsesStableOrderingAndTotalCount(t *testing.T) {
	t.Parallel()

	env := newBillingListRepoTestEnv(t)
	defer env.clean()
	seedBillingListFixtures(t, env.db)

	fallbackTrue := true
	fallbackFalse := false
	limitOne := 1
	offsetZero := 0
	limitTen := 10

	page, err := env.repo.List(context.Background(), billingqueryapp.ListQuery{
		UserID:                1,
		Q:                     "google",
		UseReceivedAtFallback: &fallbackTrue,
		Limit:                 &limitOne,
		Offset:                &offsetZero,
	})
	require.NoError(t, err)
	require.Equal(t, int64(2), page.TotalCount)
	require.Len(t, page.Items, 1)
	require.Equal(t, uint(105), page.Items[0].EmailID)

	withFallback, err := env.repo.List(context.Background(), billingqueryapp.ListQuery{
		UserID:                1,
		UseReceivedAtFallback: &fallbackTrue,
		Limit:                 &limitTen,
		Offset:                &offsetZero,
	})
	require.NoError(t, err)
	require.Equal(t, []uint{105, 102, 103, 101}, billingListEmailIDs(withFallback.Items))

	withoutFallback, err := env.repo.List(context.Background(), billingqueryapp.ListQuery{
		UserID:                1,
		UseReceivedAtFallback: &fallbackFalse,
		Limit:                 &limitTen,
		Offset:                &offsetZero,
	})
	require.NoError(t, err)
	require.Equal(t, []uint{105, 102, 101, 103}, billingListEmailIDs(withoutFallback.Items))
}

func billingListEmailIDs(items []billingqueryapp.ListItem) []uint {
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.EmailID)
	}
	return ids
}
