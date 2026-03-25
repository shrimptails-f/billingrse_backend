package application

import (
	"business/internal/library/logger"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	defaultWorkflowHistoryListLimit  = 20
	defaultWorkflowHistoryListOffset = 0
	maxWorkflowHistoryListLimit      = 100
)

var (
	// ErrInvalidListQuery indicates the workflow history list query is invalid.
	ErrInvalidListQuery = errors.New("manual mail workflow list query is invalid")
)

// StageFailureView is a read model for one persisted workflow stage failure.
type StageFailureView struct {
	ExternalMessageID *string
	ReasonCode        string
	Message           string
	CreatedAt         time.Time
}

// StageSummaryView is the per-stage summary returned by the workflow history list API.
type StageSummaryView struct {
	SuccessCount          int
	BusinessFailureCount  int
	TechnicalFailureCount int
	Failures              []StageFailureView
}

// WorkflowHistoryListItem is one workflow row returned by the list API.
type WorkflowHistoryListItem struct {
	WorkflowID         string
	Provider           string
	AccountIdentifier  string
	LabelName          string
	Since              time.Time
	Until              time.Time
	Status             string
	CurrentStage       *string
	QueuedAt           time.Time
	FinishedAt         *time.Time
	Fetch              StageSummaryView
	Analysis           StageSummaryView
	VendorResolution   StageSummaryView
	BillingEligibility StageSummaryView
	Billing            StageSummaryView
}

// ListQuery is the input contract for the workflow history list API.
type ListQuery struct {
	UserID    uint
	Limit     int
	Offset    int
	Status    *string
	HasLimit  bool
	HasOffset bool
}

// Normalize trims free-form values and applies default pagination.
func (q ListQuery) Normalize() ListQuery {
	q.Status = cloneString(q.Status)
	if q.Status != nil {
		trimmed := strings.TrimSpace(*q.Status)
		q.Status = &trimmed
	}
	if !q.HasLimit {
		q.Limit = defaultWorkflowHistoryListLimit
	}
	if !q.HasOffset {
		q.Offset = defaultWorkflowHistoryListOffset
	}

	return q
}

// Validate checks whether the list query satisfies the API contract.
func (q ListQuery) Validate() error {
	if q.UserID == 0 {
		return fmt.Errorf("%w: user_id is required", ErrInvalidListQuery)
	}
	if q.Limit < 1 || q.Limit > maxWorkflowHistoryListLimit {
		return fmt.Errorf("%w: limit must be between 1 and %d", ErrInvalidListQuery, maxWorkflowHistoryListLimit)
	}
	if q.Offset < 0 {
		return fmt.Errorf("%w: offset must be greater than or equal to zero", ErrInvalidListQuery)
	}
	if q.Status != nil && !isWorkflowStatusFilter(*q.Status) {
		return fmt.Errorf("%w: unsupported status %q", ErrInvalidListQuery, *q.Status)
	}

	return nil
}

// ListResult is the paginated workflow history list result.
type ListResult struct {
	Items      []WorkflowHistoryListItem
	TotalCount int64
}

// WorkflowHistoryListRepository loads workflow history read models.
type WorkflowHistoryListRepository interface {
	List(ctx context.Context, query ListQuery) (ListResult, error)
}

// ListUseCase loads manual mail workflow history items for the authenticated user.
type ListUseCase interface {
	List(ctx context.Context, query ListQuery) (ListResult, error)
}

type listUseCase struct {
	repository WorkflowHistoryListRepository
	log        logger.Interface
}

// NewListUseCase creates a workflow history list use case.
func NewListUseCase(repository WorkflowHistoryListRepository, log logger.Interface) ListUseCase {
	if log == nil {
		log = logger.NewNop()
	}

	return &listUseCase{
		repository: repository,
		log:        log.With(logger.Component("manual_mail_workflow_list_usecase")),
	}
}

// List normalizes and validates the query before loading workflow history items.
func (uc *listUseCase) List(ctx context.Context, query ListQuery) (ListResult, error) {
	if ctx == nil {
		return ListResult{}, logger.ErrNilContext
	}
	if uc.repository == nil {
		return ListResult{}, fmt.Errorf("workflow_history_list_repository is not configured")
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
		result.Items = []WorkflowHistoryListItem{}
	}

	return result, nil
}

func isWorkflowStatusFilter(status string) bool {
	switch status {
	case WorkflowStatusQueued,
		WorkflowStatusRunning,
		WorkflowStatusSucceeded,
		WorkflowStatusPartialSuccess,
		WorkflowStatusFailed:
		return true
	default:
		return false
	}
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}
