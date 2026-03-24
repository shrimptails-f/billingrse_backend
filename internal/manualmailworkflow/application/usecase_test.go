package application

import (
	"business/internal/library/logger"
	"context"
	"errors"
	"testing"
	"time"
)

type stubFetchStage struct {
	execute func(ctx context.Context, cmd FetchCommand) (FetchResult, error)
}

func (s *stubFetchStage) Execute(ctx context.Context, cmd FetchCommand) (FetchResult, error) {
	return s.execute(ctx, cmd)
}

type stubAnalyzeStage struct {
	execute func(ctx context.Context, cmd AnalyzeCommand) (AnalyzeResult, error)
}

func (s *stubAnalyzeStage) Execute(ctx context.Context, cmd AnalyzeCommand) (AnalyzeResult, error) {
	return s.execute(ctx, cmd)
}

func TestUseCaseExecute_FetchThenAnalyze(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)
	fetchCalls := 0
	analyzeCalls := 0

	uc := NewUseCase(
		&stubFetchStage{
			execute: func(ctx context.Context, cmd FetchCommand) (FetchResult, error) {
				fetchCalls++
				if cmd.UserID != 3 || cmd.ConnectionID != 8 {
					t.Fatalf("unexpected fetch command: %+v", cmd)
				}
				if cmd.Condition.LabelName != "billing" {
					t.Fatalf("unexpected label: %q", cmd.Condition.LabelName)
				}
				return FetchResult{
					Provider:            "gmail",
					AccountIdentifier:   "user@example.com",
					MatchedMessageCount: 2,
					CreatedEmailIDs:     []uint{101},
					CreatedEmails: []CreatedEmail{
						{
							EmailID:           101,
							ExternalMessageID: "msg-1",
							Subject:           "Invoice",
							From:              "billing@example.com",
							To:                []string{"user@example.com"},
							ReceivedAt:        now,
							Body:              "invoice body",
						},
					},
					ExistingEmailIDs: []uint{202},
					Failures: []FetchFailure{
						{ExternalMessageID: "msg-2", Stage: "save", Code: "email_save_failed"},
					},
				}, nil
			},
		},
		&stubAnalyzeStage{
			execute: func(ctx context.Context, cmd AnalyzeCommand) (AnalyzeResult, error) {
				analyzeCalls++
				if cmd.UserID != 3 {
					t.Fatalf("unexpected analyze user id: %d", cmd.UserID)
				}
				if len(cmd.Emails) != 1 || cmd.Emails[0].EmailID != 101 || cmd.Emails[0].ReceivedAt != now {
					t.Fatalf("unexpected analyze emails: %+v", cmd.Emails)
				}
				return AnalyzeResult{
					ParsedEmailIDs:     []uint{9001, 9002},
					AnalyzedEmailCount: 1,
					ParsedEmailCount:   2,
					Failures: []AnalysisFailure{
						{EmailID: 101, ExternalMessageID: "msg-1", Stage: "save", Code: "parsed_email_save_failed"},
					},
				}, nil
			},
		},
		logger.NewNop(),
	)

	result, err := uc.Execute(context.Background(), Command{
		UserID:       3,
		ConnectionID: 8,
		Condition: FetchCondition{
			LabelName: " billing ",
			Since:     now.Add(-time.Hour),
			Until:     now.Add(time.Hour),
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if fetchCalls != 1 {
		t.Fatalf("expected 1 fetch call, got %d", fetchCalls)
	}
	if analyzeCalls != 1 {
		t.Fatalf("expected 1 analyze call, got %d", analyzeCalls)
	}
	if result.Fetch.Provider != "gmail" {
		t.Fatalf("unexpected fetch result: %+v", result.Fetch)
	}
	if len(result.Analysis.ParsedEmailIDs) != 2 {
		t.Fatalf("unexpected analysis result: %+v", result.Analysis)
	}
}

func TestUseCaseExecute_SkipsAnalyzeWhenNoCreatedEmails(t *testing.T) {
	t.Parallel()

	analyzeCalled := false
	uc := NewUseCase(
		&stubFetchStage{
			execute: func(ctx context.Context, cmd FetchCommand) (FetchResult, error) {
				return FetchResult{
					Provider:            "gmail",
					AccountIdentifier:   "user@example.com",
					MatchedMessageCount: 1,
					ExistingEmailIDs:    []uint{10},
				}, nil
			},
		},
		&stubAnalyzeStage{
			execute: func(ctx context.Context, cmd AnalyzeCommand) (AnalyzeResult, error) {
				analyzeCalled = true
				return AnalyzeResult{}, nil
			},
		},
		logger.NewNop(),
	)

	result, err := uc.Execute(context.Background(), Command{
		UserID:       1,
		ConnectionID: 2,
		Condition: FetchCondition{
			LabelName: "billing",
			Since:     time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
			Until:     time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if analyzeCalled {
		t.Fatal("analyze stage should not be called when there are no created emails")
	}
	if len(result.Analysis.ParsedEmailIDs) != 0 || result.Analysis.ParsedEmailCount != 0 {
		t.Fatalf("unexpected analysis result: %+v", result.Analysis)
	}
}

func TestUseCaseExecute_InvalidCommand(t *testing.T) {
	t.Parallel()

	uc := NewUseCase(
		&stubFetchStage{
			execute: func(ctx context.Context, cmd FetchCommand) (FetchResult, error) {
				t.Fatal("fetch stage should not be called")
				return FetchResult{}, nil
			},
		},
		&stubAnalyzeStage{
			execute: func(ctx context.Context, cmd AnalyzeCommand) (AnalyzeResult, error) {
				t.Fatal("analyze stage should not be called")
				return AnalyzeResult{}, nil
			},
		},
		logger.NewNop(),
	)

	_, err := uc.Execute(context.Background(), Command{
		UserID:       1,
		ConnectionID: 2,
		Condition: FetchCondition{
			LabelName: "billing",
			Since:     time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
			Until:     time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
		},
	})
	if !errors.Is(err, ErrFetchConditionInvalid) {
		t.Fatalf("expected ErrFetchConditionInvalid, got %v", err)
	}
}
