package domain

import (
	"errors"
	"testing"
	"time"
)

func TestBillingEligibility(t *testing.T) {
	t.Parallel()

	vendorName := "Netflix"
	billingNumber := "INV-001"
	amount := 1200.0
	currency := "JPY"
	billingDate := time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC)
	paymentCycle := "one_time"

	parsed := ParsedEmail{
		VendorName:    &vendorName,
		BillingNumber: &billingNumber,
		Amount:        &amount,
		Currency:      &currency,
		BillingDate:   &billingDate,
		PaymentCycle:  &paymentCycle,
	}

	resolution := VendorResolution{
		ResolvedVendor: &Vendor{Name: "Netflix"},
	}

	eligibility := BillingEligibility{}
	if !eligibility.IsEligible(parsed, resolution) {
		t.Fatalf("expected parsed email to be eligible")
	}
	if err := eligibility.Evaluate(parsed, resolution); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	parsedMissingRawVendor := parsed
	parsedMissingRawVendor.VendorName = nil
	if err := eligibility.Evaluate(parsedMissingRawVendor, resolution); err != nil {
		t.Fatalf("expected no error when resolved vendor exists, got %v", err)
	}

	if err := eligibility.Evaluate(parsed, VendorResolution{}); !errors.Is(err, ErrBillingEligibilityVendorUnresolved) {
		t.Fatalf("expected ErrBillingEligibilityVendorUnresolved, got %v", err)
	}

	parsedMissingAmount := parsed
	parsedMissingAmount.Amount = nil
	if err := eligibility.Evaluate(parsedMissingAmount, resolution); !errors.Is(err, ErrBillingEligibilityAmountEmpty) {
		t.Fatalf("expected ErrBillingEligibilityAmountEmpty, got %v", err)
	}

	parsedMissingCurrency := parsed
	parsedMissingCurrency.Currency = nil
	if err := eligibility.Evaluate(parsedMissingCurrency, resolution); !errors.Is(err, ErrBillingEligibilityCurrencyEmpty) {
		t.Fatalf("expected ErrBillingEligibilityCurrencyEmpty, got %v", err)
	}

	invalidCurrency := "jp"
	parsedInvalidCurrency := parsed
	parsedInvalidCurrency.Currency = &invalidCurrency
	if err := eligibility.Evaluate(parsedInvalidCurrency, resolution); !errors.Is(err, ErrBillingEligibilityCurrencyInvalid) {
		t.Fatalf("expected ErrBillingEligibilityCurrencyInvalid, got %v", err)
	}

	parsedMissingBillingDate := parsed
	parsedMissingBillingDate.BillingDate = nil
	if err := eligibility.Evaluate(parsedMissingBillingDate, resolution); err != nil {
		t.Fatalf("expected nil billing date to be allowed, got %v", err)
	}

	parsedMissingPaymentCycle := parsed
	parsedMissingPaymentCycle.PaymentCycle = nil
	if err := eligibility.Evaluate(parsedMissingPaymentCycle, resolution); !errors.Is(err, ErrBillingEligibilityPaymentCycleEmpty) {
		t.Fatalf("expected ErrBillingEligibilityPaymentCycleEmpty, got %v", err)
	}

	invalidCycle := "weekly"
	parsedInvalidCycle := parsed
	parsedInvalidCycle.PaymentCycle = &invalidCycle
	if err := eligibility.Evaluate(parsedInvalidCycle, resolution); !errors.Is(err, ErrBillingEligibilityPaymentCycleInvalid) {
		t.Fatalf("expected ErrBillingEligibilityPaymentCycleInvalid, got %v", err)
	}

	parsedMissingBillingNumber := parsed
	parsedMissingBillingNumber.BillingNumber = nil
	if err := eligibility.Evaluate(parsedMissingBillingNumber, resolution); !errors.Is(err, ErrBillingEligibilityBillingNumberEmpty) {
		t.Fatalf("expected ErrBillingEligibilityBillingNumberEmpty, got %v", err)
	}

	invalidAmount := -10.0
	parsedInvalidAmount := parsed
	parsedInvalidAmount.Amount = &invalidAmount
	if err := eligibility.Evaluate(parsedInvalidAmount, resolution); !errors.Is(err, ErrBillingEligibilityAmountInvalid) {
		t.Fatalf("expected ErrBillingEligibilityAmountInvalid, got %v", err)
	}
}
