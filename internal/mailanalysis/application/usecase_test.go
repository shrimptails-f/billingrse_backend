package application

import (
	commondomain "business/internal/common/domain"
	"business/internal/library/logger"
	"business/internal/mailanalysis/domain"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

type mockClock struct {
	now time.Time
}

func (c *mockClock) Now() time.Time {
	return c.now
}

func (c *mockClock) After(d time.Duration) <-chan time.Time {
	ch := make(chan time.Time)
	close(ch)
	return ch
}

type mockAnalyzerFactory struct {
	create func(ctx context.Context, spec AnalyzerSpec) (Analyzer, error)
}

func (m *mockAnalyzerFactory) Create(ctx context.Context, spec AnalyzerSpec) (Analyzer, error) {
	return m.create(ctx, spec)
}

type mockAnalyzer struct {
	analyze func(ctx context.Context, email EmailForAnalysisTarget) (domain.AnalysisOutput, error)
}

func (m *mockAnalyzer) Analyze(ctx context.Context, email EmailForAnalysisTarget) (domain.AnalysisOutput, error) {
	return m.analyze(ctx, email)
}

type mockParsedEmailRepository struct {
	saveAll func(ctx context.Context, input domain.SaveInput) ([]domain.ParsedEmailRecord, error)
}

func (m *mockParsedEmailRepository) SaveAll(ctx context.Context, input domain.SaveInput) ([]domain.ParsedEmailRecord, error) {
	return m.saveAll(ctx, input)
}

func TestUseCaseExecute_SavesParsedEmailsAndReturnsSummary(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Date(2026, 3, 24, 10, 0, 0, 0, time.UTC)

	factoryCalls := 0
	repositoryCalls := 0

	uc := NewUseCase(
		&mockClock{now: now},
		&mockAnalyzerFactory{
			create: func(ctx context.Context, spec AnalyzerSpec) (Analyzer, error) {
				factoryCalls++
				if spec.UserID != 9 {
					t.Fatalf("unexpected user id: %d", spec.UserID)
				}
				return &mockAnalyzer{
					analyze: func(ctx context.Context, email EmailForAnalysisTarget) (domain.AnalysisOutput, error) {
						if email.EmailID != 101 {
							t.Fatalf("unexpected email id: %d", email.EmailID)
						}
						return domain.AnalysisOutput{
							ParsedEmails: []commondomain.ParsedEmail{
								{
									VendorName:    stringPtr(" Example Vendor "),
									BillingNumber: stringPtr(" INV-001 "),
									Currency:      stringPtr(" jpy "),
									PaymentCycle:  stringPtr("one time"),
								},
								{
									Amount:       float64Ptr(12.345),
									Currency:     stringPtr("usd"),
									PaymentCycle: stringPtr(" recurring "),
								},
							},
							PromptVersion: " emailanalysis_v1 ",
						}, nil
					},
				}, nil
			},
		},
		&mockParsedEmailRepository{
			saveAll: func(ctx context.Context, input domain.SaveInput) ([]domain.ParsedEmailRecord, error) {
				repositoryCalls++
				if input.UserID != 9 || input.EmailID != 101 {
					t.Fatalf("unexpected save input ids: %+v", input)
				}
				if input.AnalysisRunID == "" {
					t.Fatal("analysis_run_id should be generated")
				}
				if !input.ExtractedAt.Equal(now) {
					t.Fatalf("unexpected extracted_at: %s", input.ExtractedAt)
				}
				if input.PromptVersion != "emailanalysis_v1" {
					t.Fatalf("unexpected metadata: %+v", input)
				}
				if got := *input.ParsedEmails[0].VendorName; got != "Example Vendor" {
					t.Fatalf("unexpected normalized vendor: %q", got)
				}
				if got := *input.ParsedEmails[0].Currency; got != "JPY" {
					t.Fatalf("unexpected normalized currency: %q", got)
				}
				if got := *input.ParsedEmails[1].BillingNumber; got != "digest_0123abcd" {
					t.Fatalf("unexpected fallback billing number: %q", got)
				}
				if got := *input.ParsedEmails[0].PaymentCycle; got != "one_time" {
					t.Fatalf("unexpected normalized payment cycle: %q", got)
				}
				if got := *input.ParsedEmails[1].PaymentCycle; got != "recurring" {
					t.Fatalf("unexpected normalized recurring cycle: %q", got)
				}

				return []domain.ParsedEmailRecord{
					{ID: 1001, EmailID: 101},
					{ID: 1002, EmailID: 101},
				}, nil
			},
		},
		logger.NewNop(),
	)

	result, err := uc.Execute(ctx, Command{
		UserID: 9,
		Emails: []EmailForAnalysisTarget{
			{
				EmailID:           101,
				ExternalMessageID: "msg-1",
				Subject:           "subject",
				From:              "from@example.com",
				ReceivedAt:        now,
				Body:              "body",
				BodyDigest:        "0123abcd",
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if factoryCalls != 1 {
		t.Fatalf("expected 1 factory call, got %d", factoryCalls)
	}
	if repositoryCalls != 1 {
		t.Fatalf("expected 1 repository call, got %d", repositoryCalls)
	}
	if result.ParsedEmailCount != 2 {
		t.Fatalf("expected parsed email count 2, got %d", result.ParsedEmailCount)
	}
	if len(result.ParsedEmailIDs) != 2 || result.ParsedEmailIDs[0] != 1001 || result.ParsedEmailIDs[1] != 1002 {
		t.Fatalf("unexpected parsed email ids: %+v", result.ParsedEmailIDs)
	}
	if len(result.Failures) != 0 {
		t.Fatalf("unexpected failures: %+v", result.Failures)
	}
	if len(result.ParsedEmails) != 2 {
		t.Fatalf("unexpected parsed email results: %+v", result.ParsedEmails)
	}
	if result.ParsedEmails[0].BodyDigest != "0123abcd" || result.ParsedEmails[1].BodyDigest != "0123abcd" {
		t.Fatalf("unexpected body digest propagation: %+v", result.ParsedEmails)
	}
}

func TestUseCaseExecute_PartialFailuresContinue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Date(2026, 3, 24, 11, 0, 0, 0, time.UTC)

	uc := NewUseCase(
		&mockClock{now: now},
		&mockAnalyzerFactory{
			create: func(ctx context.Context, spec AnalyzerSpec) (Analyzer, error) {
				return &mockAnalyzer{
					analyze: func(ctx context.Context, email EmailForAnalysisTarget) (domain.AnalysisOutput, error) {
						switch email.ExternalMessageID {
						case "msg-analyze-failed":
							return domain.AnalysisOutput{}, errors.New("openai unavailable")
						case "msg-response-invalid":
							return domain.AnalysisOutput{}, fmt.Errorf("%w: bad response", domain.ErrAnalysisResponseInvalid)
						case "msg-empty":
							return domain.AnalysisOutput{
								ParsedEmails:  nil,
								PromptVersion: "emailanalysis_v1",
							}, nil
						default:
							return domain.AnalysisOutput{
								ParsedEmails: []commondomain.ParsedEmail{
									{VendorName: stringPtr("Vendor")},
								},
								PromptVersion: "emailanalysis_v1",
							}, nil
						}
					},
				}, nil
			},
		},
		&mockParsedEmailRepository{
			saveAll: func(ctx context.Context, input domain.SaveInput) ([]domain.ParsedEmailRecord, error) {
				if input.EmailID == 5 {
					return nil, errors.New("db unavailable")
				}
				return []domain.ParsedEmailRecord{{ID: 600, EmailID: input.EmailID}}, nil
			},
		},
		logger.NewNop(),
	)

	result, err := uc.Execute(ctx, Command{
		UserID: 9,
		Emails: []EmailForAnalysisTarget{
			{EmailID: 1, ExternalMessageID: "msg-invalid", Body: "   "},
			{EmailID: 2, ExternalMessageID: "msg-analyze-failed", Body: "body"},
			{EmailID: 3, ExternalMessageID: "msg-response-invalid", Body: "body"},
			{EmailID: 4, ExternalMessageID: "msg-empty", Body: "body"},
			{EmailID: 5, ExternalMessageID: "msg-save-failed", Body: "body"},
			{EmailID: 6, ExternalMessageID: "msg-success", Body: "body"},
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if result.ParsedEmailCount != 1 {
		t.Fatalf("expected parsed email count 1, got %d", result.ParsedEmailCount)
	}
	if len(result.ParsedEmailIDs) != 1 || result.ParsedEmailIDs[0] != 600 {
		t.Fatalf("unexpected parsed email ids: %+v", result.ParsedEmailIDs)
	}
	if len(result.Failures) != 5 {
		t.Fatalf("expected 5 failures, got %+v", result.Failures)
	}

	expected := []struct {
		emailID uint
		stage   string
		code    string
		message string
	}{
		{emailID: 1, stage: domain.FailureStageNormalizeInput, code: domain.FailureCodeInvalidEmailInput, message: "メールID 1 (msg-invalid) の入力が不正です。件名、本文、外部メッセージIDを確認してください。"},
		{emailID: 2, stage: domain.FailureStageAnalyze, code: domain.FailureCodeAnalysisFailed, message: "メールID 2 (msg-analyze-failed) の解析に失敗しました。しばらく時間をおいて再実行してください。"},
		{emailID: 3, stage: domain.FailureStageResponseParse, code: domain.FailureCodeAnalysisResponseInvalid, message: "メールID 3 (msg-response-invalid) の解析結果の形式が不正でした。"},
		{emailID: 4, stage: domain.FailureStageResponseParse, code: domain.FailureCodeAnalysisResponseEmpty, message: "メールID 4 (msg-empty) の解析結果を取得できませんでした。"},
		{emailID: 5, stage: domain.FailureStageSave, code: domain.FailureCodeParsedEmailSaveFailed, message: "メールID 5 (msg-save-failed) の解析結果の保存に失敗しました。"},
	}

	for idx, failure := range result.Failures {
		if failure.EmailID != expected[idx].emailID || failure.Stage != expected[idx].stage || failure.Code != expected[idx].code || failure.Message != expected[idx].message {
			t.Fatalf("unexpected failure at %d: %+v", idx, failure)
		}
	}
}

func TestUseCaseExecute_InvalidCommand(t *testing.T) {
	t.Parallel()

	uc := NewUseCase(
		&mockClock{now: time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)},
		&mockAnalyzerFactory{
			create: func(ctx context.Context, spec AnalyzerSpec) (Analyzer, error) {
				t.Fatal("factory should not be called")
				return nil, nil
			},
		},
		&mockParsedEmailRepository{
			saveAll: func(ctx context.Context, input domain.SaveInput) ([]domain.ParsedEmailRecord, error) {
				t.Fatal("repository should not be called")
				return nil, nil
			},
		},
		logger.NewNop(),
	)

	_, err := uc.Execute(context.Background(), Command{})
	if !errors.Is(err, domain.ErrInvalidCommand) {
		t.Fatalf("expected ErrInvalidCommand, got %v", err)
	}
}

func TestUseCaseExecute_EmptyEmailsReturnsImmediately(t *testing.T) {
	t.Parallel()

	factoryCalled := false
	uc := NewUseCase(
		&mockClock{now: time.Date(2026, 3, 24, 12, 30, 0, 0, time.UTC)},
		&mockAnalyzerFactory{
			create: func(ctx context.Context, spec AnalyzerSpec) (Analyzer, error) {
				factoryCalled = true
				return nil, nil
			},
		},
		&mockParsedEmailRepository{
			saveAll: func(ctx context.Context, input domain.SaveInput) ([]domain.ParsedEmailRecord, error) {
				t.Fatal("repository should not be called")
				return nil, nil
			},
		},
		logger.NewNop(),
	)

	result, err := uc.Execute(context.Background(), Command{UserID: 7})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if factoryCalled {
		t.Fatal("factory should not be called for empty input")
	}
	if result.ParsedEmailCount != 0 || len(result.ParsedEmailIDs) != 0 || len(result.Failures) != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestUseCaseExecute_EmptyEmailsReturnsImmediatelyWithoutDependencies(t *testing.T) {
	t.Parallel()

	uc := NewUseCase(
		&mockClock{now: time.Date(2026, 3, 24, 12, 45, 0, 0, time.UTC)},
		nil,
		nil,
		logger.NewNop(),
	)

	result, err := uc.Execute(context.Background(), Command{UserID: 7})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.ParsedEmailCount != 0 || len(result.ParsedEmailIDs) != 0 || len(result.Failures) != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestUseCaseExecute_DependencyMissingReturnsError(t *testing.T) {
	t.Parallel()

	uc := NewUseCase(
		&mockClock{now: time.Date(2026, 3, 24, 13, 0, 0, 0, time.UTC)},
		nil,
		nil,
		logger.NewNop(),
	)

	_, err := uc.Execute(context.Background(), Command{
		UserID: 1,
		Emails: []EmailForAnalysisTarget{
			{EmailID: 1, ExternalMessageID: "msg-1", Body: "body"},
		},
	})
	if err == nil {
		t.Fatal("expected error when dependencies are missing")
	}
}

func stringPtr(value string) *string {
	return &value
}

func float64Ptr(value float64) *float64 {
	return &value
}
