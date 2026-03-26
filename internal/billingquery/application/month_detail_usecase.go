package application

import (
	commondomain "business/internal/common/domain"
	"business/internal/library/logger"
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const (
	defaultBillingMonthDetailCurrency = "JPY"
	billingMonthDetailVendorLimit     = 5
	billingMonthDetailOtherVendorName = "その他"
	billingMonthDetailLayout          = "2006-01"
)

// MonthDetailQuery is the application query for a single-month billing detail.
type MonthDetailQuery struct {
	UserID    uint
	YearMonth string
	Currency  string
}

// Normalize trims free-form inputs and applies defaults.
func (q MonthDetailQuery) Normalize() MonthDetailQuery {
	q.YearMonth = strings.TrimSpace(q.YearMonth)
	q.Currency = strings.TrimSpace(q.Currency)
	if q.Currency == "" {
		q.Currency = defaultBillingMonthDetailCurrency
	}
	q.Currency = strings.ToUpper(q.Currency)
	return q
}

// Validate checks the minimum contract required for the month detail.
func (q MonthDetailQuery) Validate() error {
	if q.UserID == 0 {
		return fmt.Errorf("%w: user_id is required", ErrInvalidMonthDetailQuery)
	}
	if _, _, err := q.MonthRange(); err != nil {
		return fmt.Errorf("%w: year_month must be YYYY-MM", ErrInvalidMonthDetailQuery)
	}
	if _, err := commondomain.NormalizeCurrency(q.Currency); err != nil {
		return fmt.Errorf("%w: currency must be JPY or USD", ErrInvalidMonthDetailQuery)
	}
	return nil
}

// MonthRange returns the inclusive start and exclusive end of the selected month in UTC.
func (q MonthDetailQuery) MonthRange() (time.Time, time.Time, error) {
	parsed, err := time.Parse(billingMonthDetailLayout, strings.TrimSpace(q.YearMonth))
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	start := time.Date(parsed.Year(), parsed.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	return start, end, nil
}

// MonthDetailVendorAggregate is the repository read model for a vendor subtotal row.
type MonthDetailVendorAggregate struct {
	VendorName   string
	TotalAmount  decimal.Decimal
	BillingCount int
}

// MonthDetailReadModel is the repository read model before presentation-oriented shaping.
type MonthDetailReadModel struct {
	TotalAmount          decimal.Decimal
	BillingCount         int
	FallbackBillingCount int
	VendorItems          []MonthDetailVendorAggregate
}

// MonthDetailVendorItem is the response-facing vendor breakdown item.
type MonthDetailVendorItem struct {
	VendorName   string
	TotalAmount  float64
	BillingCount int
	IsOther      bool
}

// MonthDetailResult is the usecase result for the selected-month detail.
type MonthDetailResult struct {
	YearMonth            string
	Currency             string
	TotalAmount          float64
	BillingCount         int
	FallbackBillingCount int
	VendorLimit          int
	VendorItems          []MonthDetailVendorItem
}

// BillingMonthDetailRepository loads the raw totals and vendor breakdown for one month.
type BillingMonthDetailRepository interface {
	MonthDetail(ctx context.Context, query MonthDetailQuery) (MonthDetailReadModel, error)
}

// MonthDetailUseCaseInterface provides the selected-month billing detail.
type MonthDetailUseCaseInterface interface {
	Get(ctx context.Context, query MonthDetailQuery) (MonthDetailResult, error)
}

type monthDetailUseCase struct {
	repository BillingMonthDetailRepository
	log        logger.Interface
}

// MonthDetailUseCase is the concrete billing month-detail usecase type exposed for DI.
type MonthDetailUseCase = monthDetailUseCase

// NewMonthDetailUseCase creates a billing month-detail usecase.
func NewMonthDetailUseCase(repository BillingMonthDetailRepository, log logger.Interface) *MonthDetailUseCase {
	if log == nil {
		log = logger.NewNop()
	}

	return &monthDetailUseCase{
		repository: repository,
		log:        log.With(logger.Component("billing_month_detail_usecase")),
	}
}

// Get returns the selected-month billing detail.
func (uc *monthDetailUseCase) Get(ctx context.Context, query MonthDetailQuery) (MonthDetailResult, error) {
	if ctx == nil {
		return MonthDetailResult{}, logger.ErrNilContext
	}
	if uc.repository == nil {
		return MonthDetailResult{}, fmt.Errorf("billing_month_detail_repository is not configured")
	}

	query = query.Normalize()
	if err := query.Validate(); err != nil {
		return MonthDetailResult{}, err
	}

	normalizedCurrency, err := commondomain.NormalizeCurrency(query.Currency)
	if err != nil {
		return MonthDetailResult{}, fmt.Errorf("%w: currency must be JPY or USD", ErrInvalidMonthDetailQuery)
	}
	query.Currency = normalizedCurrency

	readModel, err := uc.repository.MonthDetail(ctx, query)
	if err != nil {
		return MonthDetailResult{}, err
	}
	if readModel.VendorItems == nil {
		readModel.VendorItems = []MonthDetailVendorAggregate{}
	}

	return MonthDetailResult{
		YearMonth:            query.YearMonth,
		Currency:             query.Currency,
		TotalAmount:          readModel.TotalAmount.InexactFloat64(),
		BillingCount:         readModel.BillingCount,
		FallbackBillingCount: readModel.FallbackBillingCount,
		VendorLimit:          billingMonthDetailVendorLimit,
		VendorItems:          buildMonthDetailVendorItems(readModel.VendorItems),
	}, nil
}

func buildMonthDetailVendorItems(items []MonthDetailVendorAggregate) []MonthDetailVendorItem {
	if len(items) == 0 {
		return []MonthDetailVendorItem{}
	}

	sortedItems := make([]MonthDetailVendorAggregate, 0, len(items))
	for _, item := range items {
		sortedItems = append(sortedItems, MonthDetailVendorAggregate{
			VendorName:   strings.TrimSpace(item.VendorName),
			TotalAmount:  item.TotalAmount,
			BillingCount: item.BillingCount,
		})
	}

	sort.Slice(sortedItems, func(i, j int) bool {
		amountCompare := sortedItems[i].TotalAmount.Cmp(sortedItems[j].TotalAmount)
		if amountCompare != 0 {
			return amountCompare > 0
		}
		if sortedItems[i].BillingCount != sortedItems[j].BillingCount {
			return sortedItems[i].BillingCount > sortedItems[j].BillingCount
		}
		return sortedItems[i].VendorName < sortedItems[j].VendorName
	})

	limit := billingMonthDetailVendorLimit
	if len(sortedItems) < limit {
		limit = len(sortedItems)
	}

	result := make([]MonthDetailVendorItem, 0, limit+1)
	for _, item := range sortedItems[:limit] {
		result = append(result, MonthDetailVendorItem{
			VendorName:   item.VendorName,
			TotalAmount:  item.TotalAmount.InexactFloat64(),
			BillingCount: item.BillingCount,
			IsOther:      false,
		})
	}

	if len(sortedItems) <= billingMonthDetailVendorLimit {
		return result
	}

	otherAmount := decimal.Zero
	otherCount := 0
	for _, item := range sortedItems[billingMonthDetailVendorLimit:] {
		otherAmount = otherAmount.Add(item.TotalAmount)
		otherCount += item.BillingCount
	}
	result = append(result, MonthDetailVendorItem{
		VendorName:   billingMonthDetailOtherVendorName,
		TotalAmount:  otherAmount.InexactFloat64(),
		BillingCount: otherCount,
		IsOther:      true,
	})

	return result
}
