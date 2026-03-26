package infrastructure

import (
	commondomain "business/internal/common/domain"
	maapp "business/internal/mailanalysis/application"
	madomain "business/internal/mailanalysis/domain"
	manualapp "business/internal/manualmailworkflow/application"
	"context"
	"testing"
	"time"
)

type stubMailAnalysisUseCase struct {
	execute func(ctx context.Context, cmd maapp.Command) (maapp.Result, error)
}

func (s *stubMailAnalysisUseCase) Execute(ctx context.Context, cmd maapp.Command) (maapp.Result, error) {
	return s.execute(ctx, cmd)
}

func TestDirectMailAnalysisAdapter_Execute_MapsFailureMessages(t *testing.T) {
	t.Parallel()

	vendorName := "Acme"
	receivedAt := time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)
	adapter := NewDirectMailAnalysisAdapter(&stubMailAnalysisUseCase{
		execute: func(ctx context.Context, cmd maapp.Command) (maapp.Result, error) {
			return maapp.Result{
				ParsedEmails: []maapp.ParsedEmailResultItem{
					{
						ParsedEmailID:     9001,
						EmailID:           101,
						ExternalMessageID: "msg-1",
						Subject:           "Invoice",
						From:              "billing@example.com",
						To:                []string{"user@example.com"},
						BodyDigest:        "digest-msg-1",
						ParsedEmail: commondomain.ParsedEmail{
							VendorName: &vendorName,
						},
					},
				},
				ParsedEmailCount: 1,
				Failures: []madomain.MessageFailure{
					{
						EmailID:           101,
						ExternalMessageID: "msg-1",
						Stage:             madomain.FailureStageAnalyze,
						Code:              madomain.FailureCodeAnalysisFailed,
						Message:           "msg-1 のメール解析に失敗しました。",
					},
				},
			}, nil
		},
	})

	result, err := adapter.Execute(context.Background(), manualapp.AnalyzeCommand{
		UserID: 1,
		Emails: []manualapp.CreatedEmail{
			{
				EmailID:           101,
				ExternalMessageID: "msg-1",
				Subject:           "Invoice",
				From:              "billing@example.com",
				To:                []string{"user@example.com"},
				ReceivedAt:        receivedAt,
				Body:              "body",
				BodyDigest:        "digest-msg-1",
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if len(result.ParsedEmails) != 1 || result.ParsedEmails[0].BodyDigest != "digest-msg-1" {
		t.Fatalf("unexpected parsed emails: %+v", result.ParsedEmails)
	}
	if len(result.Failures) != 1 || result.Failures[0].Message != "msg-1 のメール解析に失敗しました。" {
		t.Fatalf("expected failure message to be mapped, got %+v", result.Failures)
	}
}
