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

func seedBillingMonthDetailFixtures(t *testing.T, db *gorm.DB) {
	t.Helper()

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)

	vendors := []billingListVendorRecord{
		{ID: 1, Name: "AWS", NormalizedName: "aws", CreatedAt: now, UpdatedAt: now},
		{ID: 2, Name: "Google Workspace", NormalizedName: "google workspace", CreatedAt: now, UpdatedAt: now},
		{ID: 3, Name: "OpenAI", NormalizedName: "openai", CreatedAt: now, UpdatedAt: now},
		{ID: 4, Name: "Notion", NormalizedName: "notion", CreatedAt: now, UpdatedAt: now},
		{ID: 5, Name: "GitHub", NormalizedName: "github", CreatedAt: now, UpdatedAt: now},
		{ID: 6, Name: "Slack", NormalizedName: "slack", CreatedAt: now, UpdatedAt: now},
		{ID: 7, Name: "Zoom", NormalizedName: "zoom", CreatedAt: now, UpdatedAt: now},
		{ID: 8, Name: "Dropbox", NormalizedName: "dropbox", CreatedAt: now, UpdatedAt: now},
	}
	require.NoError(t, db.Create(&vendors).Error)

	emails := []billingListEmailRecord{
		{
			ID:                201,
			UserID:            1,
			Provider:          "gmail",
			AccountIdentifier: "user1@example.com",
			ExternalMessageID: "msg-aws-main",
			Subject:           "AWS billing",
			FromRaw:           "AWS Billing <billing@aws.amazon.com>",
			ToJSON:            `["user1@example.com"]`,
			BodyDigest:        "digest-201",
			ReceivedAt:        time.Date(2026, 3, 5, 10, 0, 0, 0, time.UTC),
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                202,
			UserID:            1,
			Provider:          "gmail",
			AccountIdentifier: "user1@example.com",
			ExternalMessageID: "msg-google-jpy",
			Subject:           "Google billing",
			FromRaw:           "Google Workspace <billing@google.com>",
			ToJSON:            `["user1@example.com"]`,
			BodyDigest:        "digest-202",
			ReceivedAt:        time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC),
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                203,
			UserID:            1,
			Provider:          "gmail",
			AccountIdentifier: "user1@example.com",
			ExternalMessageID: "msg-openai",
			Subject:           "OpenAI billing",
			FromRaw:           "OpenAI <billing@openai.com>",
			ToJSON:            `["user1@example.com"]`,
			BodyDigest:        "digest-203",
			ReceivedAt:        time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC),
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                204,
			UserID:            1,
			Provider:          "gmail",
			AccountIdentifier: "user1@example.com",
			ExternalMessageID: "msg-notion",
			Subject:           "Notion billing",
			FromRaw:           "Notion <billing@notion.so>",
			ToJSON:            `["user1@example.com"]`,
			BodyDigest:        "digest-204",
			ReceivedAt:        time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC),
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                205,
			UserID:            1,
			Provider:          "gmail",
			AccountIdentifier: "user1@example.com",
			ExternalMessageID: "msg-github",
			Subject:           "GitHub billing",
			FromRaw:           "GitHub <billing@github.com>",
			ToJSON:            `["user1@example.com"]`,
			BodyDigest:        "digest-205",
			ReceivedAt:        time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                206,
			UserID:            1,
			Provider:          "gmail",
			AccountIdentifier: "user1@example.com",
			ExternalMessageID: "msg-slack",
			Subject:           "Slack billing",
			FromRaw:           "Slack <billing@slack.com>",
			ToJSON:            `["user1@example.com"]`,
			BodyDigest:        "digest-206",
			ReceivedAt:        time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC),
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                207,
			UserID:            1,
			Provider:          "gmail",
			AccountIdentifier: "user1@example.com",
			ExternalMessageID: "msg-zoom",
			Subject:           "Zoom billing",
			FromRaw:           "Zoom <billing@zoom.us>",
			ToJSON:            `["user1@example.com"]`,
			BodyDigest:        "digest-207",
			ReceivedAt:        time.Date(2026, 3, 22, 10, 0, 0, 0, time.UTC),
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                208,
			UserID:            1,
			Provider:          "gmail",
			AccountIdentifier: "user1@example.com",
			ExternalMessageID: "msg-aws-fallback",
			Subject:           "AWS billing fallback",
			FromRaw:           "AWS Billing <billing@aws.amazon.com>",
			ToJSON:            `["user1@example.com"]`,
			BodyDigest:        "digest-208",
			ReceivedAt:        time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC),
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                209,
			UserID:            1,
			Provider:          "gmail",
			AccountIdentifier: "user1@example.com",
			ExternalMessageID: "msg-dropbox-april",
			Subject:           "Dropbox billing",
			FromRaw:           "Dropbox <billing@dropbox.com>",
			ToJSON:            `["user1@example.com"]`,
			BodyDigest:        "digest-209",
			ReceivedAt:        time.Date(2026, 3, 25, 10, 0, 0, 0, time.UTC),
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                210,
			UserID:            1,
			Provider:          "gmail",
			AccountIdentifier: "user1@example.com",
			ExternalMessageID: "msg-google-usd",
			Subject:           "Google billing USD",
			FromRaw:           "Google Workspace <billing@google.com>",
			ToJSON:            `["user1@example.com"]`,
			BodyDigest:        "digest-210",
			ReceivedAt:        time.Date(2026, 3, 11, 10, 0, 0, 0, time.UTC),
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                211,
			UserID:            2,
			Provider:          "gmail",
			AccountIdentifier: "user2@example.com",
			ExternalMessageID: "msg-other-user",
			Subject:           "Other user billing",
			FromRaw:           "AWS Billing <billing@aws.amazon.com>",
			ToJSON:            `["user2@example.com"]`,
			BodyDigest:        "digest-211",
			ReceivedAt:        time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC),
			CreatedAt:         now,
			UpdatedAt:         now,
		},
	}
	require.NoError(t, db.Create(&emails).Error)

	march5 := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)
	march10 := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	march12 := time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)
	march15 := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	march20 := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	april1 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	summaryDateSlack := time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)
	summaryDateZoom := time.Date(2026, 3, 22, 10, 0, 0, 0, time.UTC)
	summaryDateAWSFallback := time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC)
	summaryDateOtherUser := time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)

	billings := []billingFixture{
		{
			ID:                 301,
			UserID:             1,
			VendorID:           1,
			EmailID:            201,
			BillingNumber:      "INV-AWS-001",
			Amount:             decimal.RequireFromString("82000.000"),
			Currency:           "JPY",
			BillingDate:        &march5,
			BillingSummaryDate: march5,
			PaymentCycle:       "recurring",
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		{
			ID:                 302,
			UserID:             1,
			VendorID:           2,
			EmailID:            202,
			BillingNumber:      "INV-GWS-JPY-001",
			Amount:             decimal.RequireFromString("36000.000"),
			Currency:           "JPY",
			BillingDate:        &march10,
			BillingSummaryDate: march10,
			PaymentCycle:       "recurring",
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		{
			ID:                 303,
			UserID:             1,
			VendorID:           3,
			EmailID:            203,
			BillingNumber:      "INV-OAI-001",
			Amount:             decimal.RequireFromString("24000.000"),
			Currency:           "JPY",
			BillingDate:        &march12,
			BillingSummaryDate: march12,
			PaymentCycle:       "recurring",
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		{
			ID:                 304,
			UserID:             1,
			VendorID:           4,
			EmailID:            204,
			BillingNumber:      "INV-NOTION-001",
			Amount:             decimal.RequireFromString("15000.000"),
			Currency:           "JPY",
			BillingDate:        &march15,
			BillingSummaryDate: march15,
			PaymentCycle:       "recurring",
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		{
			ID:                 305,
			UserID:             1,
			VendorID:           5,
			EmailID:            205,
			BillingNumber:      "INV-GH-001",
			Amount:             decimal.RequireFromString("11200.000"),
			Currency:           "JPY",
			BillingDate:        &march20,
			BillingSummaryDate: march20,
			PaymentCycle:       "recurring",
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		{
			ID:                 306,
			UserID:             1,
			VendorID:           6,
			EmailID:            206,
			BillingNumber:      "INV-SLACK-001",
			Amount:             decimal.RequireFromString("8200.000"),
			Currency:           "JPY",
			BillingDate:        nil,
			BillingSummaryDate: summaryDateSlack,
			PaymentCycle:       "recurring",
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		{
			ID:                 307,
			UserID:             1,
			VendorID:           7,
			EmailID:            207,
			BillingNumber:      "INV-ZOOM-001",
			Amount:             decimal.RequireFromString("6000.000"),
			Currency:           "JPY",
			BillingDate:        nil,
			BillingSummaryDate: summaryDateZoom,
			PaymentCycle:       "recurring",
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		{
			ID:                 308,
			UserID:             1,
			VendorID:           1,
			EmailID:            208,
			BillingNumber:      "INV-AWS-002",
			Amount:             decimal.RequireFromString("10000.000"),
			Currency:           "JPY",
			BillingDate:        nil,
			BillingSummaryDate: summaryDateAWSFallback,
			PaymentCycle:       "recurring",
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		{
			ID:                 309,
			UserID:             1,
			VendorID:           8,
			EmailID:            209,
			BillingNumber:      "INV-DROPBOX-001",
			Amount:             decimal.RequireFromString("9999.000"),
			Currency:           "JPY",
			BillingDate:        &april1,
			BillingSummaryDate: april1,
			PaymentCycle:       "recurring",
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		{
			ID:                 310,
			UserID:             1,
			VendorID:           2,
			EmailID:            210,
			BillingNumber:      "INV-GWS-USD-001",
			Amount:             decimal.RequireFromString("99.990"),
			Currency:           "USD",
			BillingDate:        &march10,
			BillingSummaryDate: march10,
			PaymentCycle:       "recurring",
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		{
			ID:                 311,
			UserID:             2,
			VendorID:           1,
			EmailID:            211,
			BillingNumber:      "INV-OTHER-001",
			Amount:             decimal.RequireFromString("5000.000"),
			Currency:           "JPY",
			BillingDate:        &march12,
			BillingSummaryDate: summaryDateOtherUser,
			PaymentCycle:       "recurring",
			CreatedAt:          now,
			UpdatedAt:          now,
		},
	}
	billingRecords, lineItems := billingRecordsAndLineItemsFromFixtures(billings)
	require.NoError(t, db.Create(&billingRecords).Error)
	require.NoError(t, db.Create(&lineItems).Error)
}

func TestBillingQueryRepository_MonthDetail_AggregatesTotalsAndVendorBreakdown(t *testing.T) {
	t.Parallel()

	env := newBillingListRepoTestEnv(t)
	defer env.clean()
	seedBillingMonthDetailFixtures(t, env.db)

	result, err := env.repo.MonthDetail(context.Background(), billingqueryapp.MonthDetailQuery{
		UserID:    1,
		YearMonth: "2026-03",
		Currency:  "JPY",
	})
	require.NoError(t, err)

	require.True(t, result.TotalAmount.Equal(decimal.RequireFromString("192400.000")))
	require.Equal(t, 8, result.BillingCount)
	require.Equal(t, 3, result.FallbackBillingCount)
	require.Len(t, result.VendorItems, 7)
	require.Equal(t, []string{
		"AWS",
		"Google Workspace",
		"OpenAI",
		"Notion",
		"GitHub",
		"Slack",
		"Zoom",
	}, monthDetailVendorNames(result.VendorItems))
}

func TestBillingQueryRepository_MonthDetail_RespectsCurrencyAndBillingDatePriority(t *testing.T) {
	t.Parallel()

	env := newBillingListRepoTestEnv(t)
	defer env.clean()
	seedBillingMonthDetailFixtures(t, env.db)

	jpyResult, err := env.repo.MonthDetail(context.Background(), billingqueryapp.MonthDetailQuery{
		UserID:    1,
		YearMonth: "2026-03",
		Currency:  "JPY",
	})
	require.NoError(t, err)
	require.Equal(t, 8, jpyResult.BillingCount)

	usdResult, err := env.repo.MonthDetail(context.Background(), billingqueryapp.MonthDetailQuery{
		UserID:    1,
		YearMonth: "2026-03",
		Currency:  "USD",
	})
	require.NoError(t, err)
	require.True(t, usdResult.TotalAmount.Equal(decimal.RequireFromString("99.990")))
	require.Equal(t, 1, usdResult.BillingCount)
	require.Equal(t, 0, usdResult.FallbackBillingCount)
	require.Len(t, usdResult.VendorItems, 1)
	require.Equal(t, "Google Workspace", usdResult.VendorItems[0].VendorName)

	// Dropbox is received in March but belongs to April because billing_date takes precedence.
	require.NotContains(t, monthDetailVendorNames(jpyResult.VendorItems), "Dropbox")
}

func TestBillingQueryRepository_MonthDetail_ReturnsZeroValueForEmptyMonth(t *testing.T) {
	t.Parallel()

	env := newBillingListRepoTestEnv(t)
	defer env.clean()
	seedBillingMonthDetailFixtures(t, env.db)

	result, err := env.repo.MonthDetail(context.Background(), billingqueryapp.MonthDetailQuery{
		UserID:    1,
		YearMonth: "2026-02",
		Currency:  "JPY",
	})
	require.NoError(t, err)
	require.True(t, result.TotalAmount.Equal(decimal.Zero))
	require.Equal(t, 0, result.BillingCount)
	require.Equal(t, 0, result.FallbackBillingCount)
	require.Empty(t, result.VendorItems)
}

func monthDetailVendorNames(items []billingqueryapp.MonthDetailVendorAggregate) []string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.VendorName)
	}
	return names
}
