package application

import (
	"business/internal/library/logger"
	"context"
	"errors"
	"testing"
	"time"
)

type stubDashboardSummaryRepository struct {
	countCurrentMonthAnalysisSuccess func(ctx context.Context, userID uint, monthStartAt, nextMonthStartAt time.Time) (int, error)
	getBillingCounts                 func(ctx context.Context, userID uint, monthStartAt, nextMonthStartAt time.Time) (BillingCounts, error)
}

func (s *stubDashboardSummaryRepository) CountCurrentMonthAnalysisSuccess(
	ctx context.Context,
	userID uint,
	monthStartAt,
	nextMonthStartAt time.Time,
) (int, error) {
	return s.countCurrentMonthAnalysisSuccess(ctx, userID, monthStartAt, nextMonthStartAt)
}

func (s *stubDashboardSummaryRepository) GetBillingCounts(
	ctx context.Context,
	userID uint,
	monthStartAt,
	nextMonthStartAt time.Time,
) (BillingCounts, error) {
	return s.getBillingCounts(ctx, userID, monthStartAt, nextMonthStartAt)
}

type dashboardSummaryFixedClock struct {
	now time.Time
}

func (c *dashboardSummaryFixedClock) Now() time.Time {
	return c.now
}

func (c *dashboardSummaryFixedClock) After(d time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	ch <- c.now.Add(d)
	return ch
}

func TestSummaryUseCase_Get_UsesUTCCurrentMonthAndBuildsResult(t *testing.T) {
	t.Parallel()

	jst := time.FixedZone("JST", 9*60*60)
	now := time.Date(2026, 4, 1, 8, 30, 0, 0, jst)

	var capturedUserID uint
	var capturedMonthStartAt time.Time
	var capturedNextMonthStartAt time.Time

	uc := NewSummaryUseCase(&stubDashboardSummaryRepository{
		countCurrentMonthAnalysisSuccess: func(
			ctx context.Context,
			userID uint,
			monthStartAt,
			nextMonthStartAt time.Time,
		) (int, error) {
			capturedUserID = userID
			capturedMonthStartAt = monthStartAt
			capturedNextMonthStartAt = nextMonthStartAt
			return 12, nil
		},
		getBillingCounts: func(ctx context.Context, userID uint, monthStartAt, nextMonthStartAt time.Time) (BillingCounts, error) {
			if userID != 7 {
				t.Fatalf("unexpected userID for billing counts: %d", userID)
			}
			expectedMonthStartAt := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
			expectedNextMonthStartAt := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
			if !monthStartAt.Equal(expectedMonthStartAt) {
				t.Fatalf("expected billing monthStartAt %s, got %s", expectedMonthStartAt, monthStartAt)
			}
			if !nextMonthStartAt.Equal(expectedNextMonthStartAt) {
				t.Fatalf("expected billing nextMonthStartAt %s, got %s", expectedNextMonthStartAt, nextMonthStartAt)
			}
			return BillingCounts{
				TotalSavedBillingCount:           34,
				CurrentMonthFallbackBillingCount: 5,
			}, nil
		},
	}, &dashboardSummaryFixedClock{now: now}, logger.NewNop())

	result, err := uc.Get(context.Background(), SummaryQuery{UserID: 7})
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}

	if capturedUserID != 7 {
		t.Fatalf("expected userID 7, got %d", capturedUserID)
	}
	expectedMonthStartAt := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	expectedNextMonthStartAt := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	if !capturedMonthStartAt.Equal(expectedMonthStartAt) {
		t.Fatalf("expected monthStartAt %s, got %s", expectedMonthStartAt, capturedMonthStartAt)
	}
	if !capturedNextMonthStartAt.Equal(expectedNextMonthStartAt) {
		t.Fatalf("expected nextMonthStartAt %s, got %s", expectedNextMonthStartAt, capturedNextMonthStartAt)
	}

	if result.CurrentMonthAnalysisSuccessCount != 12 {
		t.Fatalf("unexpected current month analysis success count: %+v", result)
	}
	if result.TotalSavedBillingCount != 34 {
		t.Fatalf("unexpected total saved billing count: %+v", result)
	}
	if result.CurrentMonthFallbackBillingCount != 5 {
		t.Fatalf("unexpected current month fallback billing count: %+v", result)
	}
}

func TestSummaryUseCase_Get_RejectsInvalidQuery(t *testing.T) {
	t.Parallel()

	uc := NewSummaryUseCase(&stubDashboardSummaryRepository{
		countCurrentMonthAnalysisSuccess: func(
			ctx context.Context,
			userID uint,
			monthStartAt,
			nextMonthStartAt time.Time,
		) (int, error) {
			t.Fatal("repository should not be called for invalid query")
			return 0, nil
		},
		getBillingCounts: func(ctx context.Context, userID uint, monthStartAt, nextMonthStartAt time.Time) (BillingCounts, error) {
			t.Fatal("repository should not be called for invalid query")
			return BillingCounts{}, nil
		},
	}, &dashboardSummaryFixedClock{now: time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)}, logger.NewNop())

	_, err := uc.Get(context.Background(), SummaryQuery{})
	if !errors.Is(err, ErrInvalidSummaryQuery) {
		t.Fatalf("expected ErrInvalidSummaryQuery, got %v", err)
	}
}

func TestSummaryUseCase_Get_ReturnsZeroCounts(t *testing.T) {
	t.Parallel()

	uc := NewSummaryUseCase(&stubDashboardSummaryRepository{
		countCurrentMonthAnalysisSuccess: func(
			ctx context.Context,
			userID uint,
			monthStartAt,
			nextMonthStartAt time.Time,
		) (int, error) {
			return 0, nil
		},
		getBillingCounts: func(ctx context.Context, userID uint, monthStartAt, nextMonthStartAt time.Time) (BillingCounts, error) {
			return BillingCounts{}, nil
		},
	}, &dashboardSummaryFixedClock{now: time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)}, logger.NewNop())

	result, err := uc.Get(context.Background(), SummaryQuery{UserID: 1})
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if result.CurrentMonthAnalysisSuccessCount != 0 || result.TotalSavedBillingCount != 0 || result.CurrentMonthFallbackBillingCount != 0 {
		t.Fatalf("expected zero counts, got %+v", result)
	}
}
