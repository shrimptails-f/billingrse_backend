package infrastructure

import (
	billingapp "business/internal/billing/application"
	billingdomain "business/internal/billing/domain"
	manualapp "business/internal/manualmailworkflow/application"
	"context"
	"testing"
	"time"
)

type stubBillingUseCase struct {
	execute func(ctx context.Context, cmd billingapp.Command) (billingapp.Result, error)
}

func (s *stubBillingUseCase) Execute(ctx context.Context, cmd billingapp.Command) (billingapp.Result, error) {
	return s.execute(ctx, cmd)
}

func TestDirectBillingAdapter_Execute_ConvertsCommandAndResult(t *testing.T) {
	t.Parallel()

	invoiceNumber := "T1234567890123"
	billingDate := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)
	productNameDisplay := "Example Product"
	lineItemName := "Example Product Full Name"
	adapter := NewDirectBillingAdapter(&stubBillingUseCase{
		execute: func(ctx context.Context, cmd billingapp.Command) (billingapp.Result, error) {
			if len(cmd.EligibleItems) != 1 {
				t.Fatalf("unexpected command: %+v", cmd)
			}
			if cmd.EligibleItems[0].InvoiceNumber == nil || *cmd.EligibleItems[0].InvoiceNumber != invoiceNumber {
				t.Fatalf("expected invoice number in target, got %+v", cmd.EligibleItems[0])
			}
			if cmd.EligibleItems[0].BillingDate == nil || !cmd.EligibleItems[0].BillingDate.Equal(billingDate) {
				t.Fatalf("expected billing date in target, got %+v", cmd.EligibleItems[0])
			}
			if cmd.EligibleItems[0].ProductNameDisplay == nil || *cmd.EligibleItems[0].ProductNameDisplay != productNameDisplay {
				t.Fatalf("expected product name display in target, got %+v", cmd.EligibleItems[0])
			}
			if len(cmd.EligibleItems[0].LineItems) != 1 {
				t.Fatalf("expected line items in target, got %+v", cmd.EligibleItems[0])
			}
			if cmd.EligibleItems[0].LineItems[0].ProductNameRaw == nil || *cmd.EligibleItems[0].LineItems[0].ProductNameRaw != lineItemName {
				t.Fatalf("expected line item product name in target, got %+v", cmd.EligibleItems[0].LineItems[0])
			}
			return billingapp.Result{
				CreatedItems: []billingdomain.CreatedItem{
					{
						BillingID:         7001,
						ParsedEmailID:     9001,
						EmailID:           101,
						ExternalMessageID: "msg-1",
						VendorID:          3001,
						VendorName:        "Acme",
						BillingNumber:     "INV-001",
					},
				},
				CreatedCount: 1,
				DuplicateItems: []billingdomain.DuplicateItem{
					{
						ExistingBillingID: 7000,
						ParsedEmailID:     9002,
						EmailID:           102,
						ExternalMessageID: "msg-2",
						VendorID:          3001,
						VendorName:        "Acme",
						BillingNumber:     "INV-002",
						ReasonCode:        billingdomain.ReasonCodeDuplicateBilling,
						Message:           "msg-2 の請求は既存請求と重複しています。",
					},
				},
				DuplicateCount: 1,
				Failures: []billingdomain.Failure{
					{
						ParsedEmailID:     9003,
						EmailID:           103,
						ExternalMessageID: "msg-3",
						Stage:             billingdomain.FailureStageSaveBilling,
						Code:              billingdomain.FailureCodeBillingPersistFailed,
						Message:           "msg-3 の請求保存に失敗しました。",
					},
				},
			}, nil
		},
	})

	result, err := adapter.Execute(context.Background(), manualapp.BillingCommand{
		UserID: 1,
		EligibleItems: []manualapp.EligibleItem{
			{
				ParsedEmailID:      9001,
				EmailID:            101,
				ExternalMessageID:  "msg-1",
				VendorID:           3001,
				VendorName:         "Acme",
				MatchedBy:          "name_exact",
				ProductNameDisplay: &productNameDisplay,
				BillingNumber:      "INV-001",
				InvoiceNumber:      &invoiceNumber,
				Amount:             1200,
				Currency:           "JPY",
				BillingDate:        &billingDate,
				PaymentCycle:       "one_time",
				LineItems: []manualapp.EligibleLineItem{
					{
						ProductNameRaw: &lineItemName,
						Amount:         localFloat64Ptr(1200),
						Currency:       localStringPtr("JPY"),
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if result.CreatedCount != 1 || len(result.CreatedItems) != 1 {
		t.Fatalf("unexpected created result: %+v", result)
	}
	if result.CreatedItems[0].BillingID != 7001 {
		t.Fatalf("unexpected created item: %+v", result.CreatedItems[0])
	}
	if result.DuplicateCount != 1 || len(result.DuplicateItems) != 1 {
		t.Fatalf("unexpected duplicate result: %+v", result)
	}
	if result.DuplicateItems[0].ExistingBillingID != 7000 {
		t.Fatalf("unexpected duplicate item: %+v", result.DuplicateItems[0])
	}
	if result.DuplicateItems[0].ReasonCode != billingdomain.ReasonCodeDuplicateBilling {
		t.Fatalf("unexpected duplicate reason code: %+v", result.DuplicateItems[0])
	}
	if result.DuplicateItems[0].Message != "msg-2 の請求は既存請求と重複しています。" {
		t.Fatalf("expected duplicate message to be mapped, got %+v", result.DuplicateItems[0])
	}
	if len(result.Failures) != 1 || result.Failures[0].Code != billingdomain.FailureCodeBillingPersistFailed {
		t.Fatalf("unexpected failures: %+v", result.Failures)
	}
	if result.Failures[0].Message != "msg-3 の請求保存に失敗しました。" {
		t.Fatalf("expected failure message to be mapped, got %+v", result.Failures[0])
	}
}

func localStringPtr(value string) *string {
	return &value
}

func localFloat64Ptr(value float64) *float64 {
	return &value
}
