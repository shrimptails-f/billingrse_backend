package infrastructure

import (
	beapp "business/internal/billingeligibility/application"
	bedomain "business/internal/billingeligibility/domain"
	commondomain "business/internal/common/domain"
	manualapp "business/internal/manualmailworkflow/application"
	"context"
	"testing"
	"time"
)

type stubBillingEligibilityUseCase struct {
	execute func(ctx context.Context, cmd beapp.Command) (beapp.Result, error)
}

func (s *stubBillingEligibilityUseCase) Execute(ctx context.Context, cmd beapp.Command) (beapp.Result, error) {
	return s.execute(ctx, cmd)
}

func TestDirectBillingEligibilityAdapter_Execute_ConvertsCommandAndResult(t *testing.T) {
	t.Parallel()

	billingNumber := "INV-001"
	invoiceNumber := "T1234567890123"
	billingDate := time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC)
	adapter := NewDirectBillingEligibilityAdapter(&stubBillingEligibilityUseCase{
		execute: func(ctx context.Context, cmd beapp.Command) (beapp.Result, error) {
			if len(cmd.ResolvedItems) != 1 {
				t.Fatalf("unexpected command: %+v", cmd)
			}
			if cmd.ResolvedItems[0].Data.BillingNumber == nil || *cmd.ResolvedItems[0].Data.BillingNumber != billingNumber {
				t.Fatalf("expected billing number in target data, got %+v", cmd.ResolvedItems[0])
			}
			return beapp.Result{
				EligibleItems: []bedomain.EligibleItem{
					{
						ParsedEmailID:     9001,
						EmailID:           101,
						ExternalMessageID: "msg-1",
						VendorID:          3001,
						VendorName:        "Acme",
						MatchedBy:         "name_exact",
						BillingNumber:     billingNumber,
						InvoiceNumber:     &invoiceNumber,
						Amount:            1200,
						Currency:          "JPY",
						BillingDate:       &billingDate,
						PaymentCycle:      "one_time",
					},
				},
				EligibleCount: 1,
				IneligibleItems: []bedomain.IneligibleItem{
					{
						ParsedEmailID:     9002,
						EmailID:           102,
						ExternalMessageID: "msg-2",
						VendorID:          3001,
						VendorName:        "Acme",
						MatchedBy:         "name_exact",
						ReasonCode:        bedomain.ReasonCodeCurrencyEmpty,
					},
				},
				IneligibleCount: 1,
				Failures: []bedomain.Failure{
					{
						ParsedEmailID:     9003,
						EmailID:           103,
						ExternalMessageID: "msg-3",
						Stage:             bedomain.FailureStageNormalizeInput,
						Code:              bedomain.FailureCodeInvalidEligibilityTarget,
					},
				},
			}, nil
		},
	})

	result, err := adapter.Execute(context.Background(), manualapp.BillingEligibilityCommand{
		UserID: 1,
		ResolvedItems: []manualapp.ResolvedItem{
			{
				ParsedEmailID:     9001,
				EmailID:           101,
				ExternalMessageID: "msg-1",
				VendorID:          3001,
				VendorName:        "Acme",
				MatchedBy:         "name_exact",
				Data: commondomain.ParsedEmail{
					BillingNumber: &billingNumber,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if result.EligibleCount != 1 || len(result.EligibleItems) != 1 {
		t.Fatalf("unexpected eligible result: %+v", result)
	}
	if result.EligibleItems[0].InvoiceNumber == nil || *result.EligibleItems[0].InvoiceNumber != invoiceNumber {
		t.Fatalf("expected invoice number to be mapped, got %+v", result.EligibleItems[0])
	}
	if result.IneligibleCount != 1 || result.IneligibleItems[0].ReasonCode != bedomain.ReasonCodeCurrencyEmpty {
		t.Fatalf("unexpected ineligible result: %+v", result)
	}
	if len(result.Failures) != 1 || result.Failures[0].Code != bedomain.FailureCodeInvalidEligibilityTarget {
		t.Fatalf("unexpected failures: %+v", result)
	}
}
