package application

import (
	commondomain "business/internal/common/domain"
	"business/internal/library/logger"
	"business/internal/vendorresolution/domain"
	"context"
	"errors"
	"testing"
	"time"
)

type stubVendorResolutionRepository struct {
	fetchFacts func(ctx context.Context, plan domain.VendorResolutionFetchPlan) (domain.VendorResolutionFacts, error)
}

// FetchFacts は usecase テスト用の resolution repository stub。
func (s *stubVendorResolutionRepository) FetchFacts(ctx context.Context, plan domain.VendorResolutionFetchPlan) (domain.VendorResolutionFacts, error) {
	return s.fetchFacts(ctx, plan)
}

type stubVendorRegistrationRepository struct {
	ensureByPlan func(ctx context.Context, plan domain.VendorRegistrationPlan) (*commondomain.Vendor, error)
}

// EnsureByPlan は usecase テスト用の registration repository stub。
func (s *stubVendorRegistrationRepository) EnsureByPlan(ctx context.Context, plan domain.VendorRegistrationPlan) (*commondomain.Vendor, error) {
	return s.ensureByPlan(ctx, plan)
}

// 観点:
// - resolved / unresolved / validation failure / fetch failure を 1 回の実行で併存できること
// - unresolved_external_message_ids が重複なく集約されること
// - 部分失敗が stage 全体失敗ではなく Failures に積まれること
func TestUseCaseExecute_MixedOutcome(t *testing.T) {
	t.Parallel()

	uc := NewUseCase(
		&stubVendorResolutionRepository{
			fetchFacts: func(ctx context.Context, plan domain.VendorResolutionFetchPlan) (domain.VendorResolutionFacts, error) {
				switch {
				case plan.NameExactValue == "acme":
					return domain.VendorResolutionFacts{
						NameExactCandidates: []commondomain.VendorAliasCandidate{
							aliasCandidate(1, domain.MatchedByNameExact, "acme", commondomain.Vendor{ID: 700, Name: "Acme"}, testTime(9, 0)),
						},
					}, nil
				case plan.NameExactValue == "unknown":
					return domain.VendorResolutionFacts{}, nil
				case plan.SubjectValue == "resolverfail":
					return domain.VendorResolutionFacts{}, errors.New("fetch failed")
				default:
					t.Fatalf("unexpected fetch plan: %+v", plan)
					return domain.VendorResolutionFacts{}, nil
				}
			},
		},
		&stubVendorRegistrationRepository{
			ensureByPlan: func(ctx context.Context, plan domain.VendorRegistrationPlan) (*commondomain.Vendor, error) {
				return nil, nil
			},
		},
		logger.NewNop(),
	)

	result, err := uc.Execute(context.Background(), Command{
		UserID: 10,
		ParsedEmails: []ResolutionTarget{
			{
				ParsedEmailID:     1,
				EmailID:           11,
				ExternalMessageID: "msg-1",
				Subject:           "Invoice",
				From:              "billing@acme.test",
				ParsedEmail:       commondomain.ParsedEmail{VendorName: stringPtr(" Acme ")},
			},
			{
				ParsedEmailID:     2,
				EmailID:           22,
				ExternalMessageID: "msg-2",
				Subject:           "Reminder",
				From:              "notice@example.test",
				ParsedEmail:       commondomain.ParsedEmail{VendorName: stringPtr(" Unknown ")},
			},
			{
				ParsedEmailID:     0,
				EmailID:           33,
				ExternalMessageID: "msg-3",
				Subject:           "Invalid",
				From:              "notice@example.test",
				ParsedEmail:       commondomain.ParsedEmail{VendorName: stringPtr("Broken")},
			},
			{
				ParsedEmailID:     4,
				EmailID:           44,
				ExternalMessageID: "msg-4",
				Subject:           "ResolverFail",
				From:              "ops@example.test",
				ParsedEmail:       commondomain.ParsedEmail{VendorName: stringPtr("ResolverFail")},
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if result.ResolvedCount != 1 {
		t.Fatalf("expected 1 resolved item, got %d", result.ResolvedCount)
	}
	if result.UnresolvedCount != 1 {
		t.Fatalf("expected 1 unresolved item, got %d", result.UnresolvedCount)
	}
	if len(result.UnresolvedItems) != 1 {
		t.Fatalf("expected 1 unresolved item detail, got %+v", result.UnresolvedItems)
	}
	if len(result.ResolvedItems) != 1 {
		t.Fatalf("expected 1 resolved item detail, got %+v", result.ResolvedItems)
	}
	if result.ResolvedItems[0].VendorName != "Acme" || result.ResolvedItems[0].MatchedBy != domain.MatchedByNameExact {
		t.Fatalf("unexpected resolved item: %+v", result.ResolvedItems[0])
	}
	if result.UnresolvedItems[0].ReasonCode != domain.ReasonCodeVendorUnresolved {
		t.Fatalf("unexpected unresolved item: %+v", result.UnresolvedItems[0])
	}
	if result.UnresolvedItems[0].ExternalMessageID != "msg-2" || result.UnresolvedItems[0].CandidateVendorName != "Unknown" {
		t.Fatalf("unexpected unresolved item: %+v", result.UnresolvedItems[0])
	}
	if result.UnresolvedItems[0].Message != "msg-2 の候補「Unknown」を支払先として特定できませんでした。" {
		t.Fatalf("unexpected unresolved message: %+v", result.UnresolvedItems[0])
	}
	if len(result.Failures) != 2 {
		t.Fatalf("expected 2 failures, got %+v", result.Failures)
	}

	expectedFailures := []domain.Failure{
		{
			EmailID:           33,
			ExternalMessageID: "msg-3",
			Stage:             domain.FailureStageNormalizeInput,
			Code:              domain.FailureCodeInvalidResolutionTarget,
			Message:           "msg-3 の候補「Broken」を使った支払先解決入力が不正でした。",
		},
		{
			ParsedEmailID:     4,
			EmailID:           44,
			ExternalMessageID: "msg-4",
			Stage:             domain.FailureStageResolveVendor,
			Code:              domain.FailureCodeVendorResolveFail,
			Message:           "msg-4 の候補「ResolverFail」の支払先解決に失敗しました。",
		},
	}
	for idx, failure := range expectedFailures {
		if result.Failures[idx] != failure {
			t.Fatalf("unexpected failure at %d: %+v", idx, result.Failures[idx])
		}
	}
}

// 観点:
// - user_id 不正時は下流 repository を呼ばずに command error を返すこと
func TestUseCaseExecute_InvalidCommand(t *testing.T) {
	t.Parallel()

	uc := NewUseCase(
		&stubVendorResolutionRepository{fetchFacts: func(ctx context.Context, plan domain.VendorResolutionFetchPlan) (domain.VendorResolutionFacts, error) {
			t.Fatal("resolution repository should not be called")
			return domain.VendorResolutionFacts{}, nil
		}},
		&stubVendorRegistrationRepository{ensureByPlan: func(ctx context.Context, plan domain.VendorRegistrationPlan) (*commondomain.Vendor, error) {
			t.Fatal("registration repository should not be called")
			return nil, nil
		}},
		logger.NewNop(),
	)

	_, err := uc.Execute(context.Background(), Command{
		UserID: 0,
		ParsedEmails: []ResolutionTarget{
			{ParsedEmailID: 1, EmailID: 1},
		},
	})
	if !errors.Is(err, domain.ErrInvalidCommand) {
		t.Fatalf("expected ErrInvalidCommand, got %v", err)
	}
}

// 観点:
// - facts から unresolved の場合に candidate vendor 名から自動登録して解決へ進めること
// - 自動登録後の matched_by は name_exact として返ること
func TestUseCaseExecute_AutoRegistersCandidateVendor(t *testing.T) {
	t.Parallel()

	fetchCalls := 0
	registrationCalls := 0

	uc := NewUseCase(
		&stubVendorResolutionRepository{
			fetchFacts: func(ctx context.Context, plan domain.VendorResolutionFetchPlan) (domain.VendorResolutionFacts, error) {
				fetchCalls++
				return domain.VendorResolutionFacts{}, nil
			},
		},
		&stubVendorRegistrationRepository{
			ensureByPlan: func(ctx context.Context, plan domain.VendorRegistrationPlan) (*commondomain.Vendor, error) {
				registrationCalls++
				if plan.VendorName != "アマゾンジャパン合同会社" {
					t.Fatalf("unexpected plan: %+v", plan)
				}
				if plan.NormalizedVendorName != "アマゾンジャパン合同会社" {
					t.Fatalf("unexpected plan: %+v", plan)
				}
				if len(plan.Aliases) != 1 || plan.Aliases[0].AliasType != domain.MatchedByNameExact {
					t.Fatalf("unexpected plan: %+v", plan)
				}
				return &commondomain.Vendor{ID: 501, Name: plan.VendorName}, nil
			},
		},
		logger.NewNop(),
	)

	result, err := uc.Execute(context.Background(), Command{
		UserID: 1,
		ParsedEmails: []ResolutionTarget{
			{
				ParsedEmailID:     10,
				EmailID:           100,
				ExternalMessageID: "msg-1",
				Subject:           "Fwd: 注文済み",
				From:              "\"遠藤、\" <pikachu.endu@gmail.com>",
				ParsedEmail:       commondomain.ParsedEmail{VendorName: stringPtr("アマゾンジャパン合同会社")},
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if fetchCalls != 1 {
		t.Fatalf("expected 1 fetch call, got %d", fetchCalls)
	}
	if registrationCalls != 1 {
		t.Fatalf("expected 1 registration call, got %d", registrationCalls)
	}
	if result.ResolvedCount != 1 || result.UnresolvedCount != 0 {
		t.Fatalf("unexpected result counters: %+v", result)
	}
	if len(result.ResolvedItems) != 1 {
		t.Fatalf("unexpected resolved items: %+v", result.ResolvedItems)
	}
	if result.ResolvedItems[0].VendorID != 501 || result.ResolvedItems[0].MatchedBy != domain.MatchedByNameExact {
		t.Fatalf("unexpected resolved item: %+v", result.ResolvedItems[0])
	}
}

// 観点:
// - 自動登録で DB 書き込みに失敗した場合は unresolved ではなく technical failure として返すこと
func TestUseCaseExecute_AutoRegisterFailureBecomesFailure(t *testing.T) {
	t.Parallel()

	uc := NewUseCase(
		&stubVendorResolutionRepository{
			fetchFacts: func(ctx context.Context, plan domain.VendorResolutionFetchPlan) (domain.VendorResolutionFacts, error) {
				return domain.VendorResolutionFacts{}, nil
			},
		},
		&stubVendorRegistrationRepository{
			ensureByPlan: func(ctx context.Context, plan domain.VendorRegistrationPlan) (*commondomain.Vendor, error) {
				return nil, errors.New("insert failed")
			},
		},
		logger.NewNop(),
	)

	result, err := uc.Execute(context.Background(), Command{
		UserID: 1,
		ParsedEmails: []ResolutionTarget{
			{
				ParsedEmailID:     10,
				EmailID:           100,
				ExternalMessageID: "msg-1",
				ParsedEmail:       commondomain.ParsedEmail{VendorName: stringPtr("Acme")},
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if result.ResolvedCount != 0 || result.UnresolvedCount != 0 {
		t.Fatalf("unexpected result counters: %+v", result)
	}
	if len(result.UnresolvedItems) != 0 {
		t.Fatalf("unexpected unresolved details: %+v", result)
	}
	if len(result.Failures) != 1 {
		t.Fatalf("unexpected failures: %+v", result.Failures)
	}
	if result.Failures[0].Stage != domain.FailureStageRegisterVendor || result.Failures[0].Code != domain.FailureCodeVendorRegisterFail {
		t.Fatalf("unexpected failure: %+v", result.Failures[0])
	}
	if result.Failures[0].Message != "msg-1 の候補「Acme」の支払先登録に失敗しました。" {
		t.Fatalf("unexpected failure message: %+v", result.Failures[0])
	}
}

func aliasCandidate(aliasID uint, aliasType, normalizedValue string, vendor commondomain.Vendor, createdAt time.Time) commondomain.VendorAliasCandidate {
	return commondomain.VendorAliasCandidate{
		AliasID:         aliasID,
		AliasType:       aliasType,
		AliasValue:      normalizedValue,
		NormalizedValue: normalizedValue,
		AliasCreatedAt:  createdAt,
		Vendor:          vendor,
	}
}

func testTime(hour, minute int) time.Time {
	return time.Date(2026, 3, 24, hour, minute, 0, 0, time.UTC)
}

// stringPtr はテストデータ作成用の小さな helper。
func stringPtr(value string) *string {
	return &value
}
