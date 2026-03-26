package application

import (
	commondomain "business/internal/common/domain"
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

type stubVendorResolutionStage struct {
	execute func(ctx context.Context, cmd VendorResolutionCommand) (VendorResolutionResult, error)
}

func (s *stubVendorResolutionStage) Execute(ctx context.Context, cmd VendorResolutionCommand) (VendorResolutionResult, error) {
	return s.execute(ctx, cmd)
}

type stubBillingEligibilityStage struct {
	execute func(ctx context.Context, cmd BillingEligibilityCommand) (BillingEligibilityResult, error)
}

func (s *stubBillingEligibilityStage) Execute(ctx context.Context, cmd BillingEligibilityCommand) (BillingEligibilityResult, error) {
	return s.execute(ctx, cmd)
}

type stubBillingStage struct {
	execute func(ctx context.Context, cmd BillingCommand) (BillingResult, error)
}

func (s *stubBillingStage) Execute(ctx context.Context, cmd BillingCommand) (BillingResult, error) {
	return s.execute(ctx, cmd)
}

func TestUseCaseExecute_FetchThenAnalyze(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)
	fetchCalls := 0
	analyzeCalls := 0
	vendorResolutionCalls := 0
	billingEligibilityCalls := 0
	billingCalls := 0
	amount := 1200.0
	currency := "JPY"
	billingNumber := "INV-001"
	paymentCycle := "one_time"

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
					CreatedEmails: []CreatedEmail{
						{
							EmailID:           101,
							ExternalMessageID: "msg-1",
							Subject:           "Invoice",
							From:              "billing@example.com",
							To:                []string{"user@example.com"},
							ReceivedAt:        now,
							Body:              "invoice body",
							BodyDigest:        "digest-msg-1",
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
				if cmd.Emails[0].BodyDigest != "digest-msg-1" {
					t.Fatalf("unexpected analyze body digest: %+v", cmd.Emails)
				}
				return AnalyzeResult{
					ParsedEmails: []ParsedEmail{
						{
							ParsedEmailID:     9001,
							EmailID:           101,
							ExternalMessageID: "msg-1",
							Subject:           "Invoice",
							From:              "billing@example.com",
							To:                []string{"user@example.com"},
							BodyDigest:        "digest-msg-1",
							Data: commondomain.ParsedEmail{
								VendorName:    stringPtr("Acme"),
								BillingNumber: stringPtr(billingNumber),
								Amount:        float64Ptr(amount),
								Currency:      stringPtr(currency),
								PaymentCycle:  stringPtr(paymentCycle),
							},
						},
						{
							ParsedEmailID:     9002,
							EmailID:           101,
							ExternalMessageID: "msg-1",
							Subject:           "Invoice",
							From:              "billing@example.com",
							To:                []string{"user@example.com"},
							BodyDigest:        "digest-msg-1",
							Data:              commondomain.ParsedEmail{VendorName: stringPtr("Unknown")},
						},
					},
					ParsedEmailCount: 2,
					Failures: []AnalysisFailure{
						{EmailID: 101, ExternalMessageID: "msg-1", Stage: "save", Code: "parsed_email_save_failed"},
					},
				}, nil
			},
		},
		&stubVendorResolutionStage{
			execute: func(ctx context.Context, cmd VendorResolutionCommand) (VendorResolutionResult, error) {
				vendorResolutionCalls++
				if cmd.UserID != 3 {
					t.Fatalf("unexpected vendor resolution user id: %d", cmd.UserID)
				}
				if len(cmd.ParsedEmails) != 2 || cmd.ParsedEmails[0].ParsedEmailID != 9001 || cmd.ParsedEmails[1].ParsedEmailID != 9002 {
					t.Fatalf("unexpected vendor resolution inputs: %+v", cmd.ParsedEmails)
				}
				if cmd.ParsedEmails[0].BodyDigest != "digest-msg-1" || cmd.ParsedEmails[1].BodyDigest != "digest-msg-1" {
					t.Fatalf("unexpected vendor resolution body digest: %+v", cmd.ParsedEmails)
				}
				return VendorResolutionResult{
					ResolvedItems: []ResolvedItem{
						{
							ParsedEmailID:     9001,
							EmailID:           101,
							ExternalMessageID: "msg-1",
							VendorID:          3001,
							VendorName:        "Acme",
							MatchedBy:         "name_exact",
							Data: commondomain.ParsedEmail{
								VendorName:    stringPtr("Acme"),
								BillingNumber: stringPtr(billingNumber),
								Amount:        float64Ptr(amount),
								Currency:      stringPtr(currency),
								PaymentCycle:  stringPtr(paymentCycle),
							},
						},
					},
					ResolvedCount: 1,
					UnresolvedItems: []UnresolvedItem{
						{ParsedEmailID: 9002, EmailID: 101, ExternalMessageID: "msg-1", ReasonCode: reasonCodeVendorUnresolved},
					},
					UnresolvedCount: 1,
				}, nil
			},
		},
		&stubBillingEligibilityStage{
			execute: func(ctx context.Context, cmd BillingEligibilityCommand) (BillingEligibilityResult, error) {
				billingEligibilityCalls++
				if cmd.UserID != 3 {
					t.Fatalf("unexpected billing eligibility user id: %d", cmd.UserID)
				}
				if len(cmd.ResolvedItems) != 1 || cmd.ResolvedItems[0].ParsedEmailID != 9001 {
					t.Fatalf("unexpected billing eligibility inputs: %+v", cmd.ResolvedItems)
				}
				if cmd.ResolvedItems[0].Data.BillingNumber == nil || *cmd.ResolvedItems[0].Data.BillingNumber != billingNumber {
					t.Fatalf("expected parsed email data to reach billing eligibility: %+v", cmd.ResolvedItems[0])
				}
				return BillingEligibilityResult{
					EligibleItems: []EligibleItem{
						{
							ParsedEmailID:     9001,
							EmailID:           101,
							ExternalMessageID: "msg-1",
							VendorID:          3001,
							VendorName:        "Acme",
							MatchedBy:         "name_exact",
							BillingNumber:     billingNumber,
							Amount:            amount,
							Currency:          currency,
							PaymentCycle:      paymentCycle,
						},
					},
					EligibleCount: 1,
				}, nil
			},
		},
		&stubBillingStage{
			execute: func(ctx context.Context, cmd BillingCommand) (BillingResult, error) {
				billingCalls++
				if cmd.UserID != 3 {
					t.Fatalf("unexpected billing user id: %d", cmd.UserID)
				}
				if len(cmd.EligibleItems) != 1 || cmd.EligibleItems[0].ParsedEmailID != 9001 {
					t.Fatalf("unexpected billing inputs: %+v", cmd.EligibleItems)
				}
				return BillingResult{
					CreatedItems: []BillingCreatedItem{
						{
							BillingID:         7001,
							ParsedEmailID:     9001,
							EmailID:           101,
							ExternalMessageID: "msg-1",
							VendorID:          3001,
							VendorName:        "Acme",
							BillingNumber:     billingNumber,
						},
					},
					CreatedCount: 1,
				}, nil
			},
		},
		&stubWorkflowStatusRepository{},
		&fixedClock{now: time.Date(2026, 3, 25, 12, 30, 0, 0, time.UTC)},
		logger.NewNop(),
	)

	result, err := uc.Execute(context.Background(), DispatchJob{
		HistoryID:    1,
		WorkflowID:   "wf-1",
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
	if vendorResolutionCalls != 1 {
		t.Fatalf("expected 1 vendor resolution call, got %d", vendorResolutionCalls)
	}
	if billingEligibilityCalls != 1 {
		t.Fatalf("expected 1 billing eligibility call, got %d", billingEligibilityCalls)
	}
	if billingCalls != 1 {
		t.Fatalf("expected 1 billing call, got %d", billingCalls)
	}
	if result.VendorResolution.ResolvedCount != 1 || result.VendorResolution.UnresolvedCount != 1 {
		t.Fatalf("unexpected vendor resolution result: %+v", result.VendorResolution)
	}
	if result.BillingEligibility.EligibleCount != 1 || len(result.BillingEligibility.EligibleItems) != 1 {
		t.Fatalf("unexpected billing eligibility result: %+v", result.BillingEligibility)
	}
	if result.Billing.CreatedCount != 1 || len(result.Billing.CreatedItems) != 1 {
		t.Fatalf("unexpected billing result: %+v", result.Billing)
	}
}

func TestUseCaseExecute_SkipsAnalyzeWhenNoCreatedEmails(t *testing.T) {
	t.Parallel()

	analyzeCalled := false
	vendorResolutionCalled := false
	billingEligibilityCalled := false
	billingCalled := false
	var fetchProgress StageProgress
	completedStatus := ""
	uc := NewUseCase(
		&stubFetchStage{
			execute: func(ctx context.Context, cmd FetchCommand) (FetchResult, error) {
				return FetchResult{
					ExistingEmailIDs: []uint{10},
				}, nil
			},
		},
		&stubAnalyzeStage{
			execute: func(ctx context.Context, cmd AnalyzeCommand) (AnalyzeResult, error) {
				analyzeCalled = true
				return AnalyzeResult{}, nil
			},
		},
		&stubVendorResolutionStage{
			execute: func(ctx context.Context, cmd VendorResolutionCommand) (VendorResolutionResult, error) {
				vendorResolutionCalled = true
				return VendorResolutionResult{}, nil
			},
		},
		&stubBillingEligibilityStage{
			execute: func(ctx context.Context, cmd BillingEligibilityCommand) (BillingEligibilityResult, error) {
				billingEligibilityCalled = true
				return BillingEligibilityResult{}, nil
			},
		},
		&stubBillingStage{
			execute: func(ctx context.Context, cmd BillingCommand) (BillingResult, error) {
				billingCalled = true
				return BillingResult{}, nil
			},
		},
		&stubWorkflowStatusRepository{
			saveStage: func(ctx context.Context, progress StageProgress) error {
				if progress.Stage == workflowStageFetch {
					fetchProgress = progress
				}
				return nil
			},
			complete: func(ctx context.Context, historyID uint64, status string, finishedAt time.Time) error {
				completedStatus = status
				return nil
			},
		},
		&fixedClock{now: time.Date(2026, 3, 25, 12, 30, 0, 0, time.UTC)},
		logger.NewNop(),
	)

	_, err := uc.Execute(context.Background(), DispatchJob{
		HistoryID:    1,
		WorkflowID:   "wf-1",
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
	if vendorResolutionCalled {
		t.Fatal("vendor resolution stage should not be called when there are no created emails")
	}
	if billingEligibilityCalled {
		t.Fatal("billing eligibility stage should not be called when there are no created emails")
	}
	if billingCalled {
		t.Fatal("billing stage should not be called when there are no created emails")
	}
	if fetchProgress.Stage != workflowStageFetch {
		t.Fatalf("fetch progress was not saved: %+v", fetchProgress)
	}
	if fetchProgress.BusinessFailureCount != 0 {
		t.Fatalf("expected zero fetch business failures, got %+v", fetchProgress)
	}
	if len(fetchProgress.FailureRecords) != 1 {
		t.Fatalf("expected one fetch failure record, got %+v", fetchProgress)
	}
	if fetchProgress.FailureRecords[0].ReasonCode != reasonCodeExistingEmailsSkipped {
		t.Fatalf("unexpected fetch reason code: %+v", fetchProgress.FailureRecords[0])
	}
	if fetchProgress.FailureRecords[0].Message != "取得したメールは全て取得済みのため、後続の処理をスキップしました。" {
		t.Fatalf("unexpected fetch message: %+v", fetchProgress.FailureRecords[0])
	}
	if completedStatus != WorkflowStatusSucceeded {
		t.Fatalf("unexpected completed status: %s", completedStatus)
	}
}

func TestUseCaseExecute_SkipsVendorResolutionWhenNoParsedEmails(t *testing.T) {
	t.Parallel()

	vendorResolutionCalled := false
	billingEligibilityCalled := false
	billingCalled := false
	uc := NewUseCase(
		&stubFetchStage{
			execute: func(ctx context.Context, cmd FetchCommand) (FetchResult, error) {
				return FetchResult{
					CreatedEmails: []CreatedEmail{
						{
							EmailID:           101,
							ExternalMessageID: "msg-1",
							Subject:           "Invoice",
							From:              "billing@example.com",
							To:                []string{"user@example.com"},
							ReceivedAt:        time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC),
							Body:              "invoice body",
						},
					},
				}, nil
			},
		},
		&stubAnalyzeStage{
			execute: func(ctx context.Context, cmd AnalyzeCommand) (AnalyzeResult, error) {
				return AnalyzeResult{
					ParsedEmails: nil,
				}, nil
			},
		},
		&stubVendorResolutionStage{
			execute: func(ctx context.Context, cmd VendorResolutionCommand) (VendorResolutionResult, error) {
				vendorResolutionCalled = true
				return VendorResolutionResult{}, nil
			},
		},
		&stubBillingEligibilityStage{
			execute: func(ctx context.Context, cmd BillingEligibilityCommand) (BillingEligibilityResult, error) {
				billingEligibilityCalled = true
				return BillingEligibilityResult{}, nil
			},
		},
		&stubBillingStage{
			execute: func(ctx context.Context, cmd BillingCommand) (BillingResult, error) {
				billingCalled = true
				return BillingResult{}, nil
			},
		},
		&stubWorkflowStatusRepository{},
		&fixedClock{now: time.Date(2026, 3, 25, 12, 30, 0, 0, time.UTC)},
		logger.NewNop(),
	)

	result, err := uc.Execute(context.Background(), DispatchJob{
		HistoryID:    1,
		WorkflowID:   "wf-1",
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

	if vendorResolutionCalled {
		t.Fatal("vendor resolution stage should not be called when there are no parsed emails")
	}
	if billingEligibilityCalled {
		t.Fatal("billing eligibility stage should not be called when there are no parsed emails")
	}
	if billingCalled {
		t.Fatal("billing stage should not be called when there are no parsed emails")
	}
	if result.VendorResolution.ResolvedCount != 0 || result.VendorResolution.UnresolvedCount != 0 {
		t.Fatalf("unexpected vendor resolution result: %+v", result.VendorResolution)
	}
}

func TestUseCaseExecute_SkipsBillingEligibilityWhenNoResolvedItems(t *testing.T) {
	t.Parallel()

	billingEligibilityCalled := false
	billingCalled := false
	uc := NewUseCase(
		&stubFetchStage{
			execute: func(ctx context.Context, cmd FetchCommand) (FetchResult, error) {
				return FetchResult{
					CreatedEmails: []CreatedEmail{
						{
							EmailID:           101,
							ExternalMessageID: "msg-1",
							Subject:           "Invoice",
							From:              "billing@example.com",
							To:                []string{"user@example.com"},
							ReceivedAt:        time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC),
						},
					},
				}, nil
			},
		},
		&stubAnalyzeStage{
			execute: func(ctx context.Context, cmd AnalyzeCommand) (AnalyzeResult, error) {
				return AnalyzeResult{
					ParsedEmails: []ParsedEmail{
						{
							ParsedEmailID:     9001,
							EmailID:           101,
							ExternalMessageID: "msg-1",
						},
					},
				}, nil
			},
		},
		&stubVendorResolutionStage{
			execute: func(ctx context.Context, cmd VendorResolutionCommand) (VendorResolutionResult, error) {
				return VendorResolutionResult{
					ResolvedItems:   nil,
					UnresolvedCount: 1,
				}, nil
			},
		},
		&stubBillingEligibilityStage{
			execute: func(ctx context.Context, cmd BillingEligibilityCommand) (BillingEligibilityResult, error) {
				billingEligibilityCalled = true
				return BillingEligibilityResult{}, nil
			},
		},
		&stubBillingStage{
			execute: func(ctx context.Context, cmd BillingCommand) (BillingResult, error) {
				billingCalled = true
				return BillingResult{}, nil
			},
		},
		&stubWorkflowStatusRepository{},
		&fixedClock{now: time.Date(2026, 3, 25, 12, 30, 0, 0, time.UTC)},
		logger.NewNop(),
	)

	result, err := uc.Execute(context.Background(), DispatchJob{
		HistoryID:    1,
		WorkflowID:   "wf-1",
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

	if billingEligibilityCalled {
		t.Fatal("billing eligibility stage should not be called when there are no resolved items")
	}
	if billingCalled {
		t.Fatal("billing stage should not be called when there are no resolved items")
	}
	if result.BillingEligibility.EligibleCount != 0 || result.BillingEligibility.IneligibleCount != 0 {
		t.Fatalf("unexpected billing eligibility result: %+v", result.BillingEligibility)
	}
}

func TestUseCaseExecute_SkipsBillingWhenNoEligibleItems(t *testing.T) {
	t.Parallel()

	billingCalled := false
	uc := NewUseCase(
		&stubFetchStage{
			execute: func(ctx context.Context, cmd FetchCommand) (FetchResult, error) {
				return FetchResult{
					CreatedEmails: []CreatedEmail{
						{
							EmailID:           101,
							ExternalMessageID: "msg-1",
							Subject:           "Invoice",
							From:              "billing@example.com",
							To:                []string{"user@example.com"},
							ReceivedAt:        time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC),
						},
					},
				}, nil
			},
		},
		&stubAnalyzeStage{
			execute: func(ctx context.Context, cmd AnalyzeCommand) (AnalyzeResult, error) {
				return AnalyzeResult{
					ParsedEmails: []ParsedEmail{
						{
							ParsedEmailID:     9001,
							EmailID:           101,
							ExternalMessageID: "msg-1",
						},
					},
				}, nil
			},
		},
		&stubVendorResolutionStage{
			execute: func(ctx context.Context, cmd VendorResolutionCommand) (VendorResolutionResult, error) {
				return VendorResolutionResult{
					ResolvedItems: []ResolvedItem{
						{
							ParsedEmailID:     9001,
							EmailID:           101,
							ExternalMessageID: "msg-1",
							VendorID:          301,
							VendorName:        "Acme",
						},
					},
					ResolvedCount: 1,
				}, nil
			},
		},
		&stubBillingEligibilityStage{
			execute: func(ctx context.Context, cmd BillingEligibilityCommand) (BillingEligibilityResult, error) {
				return BillingEligibilityResult{
					EligibleItems:   nil,
					EligibleCount:   0,
					IneligibleItems: []IneligibleItem{{ParsedEmailID: 9001, ReasonCode: "currency_empty"}},
					IneligibleCount: 1,
				}, nil
			},
		},
		&stubBillingStage{
			execute: func(ctx context.Context, cmd BillingCommand) (BillingResult, error) {
				billingCalled = true
				return BillingResult{}, nil
			},
		},
		&stubWorkflowStatusRepository{},
		&fixedClock{now: time.Date(2026, 3, 25, 12, 30, 0, 0, time.UTC)},
		logger.NewNop(),
	)

	result, err := uc.Execute(context.Background(), DispatchJob{
		HistoryID:    1,
		WorkflowID:   "wf-1",
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

	if billingCalled {
		t.Fatal("billing stage should not be called when there are no eligible items")
	}
	if result.Billing.CreatedCount != 0 || result.Billing.DuplicateCount != 0 {
		t.Fatalf("unexpected billing result: %+v", result.Billing)
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
		&stubVendorResolutionStage{
			execute: func(ctx context.Context, cmd VendorResolutionCommand) (VendorResolutionResult, error) {
				t.Fatal("vendor resolution stage should not be called")
				return VendorResolutionResult{}, nil
			},
		},
		&stubBillingEligibilityStage{
			execute: func(ctx context.Context, cmd BillingEligibilityCommand) (BillingEligibilityResult, error) {
				t.Fatal("billing eligibility stage should not be called")
				return BillingEligibilityResult{}, nil
			},
		},
		&stubBillingStage{
			execute: func(ctx context.Context, cmd BillingCommand) (BillingResult, error) {
				t.Fatal("billing stage should not be called")
				return BillingResult{}, nil
			},
		},
		&stubWorkflowStatusRepository{},
		&fixedClock{now: time.Date(2026, 3, 25, 12, 30, 0, 0, time.UTC)},
		logger.NewNop(),
	)

	_, err := uc.Execute(context.Background(), DispatchJob{
		HistoryID:    1,
		WorkflowID:   "wf-1",
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

func TestUseCaseExecute_SavesProgressAndCompletesPartialSuccess(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	stageOrder := make([]string, 0, 4)
	savedProgress := make(map[string]StageProgress)
	completedStatus := ""

	uc := NewUseCase(
		&stubFetchStage{
			execute: func(ctx context.Context, cmd FetchCommand) (FetchResult, error) {
				return FetchResult{
					CreatedEmails: []CreatedEmail{
						{
							EmailID:           101,
							ExternalMessageID: "msg-1",
							Subject:           "Invoice",
							From:              "billing@example.com",
							To:                []string{"user@example.com"},
							ReceivedAt:        now,
							Body:              "invoice body",
							BodyDigest:        "digest-msg-1",
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
				return AnalyzeResult{
					ParsedEmails:     []ParsedEmail{{ParsedEmailID: 9001, EmailID: 101, ExternalMessageID: "msg-1"}},
					ParsedEmailCount: 1,
				}, nil
			},
		},
		&stubVendorResolutionStage{
			execute: func(ctx context.Context, cmd VendorResolutionCommand) (VendorResolutionResult, error) {
				return VendorResolutionResult{
					ResolvedItems:   nil,
					ResolvedCount:   0,
					UnresolvedCount: 1,
				}, nil
			},
		},
		&stubBillingEligibilityStage{
			execute: func(ctx context.Context, cmd BillingEligibilityCommand) (BillingEligibilityResult, error) {
				t.Fatal("billing eligibility should be skipped when nothing is resolved")
				return BillingEligibilityResult{}, nil
			},
		},
		&stubBillingStage{
			execute: func(ctx context.Context, cmd BillingCommand) (BillingResult, error) {
				t.Fatal("billing should be skipped when nothing is resolved")
				return BillingResult{}, nil
			},
		},
		&stubWorkflowStatusRepository{
			markRunning: func(ctx context.Context, historyID uint64, currentStage string) error {
				stageOrder = append(stageOrder, currentStage)
				return nil
			},
			saveStage: func(ctx context.Context, progress StageProgress) error {
				savedProgress[progress.Stage] = progress
				return nil
			},
			complete: func(ctx context.Context, historyID uint64, status string, finishedAt time.Time) error {
				completedStatus = status
				return nil
			},
		},
		&fixedClock{now: now},
		logger.NewNop(),
	)

	_, err := uc.Execute(context.Background(), DispatchJob{
		HistoryID:    77,
		WorkflowID:   "wf-progress",
		UserID:       1,
		ConnectionID: 2,
		Condition: FetchCondition{
			LabelName: "billing",
			Since:     now.Add(-time.Hour),
			Until:     now.Add(time.Hour),
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if len(stageOrder) != 3 {
		t.Fatalf("unexpected stage order count: %+v", stageOrder)
	}
	expectedOrder := []string{workflowStageFetch, workflowStageAnalysis, workflowStageVendorResolution}
	for idx, stage := range expectedOrder {
		if stageOrder[idx] != stage {
			t.Fatalf("unexpected stage order: %+v", stageOrder)
		}
	}

	fetchProgress, ok := savedProgress[workflowStageFetch]
	if !ok {
		t.Fatal("fetch progress was not saved")
	}
	if fetchProgress.SuccessCount != 2 || fetchProgress.BusinessFailureCount != 0 || fetchProgress.TechnicalFailureCount != 1 {
		t.Fatalf("unexpected fetch progress: %+v", fetchProgress)
	}

	vendorProgress, ok := savedProgress[workflowStageVendorResolution]
	if !ok {
		t.Fatal("vendorresolution progress was not saved")
	}
	if vendorProgress.SuccessCount != 0 || vendorProgress.BusinessFailureCount != 1 || vendorProgress.TechnicalFailureCount != 0 {
		t.Fatalf("unexpected vendor progress: %+v", vendorProgress)
	}
	if len(vendorProgress.FailureRecords) != 1 || vendorProgress.FailureRecords[0].ReasonCode != reasonCodeVendorUnresolved {
		t.Fatalf("unexpected vendor failure records: %+v", vendorProgress.FailureRecords)
	}

	if completedStatus != WorkflowStatusPartialSuccess {
		t.Fatalf("unexpected completed status: %s", completedStatus)
	}
}

func TestUseCaseExecute_FetchErrorMarksWorkflowFailedWithErrorMessage(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("failed to create gmail service: invalid_grant")
	failCalled := 0

	uc := NewUseCase(
		&stubFetchStage{
			execute: func(ctx context.Context, cmd FetchCommand) (FetchResult, error) {
				return FetchResult{}, wantErr
			},
		},
		&stubAnalyzeStage{
			execute: func(ctx context.Context, cmd AnalyzeCommand) (AnalyzeResult, error) {
				t.Fatal("analyze stage should not be called")
				return AnalyzeResult{}, nil
			},
		},
		&stubVendorResolutionStage{
			execute: func(ctx context.Context, cmd VendorResolutionCommand) (VendorResolutionResult, error) {
				t.Fatal("vendor resolution stage should not be called")
				return VendorResolutionResult{}, nil
			},
		},
		&stubBillingEligibilityStage{
			execute: func(ctx context.Context, cmd BillingEligibilityCommand) (BillingEligibilityResult, error) {
				t.Fatal("billing eligibility stage should not be called")
				return BillingEligibilityResult{}, nil
			},
		},
		&stubBillingStage{
			execute: func(ctx context.Context, cmd BillingCommand) (BillingResult, error) {
				t.Fatal("billing stage should not be called")
				return BillingResult{}, nil
			},
		},
		&stubWorkflowStatusRepository{
			fail: func(ctx context.Context, historyID uint64, currentStage string, finishedAt time.Time, errorMessage string) error {
				failCalled++
				if historyID != 1 {
					t.Fatalf("unexpected history id: %d", historyID)
				}
				if currentStage != workflowStageFetch {
					t.Fatalf("unexpected current stage: %q", currentStage)
				}
				if errorMessage != "Gmail連携が無効になっています。再連携してください。" {
					t.Fatalf("unexpected error message: %q", errorMessage)
				}
				return nil
			},
		},
		&fixedClock{now: time.Date(2026, 3, 25, 12, 30, 0, 0, time.UTC)},
		logger.NewNop(),
	)

	_, err := uc.Execute(context.Background(), DispatchJob{
		HistoryID:    1,
		WorkflowID:   "wf-failed",
		UserID:       1,
		ConnectionID: 2,
		Condition: FetchCondition{
			LabelName: "billing",
			Since:     time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
			Until:     time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
		},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected fetch error, got %v", err)
	}
	if failCalled != 1 {
		t.Fatalf("expected fail to be called once, got %d", failCalled)
	}
}

func stringPtr(value string) *string {
	return &value
}

func float64Ptr(value float64) *float64 {
	return &value
}
