package application

import (
	commondomain "business/internal/common/domain"
	"business/internal/library/logger"
	"business/internal/vendorresolution/domain"
	"context"
	"fmt"
	"strings"
)

// resolveTarget は repository で材料を集め、policy で 1 回の最終判定を行う。
func (uc *useCase) resolveTarget(ctx context.Context, userID uint, target ResolutionTarget, reqLog logger.Interface) (domain.ResolutionDecision, *domain.Failure, error) {
	input := buildVendorResolutionInput(target)
	plan := uc.policy.BuildFetchPlan(input)
	plan.UserID = userID

	facts, err := uc.resolutionRepository.FetchFacts(ctx, plan)
	if err != nil {
		return domain.ResolutionDecision{}, newFailure(target, domain.FailureStageResolveVendor, domain.FailureCodeVendorResolveFail, messageForResolutionFailure(target, domain.FailureCodeVendorResolveFail)), err
	}

	decision := uc.policy.Resolve(facts)
	if decision.Resolution.IsResolved() {
		return decision, nil, nil
	}

	decision, err = uc.ensureVendorByCandidateName(ctx, userID, target, input, decision, reqLog)
	if err != nil {
		return domain.ResolutionDecision{}, newFailure(target, domain.FailureStageRegisterVendor, domain.FailureCodeVendorRegisterFail, messageForResolutionFailure(target, domain.FailureCodeVendorRegisterFail)), err
	}

	return decision, nil, nil
}

// buildVendorResolutionInput は workflow 入力から domain policy 用の事実入力を作る。
func buildVendorResolutionInput(target ResolutionTarget) commondomain.VendorResolutionInput {
	return commondomain.VendorResolutionInput{
		CandidateVendorName: target.ParsedEmail.VendorName,
		Subject:             target.Subject,
		From:                target.From,
		To:                  append([]string(nil), target.To...),
	}.Normalize()
}

// newFailure は target に紐づく failure を一定の形式で生成する。
func newFailure(target ResolutionTarget, stage, code, message string) *domain.Failure {
	return &domain.Failure{
		ParsedEmailID:     target.ParsedEmailID,
		EmailID:           target.EmailID,
		ExternalMessageID: target.ExternalMessageID,
		Stage:             stage,
		Code:              code,
		Message:           message,
	}
}

func messageForUnresolvedItem(target ResolutionTarget) string {
	return fmt.Sprintf("%s の候補「%s」を支払先として特定できませんでした。",
		externalMessageIDText(target.ExternalMessageID),
		candidateVendorText(target),
	)
}

func messageForResolutionFailure(target ResolutionTarget, code string) string {
	switch code {
	case domain.FailureCodeInvalidResolutionTarget:
		return fmt.Sprintf("%s の候補「%s」を使った支払先解決入力が不正でした。",
			externalMessageIDText(target.ExternalMessageID),
			candidateVendorText(target),
		)
	case domain.FailureCodeVendorResolveFail:
		return fmt.Sprintf("%s の候補「%s」の支払先解決に失敗しました。",
			externalMessageIDText(target.ExternalMessageID),
			candidateVendorText(target),
		)
	case domain.FailureCodeVendorRegisterFail:
		return fmt.Sprintf("%s の候補「%s」の支払先登録に失敗しました。",
			externalMessageIDText(target.ExternalMessageID),
			candidateVendorText(target),
		)
	default:
		return fmt.Sprintf("%s の支払先解決でエラーが発生しました。",
			externalMessageIDText(target.ExternalMessageID),
		)
	}
}

func candidateVendorText(target ResolutionTarget) string {
	value := strings.TrimSpace(stringValue(target.ParsedEmail.VendorName))
	if value == "" {
		return "不明"
	}
	return value
}

func externalMessageIDText(externalMessageID string) string {
	value := strings.TrimSpace(externalMessageID)
	if value == "" {
		return "不明なメッセージ"
	}
	return value
}
