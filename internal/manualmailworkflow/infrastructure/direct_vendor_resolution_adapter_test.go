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
						BodyDigest:        "digest-msg-1",
						VendorID:          3001,
						VendorName:        "Acme",
						MatchedBy:         vrdomain.MatchedByNameExact,
					},
				},
				ResolvedCount: 1,
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
}
