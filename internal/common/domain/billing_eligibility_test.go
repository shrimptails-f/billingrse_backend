package domain

import (
	"errors"
	"testing"
)

func TestBillingEligibility(t *testing.T) {
	t.Parallel()

	vendorName := "Netflix"
	amount := 1200.0

	parsed := ParsedEmail{
		VendorName: &vendorName,
		Amount:     &amount,
	}

	eligibility := BillingEligibility{}
	if !eligibility.IsEligible(parsed) {
		t.Fatalf("expected parsed email to be eligible")
	}
	if err := eligibility.Evaluate(parsed); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	parsedMissingVendor := parsed
	parsedMissingVendor.VendorName = nil
	if err := eligibility.Evaluate(parsedMissingVendor); !errors.Is(err, ErrBillingEligibilityVendorNameEmpty) {
		t.Fatalf("expected ErrBillingEligibilityVendorNameEmpty, got %v", err)
	}

	parsedMissingAmount := parsed
	parsedMissingAmount.Amount = nil
	if err := eligibility.Evaluate(parsedMissingAmount); !errors.Is(err, ErrBillingEligibilityAmountEmpty) {
		t.Fatalf("expected ErrBillingEligibilityAmountEmpty, got %v", err)
	}

	invalidAmount := -10.0
	parsedInvalidAmount := parsed
	parsedInvalidAmount.Amount = &invalidAmount
	if err := eligibility.Evaluate(parsedInvalidAmount); !errors.Is(err, ErrBillingEligibilityAmountInvalid) {
		t.Fatalf("expected ErrBillingEligibilityAmountInvalid, got %v", err)
	}
}
