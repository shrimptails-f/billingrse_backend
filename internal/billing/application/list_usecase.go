package application

import (
	"business/internal/billing/domain"
	"business/internal/library/logger"
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	defaultBillingListLimit  = 50
	defaultBillingListOffset = 0
	maxBillingListLimit      = 100
)

// ListQuery represents the input contract for the billing list use case.
type ListQuery struct {
	UserID                uint
	Q                     string
	EmailID               *uint
	ExternalMessageID     string
	DateFrom              *time.Time
	DateTo                *time.Time
	UseReceivedAtFallback *bool
	Limit                 *int
	Offset                *int
}

// Normalize trims free-form values, UTC-normalizes timestamps, and applies defaults.
func (q ListQuery) Normalize() ListQuery {
	q.Q = strings.TrimSpace(q.Q)
	q.ExternalMessageID = strings.TrimSpace(q.ExternalMessageID)
	q.EmailID = cloneUint(q.EmailID)
	q.DateFrom = cloneTime(q.DateFrom)
	q.DateTo = cloneTime(q.DateTo)
	q.UseReceivedAtFallback = cloneBool(q.UseReceivedAtFallback)
	q.Limit = cloneInt(q.Limit)
	q.Offset = cloneInt(q.Offset)

	if q.UseReceivedAtFallback == nil {
		defaultValue := true
		q.UseReceivedAtFallback = &defaultValue
	}
	if q.Limit == nil {
		defaultValue := defaultBillingListLimit
		q.Limit = &defaultValue
	}
	if q.Offset == nil {
		defaultValue := defaultBillingListOffset
		q.Offset = &defaultValue
	}

	return q
}

// Validate checks whether the query satisfies the billing list contract.
func (q ListQuery) Validate() error {
	if q.UserID == 0 {
		return fmt.Errorf("%w: user_id is required", domain.ErrInvalidListQuery)
	}
	if q.EmailID != nil && *q.EmailID == 0 {
		return fmt.Errorf("%w: email_id must be greater than zero", domain.ErrInvalidListQuery)
	}
	if q.Limit == nil || *q.Limit < 1 || *q.Limit > maxBillingListLimit {
		return fmt.Errorf("%w: limit must be between 1 and %d", domain.ErrInvalidListQuery, maxBillingListLimit)
	}
	if q.Offset == nil || *q.Offset < 0 {
		return fmt.Errorf("%w: offset must be greater than or equal to zero", domain.ErrInvalidListQuery)
	}
	if q.DateFrom != nil && q.DateTo != nil && q.DateFrom.After(*q.DateTo) {
		return fmt.Errorf("%w: date_from must be earlier than or equal to date_to", domain.ErrInvalidListQuery)
	}
	return nil
}

// ListItem is the read model item returned by the billing list API.
type ListItem struct {
	EmailID            uint
	ExternalMessageID  string
	VendorName         string
	ReceivedAt         time.Time
	BillingDate        *time.Time
	ProductNameDisplay *string
	Amount             float64
	Currency           string
}

// ListResult is the paginated result returned by the billing list API.
type ListResult struct {
	Items      []ListItem
	Limit      int
	Offset     int
	TotalCount int64
}

// BillingListRepository loads billing list read models.
type BillingListRepository interface {
	List(ctx context.Context, query ListQuery) (ListResult, error)
}

// ListUseCase loads billing list items for the authenticated user.
type ListUseCase interface {
	List(ctx context.Context, query ListQuery) (ListResult, error)
}

type listUseCase struct {
	repository BillingListRepository
	log        logger.Interface
}

// NewListUseCase creates a billing list use case.
func NewListUseCase(repository BillingListRepository, log logger.Interface) ListUseCase {
	if log == nil {
		log = logger.NewNop()
	}

	return &listUseCase{
		repository: repository,
		log:        log.With(logger.Component("billing_list_usecase")),
	}
}

// List normalizes and validates the query before loading billing list items.
func (uc *listUseCase) List(ctx context.Context, query ListQuery) (ListResult, error) {
	if ctx == nil {
		return ListResult{}, logger.ErrNilContext
	}
	if uc.repository == nil {
		return ListResult{}, fmt.Errorf("billing_list_repository is not configured")
	}

	query = query.Normalize()
	if err := query.Validate(); err != nil {
		return ListResult{}, err
	}

	result, err := uc.repository.List(ctx, query)
	if err != nil {
		return ListResult{}, err
	}

	if result.Items == nil {
		result.Items = []ListItem{}
	}
	result.Limit = *query.Limit
	result.Offset = *query.Offset

	return result, nil
}

func cloneBool(value *bool) *bool {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}

func cloneUint(value *uint) *uint {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}
