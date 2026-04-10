package application

import (
	"business/internal/library/logger"
	"business/internal/library/timewrapper"
	"context"
	"fmt"
	"time"
)

// SummaryQuery is the application query for the dashboard summary API.
type SummaryQuery struct {
	UserID uint
}

// Validate checks the minimum contract required for the dashboard summary API.
func (q SummaryQuery) Validate() error {
	if q.UserID == 0 {
		return fmt.Errorf("%w: user_id is required", ErrInvalidSummaryQuery)
	}
	return nil
}

// BillingCounts is the billing-side aggregate used by the dashboard summary API.
type BillingCounts struct {
	TotalSavedBillingCount           int
	CurrentMonthFallbackBillingCount int
}

// SummaryResult is the usecase result for the dashboard summary API.
type SummaryResult struct {
	CurrentMonthAnalysisSuccessCount int
	TotalSavedBillingCount           int
	CurrentMonthFallbackBillingCount int
}

// DashboardSummaryRepository loads the dashboard summary aggregates from storage.
type DashboardSummaryRepository interface {
	CountCurrentMonthAnalysisSuccess(ctx context.Context, userID uint, monthStartAt, nextMonthStartAt time.Time) (int, error)
	GetBillingCounts(ctx context.Context, userID uint, monthStartAt, nextMonthStartAt time.Time) (BillingCounts, error)
}

// SummaryUseCaseInterface provides the dashboard summary API.
type SummaryUseCaseInterface interface {
	Get(ctx context.Context, query SummaryQuery) (SummaryResult, error)
}

type summaryUseCase struct {
	repository DashboardSummaryRepository
	clock      timewrapper.ClockInterface
	log        logger.Interface
}

// SummaryUseCase is the concrete dashboard summary usecase type exposed for DI.
type SummaryUseCase = summaryUseCase

// NewSummaryUseCase creates a dashboard summary usecase.
func NewSummaryUseCase(
	repository DashboardSummaryRepository,
	clock timewrapper.ClockInterface,
	log logger.Interface,
) *SummaryUseCase {
	if clock == nil {
		clock = timewrapper.NewClock()
	}
	if log == nil {
		log = logger.NewNop()
	}

	return &summaryUseCase{
		repository: repository,
		clock:      clock,
		log:        log.With(logger.Component("dashboard_summary_usecase")),
	}
}

// Get returns the authenticated user's dashboard summary.
func (uc *summaryUseCase) Get(ctx context.Context, query SummaryQuery) (SummaryResult, error) {
	if ctx == nil {
		return SummaryResult{}, logger.ErrNilContext
	}
	if uc.repository == nil {
		return SummaryResult{}, fmt.Errorf("dashboard_summary_repository is not configured")
	}
	if err := query.Validate(); err != nil {
		return SummaryResult{}, err
	}

	currentMonthStartAt, nextMonthStartAt := currentMonthRangeUTC(uc.clock.Now())

	currentMonthAnalysisSuccessCount, err := uc.repository.CountCurrentMonthAnalysisSuccess(
		ctx,
		query.UserID,
		currentMonthStartAt,
		nextMonthStartAt,
	)
	if err != nil {
		return SummaryResult{}, err
	}

	billingCounts, err := uc.repository.GetBillingCounts(ctx, query.UserID, currentMonthStartAt, nextMonthStartAt)
	if err != nil {
		return SummaryResult{}, err
	}

	return SummaryResult{
		CurrentMonthAnalysisSuccessCount: currentMonthAnalysisSuccessCount,
		TotalSavedBillingCount:           billingCounts.TotalSavedBillingCount,
		CurrentMonthFallbackBillingCount: billingCounts.CurrentMonthFallbackBillingCount,
	}, nil
}

func currentMonthRangeUTC(now time.Time) (time.Time, time.Time) {
	utcNow := now.UTC()
	currentMonthStartAt := time.Date(utcNow.Year(), utcNow.Month(), 1, 0, 0, 0, 0, time.UTC)
	return currentMonthStartAt, currentMonthStartAt.AddDate(0, 1, 0)
}
