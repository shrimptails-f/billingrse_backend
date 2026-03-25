package application

import (
	commondomain "business/internal/common/domain"
	mfdomain "business/internal/mailfetch/domain"
	"fmt"
	"testing"
)

func TestBuildStageProgress_PrefersStageProvidedMessages(t *testing.T) {
	t.Parallel()

	fetchProgress := buildFetchStageProgress(1, FetchResult{
		Failures: []FetchFailure{
			{ExternalMessageID: "msg-fetch", Code: "email_save_failed", Message: "fetch stage message"},
		},
	})
	if len(fetchProgress.FailureRecords) != 1 || fetchProgress.FailureRecords[0].Message != "fetch stage message" {
		t.Fatalf("expected fetch message to be preserved, got %+v", fetchProgress.FailureRecords)
	}

	analysisProgress := buildAnalysisStageProgress(1, AnalyzeResult{
		Failures: []AnalysisFailure{
			{ExternalMessageID: "msg-analysis", Code: "analysis_failed", Message: "analysis stage message"},
		},
	})
	if len(analysisProgress.FailureRecords) != 1 || analysisProgress.FailureRecords[0].Message != "analysis stage message" {
		t.Fatalf("expected analysis message to be preserved, got %+v", analysisProgress.FailureRecords)
	}

	vendorProgress := buildVendorResolutionStageProgress(1, nil, VendorResolutionResult{
		UnresolvedItems: []UnresolvedItem{
			{ExternalMessageID: "msg-vendor-unresolved", ReasonCode: reasonCodeVendorUnresolved, Message: "vendor unresolved message"},
		},
		UnresolvedCount: 1,
		Failures: []VendorResolutionFailure{
			{ExternalMessageID: "msg-vendor-failure", Code: "vendor_resolution_failed", Message: "vendor failure message"},
		},
	})
	if len(vendorProgress.FailureRecords) != 2 {
		t.Fatalf("unexpected vendor failure records: %+v", vendorProgress.FailureRecords)
	}
	if vendorProgress.FailureRecords[0].Message != "vendor unresolved message" || vendorProgress.FailureRecords[1].Message != "vendor failure message" {
		t.Fatalf("expected vendor messages to be preserved, got %+v", vendorProgress.FailureRecords)
	}

	billingEligibilityProgress := buildBillingEligibilityStageProgress(1, BillingEligibilityResult{
		IneligibleItems: []IneligibleItem{
			{ExternalMessageID: "msg-ineligible", ReasonCode: "amount_empty", Message: "billing eligibility business message"},
		},
		IneligibleCount: 1,
		Failures: []BillingEligibilityFailure{
			{ExternalMessageID: "msg-eligibility-failure", Code: "billing_eligibility_failed", Message: "billing eligibility failure message"},
		},
	})
	if len(billingEligibilityProgress.FailureRecords) != 2 {
		t.Fatalf("unexpected billing eligibility failure records: %+v", billingEligibilityProgress.FailureRecords)
	}
	if billingEligibilityProgress.FailureRecords[0].Message != "billing eligibility business message" || billingEligibilityProgress.FailureRecords[1].Message != "billing eligibility failure message" {
		t.Fatalf("expected billing eligibility messages to be preserved, got %+v", billingEligibilityProgress.FailureRecords)
	}

	billingProgress := buildBillingStageProgress(1, BillingResult{
		DuplicateItems: []BillingDuplicateItem{
			{ExternalMessageID: "msg-duplicate", ReasonCode: reasonCodeDuplicateBilling, Message: "billing duplicate message"},
		},
		DuplicateCount: 1,
		Failures: []BillingFailure{
			{ExternalMessageID: "msg-billing-failure", Code: "billing_persist_failed", Message: "billing failure message"},
		},
	})
	if len(billingProgress.FailureRecords) != 2 {
		t.Fatalf("unexpected billing failure records: %+v", billingProgress.FailureRecords)
	}
	if billingProgress.FailureRecords[0].Message != "billing duplicate message" || billingProgress.FailureRecords[1].Message != "billing failure message" {
		t.Fatalf("expected billing messages to be preserved, got %+v", billingProgress.FailureRecords)
	}
}

func TestBuildVendorResolutionStageProgress_InferredUnresolvedMessageIncludesContext(t *testing.T) {
	t.Parallel()

	progress := buildVendorResolutionStageProgress(1, []ParsedEmail{
		{
			ParsedEmailID:     9001,
			ExternalMessageID: "msg-1",
			Data: commondomain.ParsedEmail{
				VendorName: stringPtr("Mystery Service"),
			},
		},
	}, VendorResolutionResult{})

	if len(progress.FailureRecords) != 1 {
		t.Fatalf("unexpected inferred vendor failure records: %+v", progress.FailureRecords)
	}
	if progress.FailureRecords[0].Message != "msg-1 の候補「Mystery Service」を支払先として特定できませんでした。" {
		t.Fatalf("unexpected inferred unresolved message: %+v", progress.FailureRecords[0])
	}
}

func TestLocalizedWorkflowErrorMessage_FetchStageUsesJapaneseMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "connection not found",
			err:  mfdomain.ErrConnectionNotFound,
			want: "指定したGmail連携が見つかりませんでした。",
		},
		{
			name: "label not found",
			err:  mfdomain.ErrProviderLabelNotFound,
			want: "指定したGmailラベルが見つかりませんでした。",
		},
		{
			name: "decrypt access token",
			err:  fmt.Errorf("%w: failed to decrypt access token: broken secret", mfdomain.ErrProviderSessionBuildFailed),
			want: "Gmail連携のアクセストークンを復号できませんでした。再連携をおねがいします。",
		},
		{
			name: "invalid grant",
			err:  fmt.Errorf("%w: failed to create gmail service: invalid_grant", mfdomain.ErrProviderSessionBuildFailed),
			want: "Gmail連携が無効になっています。再連携してください。",
		},
		{
			name: "oauth config",
			err:  fmt.Errorf("%w: failed to load gmail oauth config: missing file", mfdomain.ErrProviderSessionBuildFailed),
			want: "Gmail OAuth設定の読み込みに失敗しました。システム設定を確認してください。",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := localizedWorkflowErrorMessage(workflowStageFetch, tt.err); got != tt.want {
				t.Fatalf("unexpected localized message: got=%q want=%q", got, tt.want)
			}
		})
	}
}
