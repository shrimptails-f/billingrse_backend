package application

import (
	billingdomain "business/internal/billing/domain"
	commondomain "business/internal/common/domain"
	"business/internal/library/logger"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type stubBillingRepository struct {
	saveIfAbsent func(ctx context.Context, billing commondomain.Billing) (SaveResult, error)
}

func (s *stubBillingRepository) SaveIfAbsent(ctx context.Context, billing commondomain.Billing) (SaveResult, error) {
	return s.saveIfAbsent(ctx, billing)
}

func TestUseCaseExecute_CreatedDuplicateAndFailures(t *testing.T) {
	t.Parallel()

	invoiceNumber := "t1234567890123"
	billingDate := time.Date(2026, 3, 24, 10, 30, 0, 0, time.UTC)
	productNameDisplay := " Example Product "

	uc := NewUseCase(&stubBillingRepository{
		saveIfAbsent: func(ctx context.Context, billing commondomain.Billing) (SaveResult, error) {
			switch billing.BillingNumber.String() {
			case "INV-100":
				if billing.InvoiceNumber.String() != "T1234567890123" {
					t.Fatalf("expected normalized invoice number, got %+v", billing)
				}
				if billing.Money.Currency != "JPY" {
					t.Fatalf("expected normalized currency, got %+v", billing)
				}
				if billing.PaymentCycle.String() != "recurring" {
					t.Fatalf("expected normalized payment cycle, got %+v", billing)
				}
				if billing.BillingDate == nil || !billing.BillingDate.Equal(billingDate) {
					t.Fatalf("expected billing date to be preserved, got %+v", billing)
				}
				if billing.ProductNameDisplay == nil || *billing.ProductNameDisplay != "Example Product" {
					t.Fatalf("expected normalized product name display, got %+v", billing)
				}
				return SaveResult{BillingID: 9001}, nil
			case "INV-200":
				return SaveResult{BillingID: 8001, Duplicate: true}, nil
			case "INV-500":
				return SaveResult{}, errors.New("db unavailable")
			default:
				t.Fatalf("unexpected billing persisted: %+v", billing)
				return SaveResult{}, nil
			}
		},
	}, logger.NewNop())

	result, err := uc.Execute(context.Background(), Command{
		UserID: 7,
		EligibleItems: []CreationTarget{
			{
				ParsedEmailID:      101,
				EmailID:            201,
				ExternalMessageID:  "msg-1",
				VendorID:           301,
				VendorName:         "Acme",
				MatchedBy:          "name_exact",
				ProductNameDisplay: &productNameDisplay,
				BillingNumber:      " INV-100 ",
				InvoiceNumber:      &invoiceNumber,
				Amount:             1200.5,
				Currency:           " jpy ",
				BillingDate:        &billingDate,
				PaymentCycle:       " recurring ",
			},
			{
				ParsedEmailID:     102,
				EmailID:           202,
				ExternalMessageID: "msg-2",
				VendorID:          302,
				VendorName:        "Beta",
				MatchedBy:         "sender_domain",
				BillingNumber:     "INV-200",
				Amount:            99.9,
				Currency:          "USD",
				PaymentCycle:      "one_time",
			},
			{
				ParsedEmailID:     103,
				EmailID:           203,
				ExternalMessageID: "msg-3",
				VendorID:          0,
				VendorName:        "Gamma",
				BillingNumber:     "INV-300",
				Amount:            1,
				Currency:          "JPY",
				PaymentCycle:      "one_time",
			},
			{
				ParsedEmailID:     104,
				EmailID:           204,
				ExternalMessageID: "msg-4",
				VendorID:          304,
				VendorName:        "Delta",
				BillingNumber:     "INV-400",
				Amount:            0,
				Currency:          "JPY",
				PaymentCycle:      "one_time",
			},
			{
				ParsedEmailID:     105,
				EmailID:           205,
				ExternalMessageID: "msg-5",
				VendorID:          305,
				VendorName:        "Echo",
				BillingNumber:     "INV-500",
				Amount:            10,
				Currency:          "USD",
				PaymentCycle:      "one_time",
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if result.CreatedCount != 1 || len(result.CreatedItems) != 1 {
		t.Fatalf("unexpected created result: %+v", result)
	}
	if result.CreatedItems[0].BillingID != 9001 {
		t.Fatalf("unexpected created item: %+v", result.CreatedItems[0])
	}
	if result.DuplicateCount != 1 || len(result.DuplicateItems) != 1 {
		t.Fatalf("unexpected duplicate result: %+v", result)
	}
	if result.DuplicateItems[0].ExistingBillingID != 8001 {
		t.Fatalf("unexpected duplicate item: %+v", result.DuplicateItems[0])
	}
	if result.DuplicateItems[0].ReasonCode != billingdomain.ReasonCodeDuplicateBilling {
		t.Fatalf("unexpected duplicate reason code: %+v", result.DuplicateItems[0])
	}
	if !strings.Contains(result.DuplicateItems[0].Message, "Beta") ||
		!strings.Contains(result.DuplicateItems[0].Message, "INV-200") ||
		!strings.Contains(result.DuplicateItems[0].Message, "msg-2") {
		t.Fatalf("unexpected duplicate message: %+v", result.DuplicateItems[0])
	}
	if len(result.Failures) != 3 {
		t.Fatalf("expected 3 failures, got %+v", result.Failures)
	}
	if result.Failures[0].Stage != "normalize_input" || result.Failures[0].Code != "invalid_creation_target" {
		t.Fatalf("unexpected normalize failure: %+v", result.Failures[0])
	}
	if !strings.Contains(result.Failures[0].Message, "Gamma") ||
		!strings.Contains(result.Failures[0].Message, "INV-300") ||
		!strings.Contains(result.Failures[0].Message, "msg-3") {
		t.Fatalf("unexpected normalize failure message: %+v", result.Failures[0])
	}
	if result.Failures[1].Stage != "build_billing" || result.Failures[1].Code != "billing_construct_failed" {
		t.Fatalf("unexpected build failure: %+v", result.Failures[1])
	}
	if !strings.Contains(result.Failures[1].Message, "Delta") ||
		!strings.Contains(result.Failures[1].Message, "INV-400") ||
		!strings.Contains(result.Failures[1].Message, "msg-4") {
		t.Fatalf("unexpected build failure message: %+v", result.Failures[1])
	}
	if result.Failures[2].Stage != "save_billing" || result.Failures[2].Code != "billing_persist_failed" {
		t.Fatalf("unexpected save failure: %+v", result.Failures[2])
	}
	if !strings.Contains(result.Failures[2].Message, "Echo") ||
		!strings.Contains(result.Failures[2].Message, "INV-500") ||
		!strings.Contains(result.Failures[2].Message, "msg-5") {
		t.Fatalf("unexpected save failure message: %+v", result.Failures[2])
	}
}

func TestUseCaseExecute_AllowsNilBillingDate(t *testing.T) {
	t.Parallel()

	repositoryCalled := false
	uc := NewUseCase(&stubBillingRepository{
		saveIfAbsent: func(ctx context.Context, billing commondomain.Billing) (SaveResult, error) {
			repositoryCalled = true
			if billing.BillingDate != nil {
				t.Fatalf("expected nil billing date, got %+v", billing)
			}
			return SaveResult{BillingID: 9010}, nil
		},
	}, logger.NewNop())

	result, err := uc.Execute(context.Background(), Command{
		UserID: 1,
		EligibleItems: []CreationTarget{
			{
				ParsedEmailID:      10,
				EmailID:            20,
				ExternalMessageID:  "msg-10",
				VendorID:           30,
				VendorName:         "Acme",
				ProductNameDisplay: nil,
				BillingNumber:      "INV-010",
				Amount:             100,
				Currency:           "JPY",
				PaymentCycle:       "one_time",
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !repositoryCalled {
		t.Fatal("expected repository to be called")
	}
	if result.CreatedCount != 1 || result.CreatedItems[0].BillingID != 9010 {
		t.Fatalf("unexpected result: %+v", result)
	}
}
