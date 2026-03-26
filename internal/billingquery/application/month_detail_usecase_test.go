package application

import (
	"business/internal/library/logger"
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
)

type stubBillingMonthDetailRepository struct {
	monthDetail func(ctx context.Context, query MonthDetailQuery) (MonthDetailReadModel, error)
}

func (s *stubBillingMonthDetailRepository) MonthDetail(ctx context.Context, query MonthDetailQuery) (MonthDetailReadModel, error) {
	return s.monthDetail(ctx, query)
}

func TestMonthDetailUseCase_Get_AppliesDefaultsAndBuildsTopFivePlusOther(t *testing.T) {
	t.Parallel()

	var capturedQuery MonthDetailQuery
	uc := NewMonthDetailUseCase(&stubBillingMonthDetailRepository{
		monthDetail: func(ctx context.Context, query MonthDetailQuery) (MonthDetailReadModel, error) {
			capturedQuery = query
			return MonthDetailReadModel{
				TotalAmount:          decimal.RequireFromString("182400.000"),
				BillingCount:         12,
				FallbackBillingCount: 3,
				VendorItems: []MonthDetailVendorAggregate{
					{VendorName: " Notion ", TotalAmount: decimal.RequireFromString("15000"), BillingCount: 1},
					{VendorName: "GitHub", TotalAmount: decimal.RequireFromString("11200"), BillingCount: 1},
					{VendorName: "Google Workspace", TotalAmount: decimal.RequireFromString("36000"), BillingCount: 2},
					{VendorName: "AWS", TotalAmount: decimal.RequireFromString("82000"), BillingCount: 4},
					{VendorName: "OpenAI", TotalAmount: decimal.RequireFromString("24000"), BillingCount: 2},
					{VendorName: "Slack", TotalAmount: decimal.RequireFromString("8200"), BillingCount: 1},
					{VendorName: "Zoom", TotalAmount: decimal.RequireFromString("6000"), BillingCount: 1},
				},
			}, nil
		},
	}, logger.NewNop())

	result, err := uc.Get(context.Background(), MonthDetailQuery{
		UserID:    7,
		YearMonth: " 2026-03 ",
	})
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}

	if capturedQuery.YearMonth != "2026-03" {
		t.Fatalf("expected normalized year_month, got %#v", capturedQuery.YearMonth)
	}
	if capturedQuery.Currency != "JPY" {
		t.Fatalf("expected default currency JPY, got %#v", capturedQuery.Currency)
	}

	if result.YearMonth != "2026-03" {
		t.Fatalf("unexpected year_month: %#v", result.YearMonth)
	}
	if result.Currency != "JPY" {
		t.Fatalf("unexpected currency: %#v", result.Currency)
	}
	if result.TotalAmount != 182400 {
		t.Fatalf("unexpected total_amount: %#v", result.TotalAmount)
	}
	if result.BillingCount != 12 || result.FallbackBillingCount != 3 {
		t.Fatalf("unexpected counts: %+v", result)
	}
	if result.VendorLimit != 5 {
		t.Fatalf("unexpected vendor_limit: %d", result.VendorLimit)
	}
	if len(result.VendorItems) != 6 {
		t.Fatalf("expected top 5 + other, got %+v", result.VendorItems)
	}

	expectedOrder := []string{"AWS", "Google Workspace", "OpenAI", "Notion", "GitHub", "その他"}
	for idx, name := range expectedOrder {
		if result.VendorItems[idx].VendorName != name {
			t.Fatalf("unexpected vendor order at %d: %+v", idx, result.VendorItems)
		}
	}
	if result.VendorItems[5].TotalAmount != 14200 || result.VendorItems[5].BillingCount != 2 || !result.VendorItems[5].IsOther {
		t.Fatalf("unexpected other row: %+v", result.VendorItems[5])
	}
}

func TestMonthDetailUseCase_Get_RejectsInvalidQuery(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		query MonthDetailQuery
	}{
		{
			name:  "missing user id",
			query: MonthDetailQuery{YearMonth: "2026-03"},
		},
		{
			name:  "invalid year month",
			query: MonthDetailQuery{UserID: 1, YearMonth: "2026-13"},
		},
		{
			name:  "invalid currency",
			query: MonthDetailQuery{UserID: 1, YearMonth: "2026-03", Currency: "EUR"},
		},
	}

	uc := NewMonthDetailUseCase(&stubBillingMonthDetailRepository{
		monthDetail: func(ctx context.Context, query MonthDetailQuery) (MonthDetailReadModel, error) {
			t.Fatal("repository should not be called for invalid query")
			return MonthDetailReadModel{}, nil
		},
	}, logger.NewNop())

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := uc.Get(context.Background(), tc.query)
			if !errors.Is(err, ErrInvalidMonthDetailQuery) {
				t.Fatalf("expected ErrInvalidMonthDetailQuery, got %v", err)
			}
		})
	}
}

func TestMonthDetailUseCase_Get_InitializesEmptyVendorItems(t *testing.T) {
	t.Parallel()

	uc := NewMonthDetailUseCase(&stubBillingMonthDetailRepository{
		monthDetail: func(ctx context.Context, query MonthDetailQuery) (MonthDetailReadModel, error) {
			return MonthDetailReadModel{}, nil
		},
	}, logger.NewNop())

	result, err := uc.Get(context.Background(), MonthDetailQuery{UserID: 1, YearMonth: "2026-03"})
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if result.VendorItems == nil {
		t.Fatal("expected empty vendor_items slice, got nil")
	}
	if result.Currency != "JPY" {
		t.Fatalf("expected default currency JPY, got %#v", result.Currency)
	}
}
