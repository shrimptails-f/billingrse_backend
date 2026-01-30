package domain

import (
	"errors"
	"strings"
)

var (
	// ErrBillingEligibilityVendorNameEmpty is returned when the vendor name is missing.
	ErrBillingEligibilityVendorNameEmpty = errors.New("billing eligibility vendor name is empty")
	// ErrBillingEligibilityAmountEmpty is returned when the amount is missing.
	ErrBillingEligibilityAmountEmpty = errors.New("billing eligibility amount is empty")
	// ErrBillingEligibilityAmountInvalid is returned when the amount is invalid.
	ErrBillingEligibilityAmountInvalid = errors.New("billing eligibility amount is invalid")
	// ErrBillingEligibilityCurrencyEmpty is returned when the currency is missing.
	ErrBillingEligibilityCurrencyEmpty = errors.New("billing eligibility currency is empty")
	// ErrBillingEligibilityCurrencyInvalid is returned when the currency is invalid.
	ErrBillingEligibilityCurrencyInvalid = errors.New("billing eligibility currency is invalid")
	// ErrBillingEligibilityBillingDateEmpty is returned when the billing date is missing.
	ErrBillingEligibilityBillingDateEmpty = errors.New("billing eligibility billing date is empty")
	// ErrBillingEligibilityPaymentCycleEmpty is returned when the payment cycle is missing.
	ErrBillingEligibilityPaymentCycleEmpty = errors.New("billing eligibility payment cycle is empty")
	// ErrBillingEligibilityPaymentCycleInvalid is returned when the payment cycle is invalid.
	ErrBillingEligibilityPaymentCycleInvalid = errors.New("billing eligibility payment cycle is invalid")
	// ErrBillingEligibilityBillingNumberEmpty is returned when the billing number is missing.
	ErrBillingEligibilityBillingNumberEmpty = errors.New("billing eligibility billing number is empty")
)

// BillingEligibility represents the policy to determine whether billing can be created.
type BillingEligibility struct{}

// Evaluate checks whether the parsed email satisfies the billing requirements.
// It returns the first rule violation encountered.
func (BillingEligibility) Evaluate(parsed ParsedEmail) error {
	if parsed.VendorName == nil || strings.TrimSpace(*parsed.VendorName) == "" {
		return ErrBillingEligibilityVendorNameEmpty
	}
	if parsed.Amount == nil {
		return ErrBillingEligibilityAmountEmpty
	}
	if _, err := NormalizeAmount(*parsed.Amount); err != nil {
		return ErrBillingEligibilityAmountInvalid
	}
	if parsed.Currency == nil || strings.TrimSpace(*parsed.Currency) == "" {
		return ErrBillingEligibilityCurrencyEmpty
	}
	if _, err := NormalizeCurrency(*parsed.Currency); err != nil {
		return ErrBillingEligibilityCurrencyInvalid
	}
	if parsed.BillingDate == nil || parsed.BillingDate.IsZero() {
		return ErrBillingEligibilityBillingDateEmpty
	}
	if parsed.PaymentCycle == nil || strings.TrimSpace(*parsed.PaymentCycle) == "" {
		return ErrBillingEligibilityPaymentCycleEmpty
	}
	if _, err := NewPaymentCycle(*parsed.PaymentCycle); err != nil {
		return ErrBillingEligibilityPaymentCycleInvalid
	}
	if parsed.BillingNumber == nil || strings.TrimSpace(*parsed.BillingNumber) == "" {
		return ErrBillingEligibilityBillingNumberEmpty
	}
	return nil
}

// IsEligible reports whether the parsed email passes the billing eligibility rules.
func (e BillingEligibility) IsEligible(parsed ParsedEmail) bool {
	return e.Evaluate(parsed) == nil
}
