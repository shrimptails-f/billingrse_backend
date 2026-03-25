package infrastructure

import (
	commondomain "business/internal/common/domain"
	manualapp "business/internal/manualmailworkflow/application"
	vrapp "business/internal/vendorresolution/application"
	vrdomain "business/internal/vendorresolution/domain"
	"context"
	"testing"
)

type stubVendorResolutionUseCase struct {
	execute func(ctx context.Context, cmd vrapp.Command) (vrapp.Result, error)
}

func (s *stubVendorResolutionUseCase) Execute(ctx context.Context, cmd vrapp.Command) (vrapp.Result, error) {
	return s.execute(ctx, cmd)
}

func TestDirectVendorResolutionAdapter_Execute_SupplementsParsedEmailData(t *testing.T) {
	t.Parallel()

	billingNumber := "INV-001"
	adapter := NewDirectVendorResolutionAdapter(&stubVendorResolutionUseCase{
		execute: func(ctx context.Context, cmd vrapp.Command) (vrapp.Result, error) {
			if len(cmd.ParsedEmails) != 1 {
				t.Fatalf("unexpected command: %+v", cmd)
			}
			return vrapp.Result{
				ResolvedItems: []vrdomain.ResolvedItem{
					{
						ParsedEmailID:     9001,
						EmailID:           101,
						ExternalMessageID: "msg-1",
						VendorID:          3001,
						VendorName:        "Acme",
						MatchedBy:         vrdomain.MatchedByNameExact,
					},
				},
				ResolvedCount: 1,
				UnresolvedItems: []vrdomain.UnresolvedItem{
					{
						ParsedEmailID:       9002,
						EmailID:             102,
						ExternalMessageID:   "msg-2",
						ReasonCode:          vrdomain.ReasonCodeVendorUnresolved,
						Message:             "msg-2 の候補「Unknown」を支払先として特定できませんでした。",
						CandidateVendorName: "Unknown",
					},
				},
				UnresolvedCount: 1,
				Failures: []vrdomain.Failure{
					{
						ParsedEmailID:     9003,
						EmailID:           103,
						ExternalMessageID: "msg-3",
						Stage:             vrdomain.FailureStageResolveVendor,
						Code:              vrdomain.FailureCodeVendorResolveFail,
						Message:           "msg-3 の候補「Broken」を使った支払先解決に失敗しました。",
					},
				},
			}, nil
		},
	})

	result, err := adapter.Execute(context.Background(), manualapp.VendorResolutionCommand{
		UserID: 1,
		ParsedEmails: []manualapp.ParsedEmail{
			{
				ParsedEmailID:     9001,
				EmailID:           101,
				ExternalMessageID: "msg-1",
				BodyDigest:        "digest-msg-1",
				Data: commondomain.ParsedEmail{
					BillingNumber: &billingNumber,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if len(result.ResolvedItems) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.ResolvedItems[0].Data.BillingNumber == nil || *result.ResolvedItems[0].Data.BillingNumber != billingNumber {
		t.Fatalf("expected parsed email data to be supplemented, got %+v", result.ResolvedItems[0])
	}
	if len(result.UnresolvedItems) != 1 || result.UnresolvedItems[0].Message != "msg-2 の候補「Unknown」を支払先として特定できませんでした。" {
		t.Fatalf("expected unresolved message to be mapped, got %+v", result.UnresolvedItems)
	}
	if len(result.Failures) != 1 || result.Failures[0].Message != "msg-3 の候補「Broken」を使った支払先解決に失敗しました。" {
		t.Fatalf("expected failure message to be mapped, got %+v", result.Failures)
	}
}
