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

	parsedMissingCurrency := parsed
	parsedMissingCurrency.Currency = nil
	if err := eligibility.Evaluate(parsedMissingCurrency); !errors.Is(err, ErrBillingEligibilityCurrencyEmpty) {
		t.Fatalf("expected ErrBillingEligibilityCurrencyEmpty, got %v", err)
	}

	invalidCurrency := "jp"
	parsedInvalidCurrency := parsed
	parsedInvalidCurrency.Currency = &invalidCurrency
	if err := eligibility.Evaluate(parsedInvalidCurrency); !errors.Is(err, ErrBillingEligibilityCurrencyInvalid) {
		t.Fatalf("expected ErrBillingEligibilityCurrencyInvalid, got %v", err)
	}

	parsedMissingBillingDate := parsed
	parsedMissingBillingDate.BillingDate = nil
	if err := eligibility.Evaluate(parsedMissingBillingDate); !errors.Is(err, ErrBillingEligibilityBillingDateEmpty) {
		t.Fatalf("expected ErrBillingEligibilityBillingDateEmpty, got %v", err)
	}

	parsedMissingPaymentCycle := parsed
	parsedMissingPaymentCycle.PaymentCycle = nil
	if err := eligibility.Evaluate(parsedMissingPaymentCycle); !errors.Is(err, ErrBillingEligibilityPaymentCycleEmpty) {
		t.Fatalf("expected ErrBillingEligibilityPaymentCycleEmpty, got %v", err)
	}

	invalidCycle := "weekly"
	parsedInvalidCycle := parsed
	parsedInvalidCycle.PaymentCycle = &invalidCycle
	if err := eligibility.Evaluate(parsedInvalidCycle); !errors.Is(err, ErrBillingEligibilityPaymentCycleInvalid) {
		t.Fatalf("expected ErrBillingEligibilityPaymentCycleInvalid, got %v", err)
	}

	parsedMissingBillingNumber := parsed
	parsedMissingBillingNumber.BillingNumber = nil
	if err := eligibility.Evaluate(parsedMissingBillingNumber); !errors.Is(err, ErrBillingEligibilityBillingNumberEmpty) {
		t.Fatalf("expected ErrBillingEligibilityBillingNumberEmpty, got %v", err)
	}

	invalidAmount := -10.0
	parsedInvalidAmount := parsed
	parsedInvalidAmount.Amount = &invalidAmount
	if err := eligibility.Evaluate(parsedInvalidAmount); !errors.Is(err, ErrBillingEligibilityAmountInvalid) {
		t.Fatalf("expected ErrBillingEligibilityAmountInvalid, got %v", err)
	}
}
