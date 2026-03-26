package application

import (
	commondomain "business/internal/common/domain"
	"business/internal/library/logger"
	"business/internal/library/timewrapper"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const (
	defaultBillingMonthlyTrendCurrency = "JPY"
	billingMonthlyTrendWindowSize      = 12
	billingMonthlyTrendLayout          = "2006-01"
)

// MonthlyTrendQuery is the application query for the fixed 12-month billing trend.
type MonthlyTrendQuery struct {
	UserID         uint
	Currency       string
	WindowEndMonth *time.Time
}

// Normalize trims free-form inputs and applies defaults.
func (q MonthlyTrendQuery) Normalize(now time.Time) MonthlyTrendQuery {
	q.Currency = strings.TrimSpace(q.Currency)
	if q.Currency == "" {
		q.Currency = defaultBillingMonthlyTrendCurrency
	}
	q.Currency = strings.ToUpper(q.Currency)
	if normalizedCurrency, err := commondomain.NormalizeCurrency(q.Currency); err == nil {
		q.Currency = normalizedCurrency
	}

	if q.WindowEndMonth == nil || q.WindowEndMonth.IsZero() {
		month := normalizeMonthlyTrendMonthStart(now)
		q.WindowEndMonth = &month
		return q
	}

	month := normalizeMonthlyTrendMonthStart(*q.WindowEndMonth)
	q.WindowEndMonth = &month
	return q
}

// Validate checks the minimum contract required for the monthly trend.
func (q MonthlyTrendQuery) Validate() error {
	if q.UserID == 0 {
		return fmt.Errorf("%w: user_id is required", ErrInvalidMonthlyTrendQuery)
	}
	if _, err := commondomain.NormalizeCurrency(q.Currency); err != nil {
		return fmt.Errorf("%w: currency must be JPY or USD", ErrInvalidMonthlyTrendQuery)
	}
	if q.WindowEndMonth == nil || q.WindowEndMonth.IsZero() {
		return fmt.Errorf("%w: window_end_month must be YYYY-MM", ErrInvalidMonthlyTrendQuery)
	}
	return nil
}

// WindowStartMonth returns the first month in the fixed trend window.
func (q MonthlyTrendQuery) WindowStartMonth() time.Time {
	return q.windowEndMonthValue().AddDate(0, -(billingMonthlyTrendWindowSize - 1), 0)
}

// WindowStartAt returns the inclusive lower bound for aggregation.
func (q MonthlyTrendQuery) WindowStartAt() time.Time {
	return q.WindowStartMonth()
}

// WindowEndAtExclusive returns the exclusive upper bound for aggregation.
func (q MonthlyTrendQuery) WindowEndAtExclusive() time.Time {
	return q.windowEndMonthValue().AddDate(0, 1, 0)
}

// MonthlyTrendAggregate is the repository aggregate for a non-empty month bucket.
type MonthlyTrendAggregate struct {
	YearMonth            string
	TotalAmount          decimal.Decimal
	BillingCount         int
	FallbackBillingCount int
}

// MonthlyTrendItem is the response-facing month bucket.
type MonthlyTrendItem struct {
	YearMonth            string
	TotalAmount          float64
	BillingCount         int
	FallbackBillingCount int
}

// MonthlyTrendResult is the usecase result for the monthly trend graph API.
type MonthlyTrendResult struct {
	Currency             string
	WindowStartMonth     string
	WindowEndMonth       string
	DefaultSelectedMonth string
	Items                []MonthlyTrendItem
}

// BillingMonthlyTrendRepository loads the raw monthly totals for the fixed window.
type BillingMonthlyTrendRepository interface {
	MonthlyTrend(ctx context.Context, query MonthlyTrendQuery) ([]MonthlyTrendAggregate, error)
}

// MonthlyTrendUseCaseInterface provides the fixed 12-month billing trend.
type MonthlyTrendUseCaseInterface interface {
	Get(ctx context.Context, query MonthlyTrendQuery) (MonthlyTrendResult, error)
}

type monthlyTrendUseCase struct {
	repository BillingMonthlyTrendRepository
	clock      timewrapper.ClockInterface
	log        logger.Interface
}

// MonthlyTrendUseCase is the concrete billing monthly-trend usecase type exposed for DI.
type MonthlyTrendUseCase = monthlyTrendUseCase

// NewMonthlyTrendUseCase creates a billing monthly-trend usecase.
func NewMonthlyTrendUseCase(
	repository BillingMonthlyTrendRepository,
	clock timewrapper.ClockInterface,
	log logger.Interface,
) *MonthlyTrendUseCase {
	if clock == nil {
		clock = timewrapper.NewClock()
	}
	if log == nil {
		log = logger.NewNop()
	}

	return &monthlyTrendUseCase{
		repository: repository,
		clock:      clock,
		log:        log.With(logger.Component("billing_monthly_trend_usecase")),
	}
}

// Get returns the fixed 12-month billing trend with zero-filled month buckets.
func (uc *monthlyTrendUseCase) Get(ctx context.Context, query MonthlyTrendQuery) (MonthlyTrendResult, error) {
	if ctx == nil {
		return MonthlyTrendResult{}, logger.ErrNilContext
	}
	if uc.repository == nil {
		return MonthlyTrendResult{}, fmt.Errorf("billing_monthly_trend_repository is not configured")
	}

	query = query.Normalize(uc.clock.Now())
	if err := query.Validate(); err != nil {
		return MonthlyTrendResult{}, err
	}

	aggregates, err := uc.repository.MonthlyTrend(ctx, query)
	if err != nil {
		return MonthlyTrendResult{}, err
	}

	aggregateByMonth := make(map[string]MonthlyTrendAggregate, len(aggregates))
	for _, aggregate := range aggregates {
		aggregateByMonth[aggregate.YearMonth] = aggregate
	}

	windowStartMonth := query.WindowStartMonth()
	windowEndMonth := query.windowEndMonthValue()

	items := make([]MonthlyTrendItem, 0, billingMonthlyTrendWindowSize)
	currentMonth := windowStartMonth
	for i := 0; i < billingMonthlyTrendWindowSize; i++ {
		yearMonth := currentMonth.Format(billingMonthlyTrendLayout)
		aggregate, ok := aggregateByMonth[yearMonth]
		if !ok {
			aggregate = MonthlyTrendAggregate{
				YearMonth:   yearMonth,
				TotalAmount: decimal.Zero,
			}
		}

		items = append(items, MonthlyTrendItem{
			YearMonth:            yearMonth,
			TotalAmount:          aggregate.TotalAmount.InexactFloat64(),
			BillingCount:         aggregate.BillingCount,
			FallbackBillingCount: aggregate.FallbackBillingCount,
		})
		currentMonth = currentMonth.AddDate(0, 1, 0)
	}

	return MonthlyTrendResult{
		Currency:             query.Currency,
		WindowStartMonth:     windowStartMonth.Format(billingMonthlyTrendLayout),
		WindowEndMonth:       windowEndMonth.Format(billingMonthlyTrendLayout),
		DefaultSelectedMonth: windowEndMonth.Format(billingMonthlyTrendLayout),
		Items:                items,
	}, nil
}

func (q MonthlyTrendQuery) windowEndMonthValue() time.Time {
	if q.WindowEndMonth == nil {
		return time.Time{}
	}
	return normalizeMonthlyTrendMonthStart(*q.WindowEndMonth)
}

func normalizeMonthlyTrendMonthStart(value time.Time) time.Time {
	utc := value.UTC()
	return time.Date(utc.Year(), utc.Month(), 1, 0, 0, 0, 0, time.UTC)
}
