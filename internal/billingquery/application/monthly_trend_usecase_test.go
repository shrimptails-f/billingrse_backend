package application

import (
	"business/internal/library/logger"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

type stubBillingMonthlyTrendRepository struct {
	monthlyTrend func(ctx context.Context, query MonthlyTrendQuery) ([]MonthlyTrendAggregate, error)
}

func (s *stubBillingMonthlyTrendRepository) MonthlyTrend(ctx context.Context, query MonthlyTrendQuery) ([]MonthlyTrendAggregate, error) {
	return s.monthlyTrend(ctx, query)
}

type monthlyTrendFixedClock struct {
	now time.Time
}

func (c *monthlyTrendFixedClock) Now() time.Time {
	return c.now
}

func (c *monthlyTrendFixedClock) After(d time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	ch <- c.now.Add(d)
	return ch
}

func TestMonthlyTrendUseCase_Get_AppliesDefaultsAndZeroFills(t *testing.T) {
	t.Parallel()

	nowUTC := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	var capturedQuery MonthlyTrendQuery

	uc := NewMonthlyTrendUseCase(&stubBillingMonthlyTrendRepository{
		monthlyTrend: func(ctx context.Context, query MonthlyTrendQuery) ([]MonthlyTrendAggregate, error) {
			capturedQuery = query
			return []MonthlyTrendAggregate{
				{
					YearMonth:            "2025-05",
					TotalAmount:          decimal.RequireFromString("120.500"),
					BillingCount:         2,
					FallbackBillingCount: 1,
				},
				{
					YearMonth:            "2026-03",
					TotalAmount:          decimal.RequireFromString("299.970"),
					BillingCount:         3,
					FallbackBillingCount: 1,
				},
			}, nil
		},
	}, &monthlyTrendFixedClock{now: nowUTC}, logger.NewNop())

	result, err := uc.Get(context.Background(), MonthlyTrendQuery{UserID: 7})
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}

	if capturedQuery.Currency != "JPY" {
		t.Fatalf("expected default currency JPY, got %q", capturedQuery.Currency)
	}
	if capturedQuery.WindowEndMonth == nil || !capturedQuery.WindowEndMonth.Equal(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected normalized window_end_month, got %#v", capturedQuery.WindowEndMonth)
	}

	if result.Currency != "JPY" {
		t.Fatalf("expected result currency JPY, got %q", result.Currency)
	}
	if result.WindowStartMonth != "2025-04" || result.WindowEndMonth != "2026-03" || result.DefaultSelectedMonth != "2026-03" {
		t.Fatalf("unexpected result window: %+v", result)
	}
	if len(result.Items) != billingMonthlyTrendWindowSize {
		t.Fatalf("expected %d items, got %d", billingMonthlyTrendWindowSize, len(result.Items))
	}

	if result.Items[0].YearMonth != "2025-04" || result.Items[0].TotalAmount != 0 || result.Items[0].BillingCount != 0 || result.Items[0].FallbackBillingCount != 0 {
		t.Fatalf("expected zero-filled first month, got %+v", result.Items[0])
	}
	if result.Items[1].YearMonth != "2025-05" || result.Items[1].TotalAmount != 120.5 || result.Items[1].BillingCount != 2 || result.Items[1].FallbackBillingCount != 1 {
		t.Fatalf("unexpected filled month bucket: %+v", result.Items[1])
	}
	last := result.Items[len(result.Items)-1]
	if last.YearMonth != "2026-03" || last.TotalAmount != 299.97 || last.BillingCount != 3 || last.FallbackBillingCount != 1 {
		t.Fatalf("unexpected last month bucket: %+v", last)
	}
}

func TestMonthlyTrendUseCase_Get_NormalizesCurrencyAndWindowEndMonth(t *testing.T) {
	t.Parallel()

	var capturedQuery MonthlyTrendQuery
	uc := NewMonthlyTrendUseCase(&stubBillingMonthlyTrendRepository{
		monthlyTrend: func(ctx context.Context, query MonthlyTrendQuery) ([]MonthlyTrendAggregate, error) {
			capturedQuery = query
			return nil, nil
		},
	}, &monthlyTrendFixedClock{now: time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)}, logger.NewNop())

	result, err := uc.Get(context.Background(), MonthlyTrendQuery{
		UserID:         3,
		Currency:       " usd ",
		WindowEndMonth: timePtr(time.Date(2026, 2, 20, 15, 0, 0, 0, time.FixedZone("JST", 9*60*60))),
	})
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}

	if capturedQuery.Currency != "USD" {
		t.Fatalf("expected normalized currency USD, got %q", capturedQuery.Currency)
	}
	if capturedQuery.WindowEndMonth == nil || !capturedQuery.WindowEndMonth.Equal(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected normalized window_end_month, got %#v", capturedQuery.WindowEndMonth)
	}
	if result.WindowStartMonth != "2025-03" || result.WindowEndMonth != "2026-02" {
		t.Fatalf("unexpected window result: %+v", result)
	}
}

func TestMonthlyTrendUseCase_Get_RejectsInvalidQuery(t *testing.T) {
	t.Parallel()

	uc := NewMonthlyTrendUseCase(&stubBillingMonthlyTrendRepository{
		monthlyTrend: func(ctx context.Context, query MonthlyTrendQuery) ([]MonthlyTrendAggregate, error) {
			t.Fatal("repository should not be called for invalid query")
			return nil, nil
		},
	}, &monthlyTrendFixedClock{now: time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)}, logger.NewNop())

	testCases := []MonthlyTrendQuery{
		{Currency: "JPY"},
		{UserID: 1, Currency: "EUR"},
	}

	for _, query := range testCases {
		_, err := uc.Get(context.Background(), query)
		if !errors.Is(err, ErrInvalidMonthlyTrendQuery) {
			t.Fatalf("expected ErrInvalidMonthlyTrendQuery, got %v", err)
		}
	}
}

func timePtr(value time.Time) *time.Time {
	return &value
}
