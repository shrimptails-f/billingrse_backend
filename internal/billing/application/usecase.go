package application

import (
	"business/internal/billing/domain"
	commondomain "business/internal/common/domain"
	"business/internal/library/logger"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// SaveResult is the repository result for idempotent billing persistence.
type SaveResult struct {
	BillingID uint
	Duplicate bool
}

// BillingRepository persists billings idempotently by billing identity.
type BillingRepository interface {
	SaveIfAbsent(ctx context.Context, billing commondomain.Billing) (SaveResult, error)
}

// CreationTarget is a billing-ready item received from billingeligibility.
type CreationTarget struct {
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	VendorID          uint
	VendorName        string
	MatchedBy         string
	BillingNumber     string
	InvoiceNumber     *string
	Amount            float64
	Currency          string
	BillingDate       *time.Time
	PaymentCycle      string
}

// Normalize trims free-form strings and clones optional pointer values.
func (t CreationTarget) Normalize() CreationTarget {
	t.ExternalMessageID = strings.TrimSpace(t.ExternalMessageID)
	t.VendorName = strings.TrimSpace(t.VendorName)
	t.MatchedBy = strings.TrimSpace(t.MatchedBy)
	t.BillingNumber = strings.TrimSpace(t.BillingNumber)
	t.InvoiceNumber = cloneString(t.InvoiceNumber)
	t.Currency = strings.TrimSpace(t.Currency)
	t.PaymentCycle = strings.TrimSpace(t.PaymentCycle)
	t.BillingDate = cloneTime(t.BillingDate)
	return t
}

// Validate checks the minimum contract required from the previous stage.
func (t CreationTarget) Validate() error {
	if t.ParsedEmailID == 0 {
		return errors.New("parsed_email_id is required")
	}
	if t.EmailID == 0 {
		return errors.New("email_id is required")
	}
	if t.VendorID == 0 {
		return errors.New("vendor_id is required")
	}
	if t.BillingNumber == "" {
		return errors.New("billing_number is required")
	}
	if t.Currency == "" {
		return errors.New("currency is required")
	}
	if t.PaymentCycle == "" {
		return errors.New("payment_cycle is required")
	}
	return nil
}

// Command is the billing stage input.
type Command struct {
	UserID        uint
	EligibleItems []CreationTarget
}

// Result is the billing stage output.
type Result struct {
	CreatedItems   []domain.CreatedItem
	CreatedCount   int
	DuplicateItems []domain.DuplicateItem
	DuplicateCount int
	Failures       []domain.Failure
}

// UseCase creates and persists billings from eligible items.
type UseCase interface {
	Execute(ctx context.Context, cmd Command) (Result, error)
}

type useCase struct {
	repository BillingRepository
	log        logger.Interface
}

// NewUseCase creates a billing usecase.
func NewUseCase(repository BillingRepository, log logger.Interface) UseCase {
	if log == nil {
		log = logger.NewNop()
	}

	return &useCase{
		repository: repository,
		log:        log.With(logger.Component("billing_usecase")),
	}
}

// Execute builds Billing aggregates and persists them idempotently.
func (uc *useCase) Execute(ctx context.Context, cmd Command) (Result, error) {
	if ctx == nil {
		return Result{}, logger.ErrNilContext
	}
	if err := validateCommand(cmd); err != nil {
		return Result{}, err
	}
	if len(cmd.EligibleItems) == 0 {
		return Result{}, nil
	}
	if err := uc.validateDependencies(); err != nil {
		return Result{}, err
	}

	reqLog := uc.log
	if withContext, err := uc.log.WithContext(ctx); err == nil {
		reqLog = withContext
	}

	result := Result{}
	for _, target := range cmd.EligibleItems {
		target = target.Normalize()
		if err := target.Validate(); err != nil {
			result.Failures = append(result.Failures, domain.Failure{
				ParsedEmailID:     target.ParsedEmailID,
				EmailID:           target.EmailID,
				ExternalMessageID: target.ExternalMessageID,
				Stage:             domain.FailureStageNormalizeInput,
				Code:              domain.FailureCodeInvalidCreationTarget,
			})
			continue
		}

		billing, err := commondomain.NewBilling(
			cmd.UserID,
			target.VendorID,
			target.EmailID,
			target.BillingNumber,
			target.InvoiceNumber,
			target.Amount,
			target.Currency,
			target.BillingDate,
			target.PaymentCycle,
		)
		if err != nil {
			result.Failures = append(result.Failures, domain.Failure{
				ParsedEmailID:     target.ParsedEmailID,
				EmailID:           target.EmailID,
				ExternalMessageID: target.ExternalMessageID,
				Stage:             domain.FailureStageBuildBilling,
				Code:              domain.FailureCodeBillingConstructFailed,
			})
			continue
		}

		saveResult, err := uc.repository.SaveIfAbsent(ctx, billing)
		if err != nil {
			result.Failures = append(result.Failures, domain.Failure{
				ParsedEmailID:     target.ParsedEmailID,
				EmailID:           target.EmailID,
				ExternalMessageID: target.ExternalMessageID,
				Stage:             domain.FailureStageSaveBilling,
				Code:              domain.FailureCodeBillingPersistFailed,
			})
			continue
		}

		if saveResult.Duplicate {
			result.DuplicateItems = append(result.DuplicateItems, domain.DuplicateItem{
				ExistingBillingID: saveResult.BillingID,
				ParsedEmailID:     target.ParsedEmailID,
				EmailID:           target.EmailID,
				ExternalMessageID: target.ExternalMessageID,
				VendorID:          target.VendorID,
				VendorName:        target.VendorName,
				BillingNumber:     billing.BillingNumber.String(),
			})
			continue
		}

		result.CreatedItems = append(result.CreatedItems, domain.CreatedItem{
			BillingID:         saveResult.BillingID,
			ParsedEmailID:     target.ParsedEmailID,
			EmailID:           target.EmailID,
			ExternalMessageID: target.ExternalMessageID,
			VendorID:          target.VendorID,
			VendorName:        target.VendorName,
			BillingNumber:     billing.BillingNumber.String(),
		})
	}

	result.CreatedCount = len(result.CreatedItems)
	result.DuplicateCount = len(result.DuplicateItems)

	reqLog.Info("billing_succeeded",
		logger.UserID(cmd.UserID),
		logger.Int("input_eligible_item_count", len(cmd.EligibleItems)),
		logger.Int("created_count", result.CreatedCount),
		logger.Int("duplicate_count", result.DuplicateCount),
		logger.Int("failure_count", len(result.Failures)),
	)

	return result, nil
}

func (uc *useCase) validateDependencies() error {
	if uc.repository == nil {
		return errors.New("billing_repository is not configured")
	}
	return nil
}

func validateCommand(cmd Command) error {
	if cmd.UserID == 0 {
		return fmt.Errorf("%w: user_id is required", domain.ErrInvalidCommand)
	}
	return nil
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := strings.TrimSpace(*value)
	return &cloned
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := value.UTC()
	return &cloned
}
