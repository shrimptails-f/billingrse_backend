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
				Code:              domain.FailureCodeInvalidEligibilityTarget,
				Message:           messageForEligibilityFailure(domain.FailureCodeInvalidEligibilityTarget, target),
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
				ParsedEmailID:      target.ParsedEmailID,
				EmailID:            target.EmailID,
				ExternalMessageID:  target.ExternalMessageID,
				VendorID:           target.VendorID,
				VendorName:         target.VendorName,
				MatchedBy:          target.MatchedBy,
				ProductNameDisplay: uc.policy.ResolvedProductNameDisplay(target.Data),
				BillingNumber:      stringValue(target.Data.BillingNumber),
				InvoiceNumber:      cloneString(target.Data.InvoiceNumber),
				Amount:             float64Value(target.Data.Amount),
				Currency:           stringValue(target.Data.Currency),
				BillingDate:        cloneTime(target.Data.BillingDate),
				PaymentCycle:       stringValue(target.Data.PaymentCycle),
				LineItems:          toLineItems(target.Data.LineItems, stringValue(target.Data.Currency)),
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
				Message:           messageForEligibilityReason(reasonCode, target),
			})
			continue
		}

		result.Failures = append(result.Failures, domain.Failure{
			ParsedEmailID:     target.ParsedEmailID,
			EmailID:           target.EmailID,
			ExternalMessageID: target.ExternalMessageID,
			Code:              domain.FailureCodeBillingEligibilityFail,
			Message:           messageForEligibilityFailure(domain.FailureCodeBillingEligibilityFail, target),
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
	case errors.Is(err, commondomain.ErrBillingEligibilityProductNameEmpty):
		return domain.ReasonCodeProductNameEmpty, true
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

func messageForEligibilityReason(reasonCode string, target EligibilityTarget) string {
	vendorName := strings.TrimSpace(target.VendorName)
	if vendorName == "" {
		vendorName = "不明の支払先"
	}
	externalMessageID := strings.TrimSpace(target.ExternalMessageID)
	suffix := ""
	if externalMessageID != "" {
		suffix = " 対象メールID: " + externalMessageID
	}

	switch reasonCode {
	case domain.ReasonCodeProductNameEmpty:
		return vendorName + " の請求候補で商品名が不足しているため、請求を作成できませんでした。" + suffix
	case domain.ReasonCodeAmountEmpty:
		return vendorName + " の請求候補で金額が不足しているため、請求を作成できませんでした。" + suffix
	case domain.ReasonCodeAmountInvalid:
		return vendorName + " の請求候補で金額が不正なため、請求を作成できませんでした。" + suffix
	case domain.ReasonCodeCurrencyEmpty:
		return vendorName + " の請求候補で通貨が不足しているため、請求を作成できませんでした。" + suffix
	case domain.ReasonCodeCurrencyInvalid:
		return vendorName + " の請求候補で通貨が不正なため、請求を作成できませんでした。" + suffix
	case domain.ReasonCodeBillingNumberEmpty:
		return vendorName + " の請求候補で請求番号が不足しているため、請求を作成できませんでした。" + suffix
	case domain.ReasonCodePaymentCycleEmpty:
		return vendorName + " の請求候補で支払周期が不足しているため、請求を作成できませんでした。" + suffix
	case domain.ReasonCodePaymentCycleInvalid:
		return vendorName + " の請求候補で支払周期が不正なため、請求を作成できませんでした。" + suffix
	default:
		return vendorName + " の請求候補が請求成立条件を満たさないため、請求を作成できませんでした。" + suffix
	}
}

func messageForEligibilityFailure(code string, target EligibilityTarget) string {
	vendorName := strings.TrimSpace(target.VendorName)
	if vendorName == "" {
		vendorName = "不明の支払先"
	}
	externalMessageID := strings.TrimSpace(target.ExternalMessageID)
	suffix := ""
	if externalMessageID != "" {
		suffix = " 対象メールID: " + externalMessageID
	}

	switch code {
	case domain.FailureCodeInvalidEligibilityTarget:
		return vendorName + " の請求成立判定入力が不正でした。" + suffix
	case domain.FailureCodeBillingEligibilityFail:
		return vendorName + " の請求成立判定中に予期しないエラーが発生しました。" + suffix
	default:
		return vendorName + " の請求成立判定中にエラーが発生しました。" + suffix
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

func cloneFloat64(value *float64) *float64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func normalizeCurrencyPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	upper := strings.ToUpper(trimmed)
	return &upper
}

func toLineItems(items []commondomain.ParsedEmailLineItem, fallbackCurrency string) []domain.LineItem {
	if len(items) == 0 {
		return nil
	}

	fallbackCurrency = strings.TrimSpace(strings.ToUpper(fallbackCurrency))
	lineItems := make([]domain.LineItem, 0, len(items))
	for _, item := range items {
		normalized := item.Normalize()
		currency := normalizeCurrencyPtr(normalized.Currency)
		if currency == nil && fallbackCurrency != "" {
			currency = cloneString(&fallbackCurrency)
		}

		lineItem := domain.LineItem{
			ProductNameRaw:     cloneString(normalized.ProductNameRaw),
			ProductNameDisplay: cloneString(normalized.ProductNameDisplay),
			Amount:             cloneFloat64(normalized.Amount),
			Currency:           currency,
		}
		if lineItem.ProductNameRaw == nil &&
			lineItem.ProductNameDisplay == nil &&
			lineItem.Amount == nil &&
			lineItem.Currency == nil {
			continue
		}
		lineItems = append(lineItems, lineItem)
	}

	if len(lineItems) == 0 {
		return nil
	}
	return lineItems
}
