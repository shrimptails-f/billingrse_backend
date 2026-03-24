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

func TestUseCaseExecute_FetchThenAnalyze(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)
	fetchCalls := 0
	analyzeCalls := 0
	vendorResolutionCalls := 0
	billingEligibilityCalls := 0
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
					ParsedEmailIDs: []uint{9001, 9002},
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
					AnalyzedEmailCount: 1,
					ParsedEmailCount:   2,
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
							BodyDigest:        "digest-msg-1",
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
					ResolvedCount:                1,
					UnresolvedCount:              1,
					UnresolvedExternalMessageIDs: []string{"msg-1"},
					Failures: []VendorResolutionFailure{
						{ParsedEmailID: 9002, EmailID: 101, ExternalMessageID: "msg-1", Stage: "resolve_vendor", Code: "vendor_resolution_failed"},
					},
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
	if vendorResolutionCalls != 1 {
		t.Fatalf("expected 1 vendor resolution call, got %d", vendorResolutionCalls)
	}
	if billingEligibilityCalls != 1 {
		t.Fatalf("expected 1 billing eligibility call, got %d", billingEligibilityCalls)
	}
	if result.Fetch.Provider != "gmail" {
		t.Fatalf("unexpected fetch result: %+v", result.Fetch)
	}
	if len(result.Analysis.ParsedEmailIDs) != 2 {
		t.Fatalf("unexpected analysis result: %+v", result.Analysis)
	}
	if result.VendorResolution.ResolvedCount != 1 || result.VendorResolution.UnresolvedCount != 1 {
		t.Fatalf("unexpected vendor resolution result: %+v", result.VendorResolution)
	}
	if result.BillingEligibility.EligibleCount != 1 || len(result.BillingEligibility.EligibleItems) != 1 {
		t.Fatalf("unexpected billing eligibility result: %+v", result.BillingEligibility)
	}
}

func TestUseCaseExecute_SkipsAnalyzeWhenNoCreatedEmails(t *testing.T) {
	t.Parallel()

	analyzeCalled := false
	vendorResolutionCalled := false
	billingEligibilityCalled := false
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
	if vendorResolutionCalled {
		t.Fatal("vendor resolution stage should not be called when there are no created emails")
	}
	if billingEligibilityCalled {
		t.Fatal("billing eligibility stage should not be called when there are no created emails")
	}
	if len(result.Analysis.ParsedEmailIDs) != 0 || result.Analysis.ParsedEmailCount != 0 {
		t.Fatalf("unexpected analysis result: %+v", result.Analysis)
	}
}

func TestUseCaseExecute_SkipsVendorResolutionWhenNoParsedEmails(t *testing.T) {
	t.Parallel()

	vendorResolutionCalled := false
	billingEligibilityCalled := false
	uc := NewUseCase(
		&stubFetchStage{
			execute: func(ctx context.Context, cmd FetchCommand) (FetchResult, error) {
				return FetchResult{
					Provider:          "gmail",
					AccountIdentifier: "user@example.com",
					CreatedEmailIDs:   []uint{101},
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
					ParsedEmails:       nil,
					AnalyzedEmailCount: 1,
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

	if vendorResolutionCalled {
		t.Fatal("vendor resolution stage should not be called when there are no parsed emails")
	}
	if billingEligibilityCalled {
		t.Fatal("billing eligibility stage should not be called when there are no parsed emails")
	}
	if result.VendorResolution.ResolvedCount != 0 || result.VendorResolution.UnresolvedCount != 0 {
		t.Fatalf("unexpected vendor resolution result: %+v", result.VendorResolution)
	}
}

func TestUseCaseExecute_SkipsBillingEligibilityWhenNoResolvedItems(t *testing.T) {
	t.Parallel()

	billingEligibilityCalled := false
	uc := NewUseCase(
		&stubFetchStage{
			execute: func(ctx context.Context, cmd FetchCommand) (FetchResult, error) {
				return FetchResult{
					CreatedEmailIDs: []uint{101},
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
					ParsedEmailIDs: []uint{9001},
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

	if billingEligibilityCalled {
		t.Fatal("billing eligibility stage should not be called when there are no resolved items")
	}
	if result.BillingEligibility.EligibleCount != 0 || result.BillingEligibility.IneligibleCount != 0 {
		t.Fatalf("unexpected billing eligibility result: %+v", result.BillingEligibility)
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

func stringPtr(value string) *string {
	return &value
}

func float64Ptr(value float64) *float64 {
	return &value
}
