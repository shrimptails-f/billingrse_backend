package infrastructure

import (
	mfapp "business/internal/mailfetch/application"
	mfdomain "business/internal/mailfetch/domain"
	manualapp "business/internal/manualmailworkflow/application"
	"context"
	"testing"
	"time"
)

type stubMailFetchUseCase struct {
	execute func(ctx context.Context, cmd mfapp.Command) (mfapp.Result, error)
}

func (s *stubMailFetchUseCase) Execute(ctx context.Context, cmd mfapp.Command) (mfapp.Result, error) {
	return s.execute(ctx, cmd)
}

func TestDirectManualMailFetchAdapter_Execute_MapsFailureMessages(t *testing.T) {
	t.Parallel()

	receivedAt := time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)
	adapter := NewDirectManualMailFetchAdapter(&stubMailFetchUseCase{
		execute: func(ctx context.Context, cmd mfapp.Command) (mfapp.Result, error) {
			return mfapp.Result{
				CreatedEmails: []mfapp.CreatedEmail{
					{
						EmailID:           101,
						ExternalMessageID: "msg-1",
						Subject:           "Invoice",
						From:              "billing@example.com",
						To:                []string{"user@example.com"},
						Date:              receivedAt,
						Body:              "body",
						BodyDigest:        "digest-msg-1",
					},
				},
				ExistingEmailIDs: []uint{202},
				Failures: []mfdomain.MessageFailure{
					{
						ExternalMessageID: "msg-2",
						Stage:             mfdomain.FailureStageSave,
						Code:              mfdomain.FailureCodeEmailSaveFailed,
						Message:           "msg-2 の取得メール保存に失敗しました。",
					},
				},
			}, nil
		},
	})

	result, err := adapter.Execute(context.Background(), manualapp.FetchCommand{
		UserID:       1,
		ConnectionID: 2,
		Condition: manualapp.FetchCondition{
			LabelName: "billing",
			Since:     receivedAt.Add(-time.Hour),
			Until:     receivedAt.Add(time.Hour),
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if len(result.CreatedEmails) != 1 || result.CreatedEmails[0].BodyDigest != "digest-msg-1" {
		t.Fatalf("unexpected created emails: %+v", result.CreatedEmails)
	}
	if len(result.Failures) != 1 || result.Failures[0].Message != "msg-2 の取得メール保存に失敗しました。" {
		t.Fatalf("expected failure message to be mapped, got %+v", result.Failures)
	}
}
