package domain

import (
	"errors"
	"testing"
	"time"
)

// 観点:
// - resolved / unresolved / invalid vendor の基本整合性を守ること
func TestVendorResolutionValidate(t *testing.T) {
	t.Parallel()

	t.Run("resolved vendor is valid", func(t *testing.T) {
		t.Parallel()

		resolution := VendorResolution{
			ResolvedVendor: &Vendor{Name: "AWS"},
		}

		if !resolution.IsResolved() {
			t.Fatalf("expected resolution to be resolved")
		}
		if err := resolution.Validate(); err != nil {
			t.Fatalf("expected resolution to be valid, got %v", err)
		}
	})

	t.Run("unresolved vendor returns error", func(t *testing.T) {
		t.Parallel()

		var resolution VendorResolution
		if resolution.IsResolved() {
			t.Fatalf("expected resolution to be unresolved")
		}
		if err := resolution.Validate(); !errors.Is(err, ErrVendorResolutionUnresolved) {
			t.Fatalf("expected ErrVendorResolutionUnresolved, got %v", err)
		}
	})

	t.Run("invalid resolved vendor returns vendor error", func(t *testing.T) {
		t.Parallel()

		resolution := VendorResolution{
			ResolvedVendor: &Vendor{Name: "  "},
		}

		if err := resolution.Validate(); !errors.Is(err, ErrVendorNameEmpty) {
			t.Fatalf("expected ErrVendorNameEmpty, got %v", err)
		}
	})
}

// 観点:
// - fetch plan は判定に必要な検索キーを 1 回で組み立てること
func TestVendorResolutionPolicyBuildFetchPlan(t *testing.T) {
	t.Parallel()

	policy := VendorResolutionPolicy{}

	plan := policy.BuildFetchPlan(VendorResolutionInput{
		CandidateVendorName: stringPtrVendorResolution(" Acme Corp "),
		Subject:             "  Your Acme Invoice ",
		From:                "Acme Billing <billing@acme.example.com> ",
		To:                  []string{" user@example.com ", ""},
	})

	if plan.NameExactValue != "acme corp" {
		t.Fatalf("unexpected name exact value: %+v", plan)
	}
	if plan.SenderDomainValue != "acme.example.com" {
		t.Fatalf("unexpected sender domain value: %+v", plan)
	}
	if plan.SenderNameValue != "acme billing" {
		t.Fatalf("unexpected sender name value: %+v", plan)
	}
	if plan.SubjectValue != "your acme invoice" {
		t.Fatalf("unexpected subject value: %+v", plan)
	}
}

// 観点:
// - 判定優先順位は name_exact -> sender_domain -> sender_name -> subject_keyword の順であること
// - exact 系の競合は created_at DESC / alias_id DESC の最新が勝つこと
func TestVendorResolutionPolicyResolve_PrefersHigherPriorityAndLatestAlias(t *testing.T) {
	t.Parallel()

	policy := VendorResolutionPolicy{}
	decision := policy.Resolve(VendorResolutionFacts{
		NameExactCandidates: []VendorAliasCandidate{
			aliasCandidate(1, MatchedByNameExact, "acme", Vendor{ID: 10, Name: "Acme Old"}, testTime(9, 0)),
			aliasCandidate(2, MatchedByNameExact, "acme", Vendor{ID: 20, Name: "Acme New"}, testTime(9, 10)),
		},
		SenderDomainCandidates: []VendorAliasCandidate{
			aliasCandidate(3, MatchedBySenderDomain, "acme.example.com", Vendor{ID: 30, Name: "Domain Vendor"}, testTime(9, 20)),
		},
	})

	if !decision.Resolution.IsResolved() {
		t.Fatalf("expected decision to be resolved")
	}
	if decision.Resolution.ResolvedVendor.ID != 20 {
		t.Fatalf("unexpected resolved vendor: %+v", decision.Resolution.ResolvedVendor)
	}
	if decision.MatchedBy != MatchedByNameExact {
		t.Fatalf("unexpected matched by: %+v", decision)
	}
}

// 観点:
// - subject_keyword は最長一致を優先すること
// - 同長で複数 vendor に競合したら unresolved になること
func TestVendorResolutionPolicyResolve_SubjectKeywordRules(t *testing.T) {
	t.Parallel()

	policy := VendorResolutionPolicy{}

	t.Run("uses longest match", func(t *testing.T) {
		t.Parallel()

		decision := policy.Resolve(VendorResolutionFacts{
			SubjectKeywordCandidates: []VendorAliasCandidate{
				aliasCandidate(1, MatchedBySubjectKeyword, "invoice", Vendor{ID: 10, Name: "Short"}, testTime(9, 0)),
				aliasCandidate(2, MatchedBySubjectKeyword, "acme cloud invoice", Vendor{ID: 20, Name: "Long"}, testTime(9, 10)),
			},
		})

		if !decision.Resolution.IsResolved() {
			t.Fatalf("expected decision to be resolved")
		}
		if decision.Resolution.ResolvedVendor.ID != 20 || decision.MatchedBy != MatchedBySubjectKeyword {
			t.Fatalf("unexpected decision: %+v", decision)
		}
	})

	t.Run("returns unresolved on ambiguity", func(t *testing.T) {
		t.Parallel()

		decision := policy.Resolve(VendorResolutionFacts{
			SubjectKeywordCandidates: []VendorAliasCandidate{
				aliasCandidate(1, MatchedBySubjectKeyword, "invoice ready", Vendor{ID: 10, Name: "A"}, testTime(9, 0)),
				aliasCandidate(2, MatchedBySubjectKeyword, "invoice ready", Vendor{ID: 20, Name: "B"}, testTime(9, 10)),
			},
		})

		if decision.Resolution.IsResolved() {
			t.Fatalf("expected decision to be unresolved: %+v", decision)
		}
		if decision.MatchedBy != "" {
			t.Fatalf("unexpected matched by: %+v", decision)
		}
	})
}

// 観点:
// - unresolved の candidate vendor 名から canonical vendor + name_exact alias の登録計画を作ること
// - 既に resolved なら登録計画を作らないこと
func TestVendorResolutionPolicyBuildRegistrationPlan(t *testing.T) {
	t.Parallel()

	policy := VendorResolutionPolicy{}
	input := VendorResolutionInput{
		CandidateVendorName: stringPtrVendorResolution(" アマゾンジャパン合同会社 "),
	}

	plan := policy.BuildRegistrationPlan(input, VendorResolutionDecision{})
	if plan == nil {
		t.Fatal("expected registration plan")
	}
	if plan.VendorName != "アマゾンジャパン合同会社" {
		t.Fatalf("unexpected vendor name: %+v", plan)
	}
	if plan.NormalizedVendorName != "アマゾンジャパン合同会社" {
		t.Fatalf("unexpected normalized vendor name: %+v", plan)
	}
	if len(plan.Aliases) != 1 || plan.Aliases[0].AliasType != MatchedByNameExact {
		t.Fatalf("unexpected aliases: %+v", plan.Aliases)
	}

	resolvedPlan := policy.BuildRegistrationPlan(input, policy.ResolveRegisteredVendor(Vendor{ID: 1, Name: "Amazon"}))
	if resolvedPlan != nil {
		t.Fatalf("expected no registration plan for resolved decision: %+v", resolvedPlan)
	}
}

func aliasCandidate(aliasID uint, aliasType, normalizedValue string, vendor Vendor, createdAt time.Time) VendorAliasCandidate {
	return VendorAliasCandidate{
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

func stringPtrVendorResolution(value string) *string {
	return &value
}
