package application

import (
	"business/internal/billingeligibility/domain"
	commondomain "business/internal/common/domain"
	"business/internal/library/logger"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// EligibilityTarget is a resolved vendor item evaluated for billing creation.
type EligibilityTarget struct {
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	VendorID          uint
	VendorName        string
	MatchedBy         string
	Data              commondomain.ParsedEmail
}

// Normalize trims strings and normalizes ParsedEmail data before evaluation.
func (t EligibilityTarget) Normalize() EligibilityTarget {
	t.ExternalMessageID = strings.TrimSpace(t.ExternalMessageID)
	t.VendorName = strings.TrimSpace(t.VendorName)
	t.MatchedBy = strings.TrimSpace(t.MatchedBy)
	t.Data = t.Data.Normalize()
	return t
}

// Validate checks the minimum contract required from the previous stage.
func (t EligibilityTarget) Validate() error {
	if t.ParsedEmailID == 0 {
		return errors.New("parsed_email_id is required")
	}
	if t.EmailID == 0 {
		return errors.New("email_id is required")
	}
	if t.VendorID == 0 {
		return errors.New("vendor_id is required")
	}
	if t.VendorName == "" {
		return errors.New("vendor_name is required")
	}
	return nil
}

// Command is the billingeligibility stage input.
type Command struct {
	UserID        uint
	ResolvedItems []EligibilityTarget
}

// Result is the billingeligibility stage output.
type Result struct {
	EligibleItems   []domain.EligibleItem
	EligibleCount   int
	IneligibleItems []domain.IneligibleItem
	IneligibleCount int
	Failures        []domain.Failure
}

// UseCase evaluates whether resolved items can become billings.
type UseCase interface {
	Execute(ctx context.Context, cmd Command) (Result, error)
}

type useCase struct {
	policy commondomain.BillingEligibility
	log    logger.Interface
}

// NewUseCase creates a billingeligibility usecase.
func NewUseCase(log logger.Interface) UseCase {
	if log == nil {
		log = logger.NewNop()
	}

	return &useCase{
		policy: commondomain.BillingEligibility{},
		log:    log.With(logger.Component("billing_eligibility_usecase")),
	}
}

// Execute evaluates each resolved item and classifies it as eligible, ineligible, or failure.
func (uc *useCase) Execute(ctx context.Context, cmd Command) (Result, error) {
	if ctx == nil {
		return Result{}, logger.ErrNilContext
	}
	if err := validateCommand(cmd); err != nil {
		return Result{}, err
	}
	if len(cmd.ResolvedItems) == 0 {
		return Result{}, nil
	}

	reqLog := uc.log
	if withContext, err := uc.log.WithContext(ctx); err == nil {
		reqLog = withContext
	}

	result := Result{}
	for _, target := range cmd.ResolvedItems {
		target = target.Normalize()
		if err := target.Validate(); err != nil {
			result.Failures = append(result.Failures, domain.Failure{
				ParsedEmailID:     target.ParsedEmailID,
				EmailID:           target.EmailID,
				ExternalMessageID: target.ExternalMessageID,
				Stage:             domain.FailureStageNormalizeInput,
				Code:              domain.FailureCodeInvalidEligibilityTarget,
			})
			continue
		}

		resolution := commondomain.VendorResolution{
			ResolvedVendor: &commondomain.Vendor{
				ID:   target.VendorID,
				Name: target.VendorName,
			},
		}
		err := uc.policy.Evaluate(target.Data, resolution)
		if err == nil {
			result.EligibleItems = append(result.EligibleItems, domain.EligibleItem{
				ParsedEmailID:     target.ParsedEmailID,
				EmailID:           target.EmailID,
				ExternalMessageID: target.ExternalMessageID,
				VendorID:          target.VendorID,
				VendorName:        target.VendorName,
				MatchedBy:         target.MatchedBy,
				BillingNumber:     stringValue(target.Data.BillingNumber),
				InvoiceNumber:     cloneString(target.Data.InvoiceNumber),
				Amount:            float64Value(target.Data.Amount),
				Currency:          stringValue(target.Data.Currency),
				BillingDate:       cloneTime(target.Data.BillingDate),
				PaymentCycle:      stringValue(target.Data.PaymentCycle),
			})
			continue
		}

		if reasonCode, ok := reasonCodeForError(err); ok {
			result.IneligibleItems = append(result.IneligibleItems, domain.IneligibleItem{
				ParsedEmailID:     target.ParsedEmailID,
				EmailID:           target.EmailID,
				ExternalMessageID: target.ExternalMessageID,
				VendorID:          target.VendorID,
				VendorName:        target.VendorName,
				MatchedBy:         target.MatchedBy,
				ReasonCode:        reasonCode,
			})
			continue
		}

		result.Failures = append(result.Failures, domain.Failure{
			ParsedEmailID:     target.ParsedEmailID,
			EmailID:           target.EmailID,
			ExternalMessageID: target.ExternalMessageID,
			Stage:             domain.FailureStageEvaluateEligibility,
			Code:              domain.FailureCodeBillingEligibilityFail,
		})
	}

	result.EligibleCount = len(result.EligibleItems)
	result.IneligibleCount = len(result.IneligibleItems)

	reqLog.Info("billing_eligibility_succeeded",
		logger.UserID(cmd.UserID),
		logger.Int("input_resolved_item_count", len(cmd.ResolvedItems)),
		logger.Int("eligible_count", result.EligibleCount),
		logger.Int("ineligible_count", result.IneligibleCount),
		logger.Int("failure_count", len(result.Failures)),
	)

	return result, nil
}

func validateCommand(cmd Command) error {
	if cmd.UserID == 0 {
		return fmt.Errorf("%w: user_id is required", domain.ErrInvalidCommand)
	}
	return nil
}

func reasonCodeForError(err error) (string, bool) {
	switch {
	case errors.Is(err, commondomain.ErrBillingEligibilityAmountEmpty):
		return domain.ReasonCodeAmountEmpty, true
	case errors.Is(err, commondomain.ErrBillingEligibilityAmountInvalid):
		return domain.ReasonCodeAmountInvalid, true
	case errors.Is(err, commondomain.ErrBillingEligibilityCurrencyEmpty):
		return domain.ReasonCodeCurrencyEmpty, true
	case errors.Is(err, commondomain.ErrBillingEligibilityCurrencyInvalid):
		return domain.ReasonCodeCurrencyInvalid, true
	case errors.Is(err, commondomain.ErrBillingEligibilityBillingNumberEmpty):
		return domain.ReasonCodeBillingNumberEmpty, true
	case errors.Is(err, commondomain.ErrBillingEligibilityPaymentCycleEmpty):
		return domain.ReasonCodePaymentCycleEmpty, true
	case errors.Is(err, commondomain.ErrBillingEligibilityPaymentCycleInvalid):
		return domain.ReasonCodePaymentCycleInvalid, true
	default:
		return "", false
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func float64Value(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
