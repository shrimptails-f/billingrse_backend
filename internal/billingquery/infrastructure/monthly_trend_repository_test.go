package infrastructure

import (
	billingqueryapp "business/internal/billingquery/application"
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func seedBillingMonthlyTrendFixtures(t *testing.T, db *gorm.DB) {
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
			ExternalMessageID: "msg-jpy-apr",
			Subject:           "April billing",
			FromRaw:           "AWS Billing <billing@aws.amazon.com>",
			ToJSON:            `["user1@example.com"]`,
			BodyDigest:        "digest-101",
			ReceivedAt:        time.Date(2025, 4, 10, 10, 0, 0, 0, time.UTC),
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                102,
			UserID:            1,
			Provider:          "gmail",
			AccountIdentifier: "user1@example.com",
			ExternalMessageID: "msg-jpy-may-fallback",
			Subject:           "May billing",
			FromRaw:           "AWS Billing <billing@aws.amazon.com>",
			ToJSON:            `["user1@example.com"]`,
			BodyDigest:        "digest-102",
			ReceivedAt:        time.Date(2025, 5, 5, 9, 0, 0, 0, time.UTC),
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                103,
			UserID:            1,
			Provider:          "gmail",
			AccountIdentifier: "user1@example.com",
			ExternalMessageID: "msg-usd-jun",
			Subject:           "June billing",
			FromRaw:           "Google Workspace <billing@google.com>",
			ToJSON:            `["user1@example.com"]`,
			BodyDigest:        "digest-103",
			ReceivedAt:        time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                104,
			UserID:            1,
			Provider:          "gmail",
			AccountIdentifier: "user1@example.com",
			ExternalMessageID: "msg-jpy-feb-by-billing-date",
			Subject:           "February billing date",
			FromRaw:           "Google Workspace <billing@google.com>",
			ToJSON:            `["user1@example.com"]`,
			BodyDigest:        "digest-104",
			ReceivedAt:        time.Date(2026, 3, 3, 8, 0, 0, 0, time.UTC),
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                105,
			UserID:            1,
			Provider:          "gmail",
			AccountIdentifier: "user1@example.com",
			ExternalMessageID: "msg-jpy-mar-fallback",
			Subject:           "March fallback billing",
			FromRaw:           "Google Workspace <billing@google.com>",
			ToJSON:            `["user1@example.com"]`,
			BodyDigest:        "digest-105",
			ReceivedAt:        time.Date(2026, 3, 20, 7, 0, 0, 0, time.UTC),
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                106,
			UserID:            1,
			Provider:          "gmail",
			AccountIdentifier: "user1@example.com",
			ExternalMessageID: "msg-jpy-mar-billing-date",
			Subject:           "March billing date",
			FromRaw:           "AWS Billing <billing@aws.amazon.com>",
			ToJSON:            `["user1@example.com"]`,
			BodyDigest:        "digest-106",
			ReceivedAt:        time.Date(2026, 3, 22, 7, 0, 0, 0, time.UTC),
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                107,
			UserID:            2,
			Provider:          "gmail",
			AccountIdentifier: "user2@example.com",
			ExternalMessageID: "msg-other-user",
			Subject:           "Other user billing",
			FromRaw:           "AWS Billing <billing@aws.amazon.com>",
			ToJSON:            `["user2@example.com"]`,
			BodyDigest:        "digest-107",
			ReceivedAt:        time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC),
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                108,
			UserID:            1,
			Provider:          "gmail",
			AccountIdentifier: "user1@example.com",
			ExternalMessageID: "msg-outside-window",
			Subject:           "Outside window billing",
			FromRaw:           "AWS Billing <billing@aws.amazon.com>",
			ToJSON:            `["user1@example.com"]`,
			BodyDigest:        "digest-108",
			ReceivedAt:        time.Date(2025, 3, 31, 10, 0, 0, 0, time.UTC),
			CreatedAt:         now,
			UpdatedAt:         now,
		},
	}
	require.NoError(t, db.Create(&emails).Error)

	billingDateApril := time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC)
	billingDateJune := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	billingDateFebruary := time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)
	billingDateMarch := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)
	billingDateOutside := time.Date(2025, 3, 31, 0, 0, 0, 0, time.UTC)
	summaryDateMayFallback := time.Date(2025, 5, 5, 9, 0, 0, 0, time.UTC)
	summaryDateMarchFallback := time.Date(2026, 3, 20, 7, 0, 0, 0, time.UTC)
	summaryDateOtherUser := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
	productAWS := "AWS Support"
	productGoogle := "Google Workspace"

	billings := []billingRecord{
		{
			ID:                 201,
			UserID:             1,
			VendorID:           1,
			EmailID:            101,
			ProductNameDisplay: &productAWS,
			BillingNumber:      "INV-JPY-APR",
			Amount:             decimal.RequireFromString("100.000"),
			Currency:           "JPY",
			BillingDate:        &billingDateApril,
			BillingSummaryDate: billingDateApril,
			PaymentCycle:       "recurring",
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		{
			ID:                 202,
			UserID:             1,
			VendorID:           1,
			EmailID:            102,
			ProductNameDisplay: &productAWS,
			BillingNumber:      "INV-JPY-MAY",
			Amount:             decimal.RequireFromString("50.500"),
			Currency:           "JPY",
			BillingDate:        nil,
			BillingSummaryDate: summaryDateMayFallback,
			PaymentCycle:       "recurring",
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		{
			ID:                 203,
			UserID:             1,
			VendorID:           2,
			EmailID:            103,
			ProductNameDisplay: &productGoogle,
			BillingNumber:      "INV-USD-JUN",
			Amount:             decimal.RequireFromString("70.000"),
			Currency:           "USD",
			BillingDate:        &billingDateJune,
			BillingSummaryDate: billingDateJune,
			PaymentCycle:       "recurring",
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		{
			ID:                 204,
			UserID:             1,
			VendorID:           2,
			EmailID:            104,
			ProductNameDisplay: &productGoogle,
			BillingNumber:      "INV-JPY-FEB",
			Amount:             decimal.RequireFromString("20.250"),
			Currency:           "JPY",
			BillingDate:        &billingDateFebruary,
			BillingSummaryDate: billingDateFebruary,
			PaymentCycle:       "recurring",
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		{
			ID:                 205,
			UserID:             1,
			VendorID:           2,
			EmailID:            105,
			ProductNameDisplay: &productGoogle,
			BillingNumber:      "INV-JPY-MAR-FALLBACK",
			Amount:             decimal.RequireFromString("30.000"),
			Currency:           "JPY",
			BillingDate:        nil,
			BillingSummaryDate: summaryDateMarchFallback,
			PaymentCycle:       "recurring",
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		{
			ID:                 206,
			UserID:             1,
			VendorID:           1,
			EmailID:            106,
			ProductNameDisplay: &productAWS,
			BillingNumber:      "INV-JPY-MAR",
			Amount:             decimal.RequireFromString("10.125"),
			Currency:           "JPY",
			BillingDate:        &billingDateMarch,
			BillingSummaryDate: billingDateMarch,
			PaymentCycle:       "recurring",
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		{
			ID:                 207,
			UserID:             2,
			VendorID:           1,
			EmailID:            107,
			ProductNameDisplay: &productAWS,
			BillingNumber:      "INV-OTHER-USER",
			Amount:             decimal.RequireFromString("999.000"),
			Currency:           "JPY",
			BillingDate:        nil,
			BillingSummaryDate: summaryDateOtherUser,
			PaymentCycle:       "recurring",
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		{
			ID:                 208,
			UserID:             1,
			VendorID:           1,
			EmailID:            108,
			ProductNameDisplay: &productAWS,
			BillingNumber:      "INV-OUTSIDE-WINDOW",
			Amount:             decimal.RequireFromString("5.000"),
			Currency:           "JPY",
			BillingDate:        &billingDateOutside,
			BillingSummaryDate: billingDateOutside,
			PaymentCycle:       "recurring",
			CreatedAt:          now,
			UpdatedAt:          now,
		},
	}
	require.NoError(t, db.Create(&billings).Error)
}

func TestBillingQueryRepository_MonthlyTrend_AggregatesByMonth(t *testing.T) {
	t.Parallel()

	env := newBillingListRepoTestEnv(t)
	defer env.clean()
	seedBillingMonthlyTrendFixtures(t, env.db)

	windowEndMonth := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	result, err := env.repo.MonthlyTrend(context.Background(), billingqueryapp.MonthlyTrendQuery{
		UserID:         1,
		Currency:       "JPY",
		WindowEndMonth: &windowEndMonth,
	})
	require.NoError(t, err)
	require.Equal(t, []billingqueryapp.MonthlyTrendAggregate{
		{
			YearMonth:            "2025-04",
			TotalAmount:          decimal.RequireFromString("100.000"),
			BillingCount:         1,
			FallbackBillingCount: 0,
		},
		{
			YearMonth:            "2025-05",
			TotalAmount:          decimal.RequireFromString("50.500"),
			BillingCount:         1,
			FallbackBillingCount: 1,
		},
		{
			YearMonth:            "2026-02",
			TotalAmount:          decimal.RequireFromString("20.250"),
			BillingCount:         1,
			FallbackBillingCount: 0,
		},
		{
			YearMonth:            "2026-03",
			TotalAmount:          decimal.RequireFromString("40.125"),
			BillingCount:         2,
			FallbackBillingCount: 1,
		},
	}, result)
}

func TestBillingQueryRepository_MonthlyTrend_FiltersCurrency(t *testing.T) {
	t.Parallel()

	env := newBillingListRepoTestEnv(t)
	defer env.clean()
	seedBillingMonthlyTrendFixtures(t, env.db)

	windowEndMonth := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	result, err := env.repo.MonthlyTrend(context.Background(), billingqueryapp.MonthlyTrendQuery{
		UserID:         1,
		Currency:       "USD",
		WindowEndMonth: &windowEndMonth,
	})
	require.NoError(t, err)
	require.Equal(t, []billingqueryapp.MonthlyTrendAggregate{
		{
			YearMonth:            "2025-06",
			TotalAmount:          decimal.RequireFromString("70.000"),
			BillingCount:         1,
			FallbackBillingCount: 0,
		},
	}, result)
}
