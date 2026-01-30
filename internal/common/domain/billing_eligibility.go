package domain

import (
	"errors"
	"math"
	"strings"
)

var (
	// ErrBillingEligibilityVendorNameEmpty is returned when the vendor name is missing.
	ErrBillingEligibilityVendorNameEmpty = errors.New("billing eligibility vendor name is empty")
	// ErrBillingEligibilityAmountEmpty is returned when the amount is missing.
	ErrBillingEligibilityAmountEmpty = errors.New("billing eligibility amount is empty")
	// ErrBillingEligibilityAmountInvalid is returned when the amount is invalid.
	ErrBillingEligibilityAmountInvalid = errors.New("billing eligibility amount is invalid")
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
	if math.IsNaN(*parsed.Amount) || math.IsInf(*parsed.Amount, 0) || *parsed.Amount <= 0 {
		return ErrBillingEligibilityAmountInvalid
	}
	return nil
}

// IsEligible reports whether the parsed email passes the billing eligibility rules.
func (e BillingEligibility) IsEligible(parsed ParsedEmail) bool {
	return e.Evaluate(parsed) == nil
}
