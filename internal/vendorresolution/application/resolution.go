package application

import (
	commondomain "business/internal/common/domain"
	"business/internal/library/logger"
	"business/internal/vendorresolution/domain"
	"context"
)

// resolveTarget は repository で材料を集め、policy で 1 回の最終判定を行う。
func (uc *useCase) resolveTarget(ctx context.Context, userID uint, target ResolutionTarget, reqLog logger.Interface) (domain.ResolutionDecision, *domain.Failure, error) {
	input := buildVendorResolutionInput(target)
	facts, err := uc.resolutionRepository.FetchFacts(ctx, uc.policy.BuildFetchPlan(input))
	if err != nil {
		return domain.ResolutionDecision{}, newFailure(target, domain.FailureStageResolveVendor, domain.FailureCodeVendorResolveFail), err
	}

	decision := uc.policy.Resolve(facts)
	if decision.Resolution.IsResolved() {
		return decision, nil, nil
	}

	decision, err = uc.ensureVendorByCandidateName(ctx, userID, target, input, decision, reqLog)
	if err != nil {
		return domain.ResolutionDecision{}, newFailure(target, domain.FailureStageRegisterVendor, domain.FailureCodeVendorRegisterFail), err
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
func newFailure(target ResolutionTarget, stage, code string) *domain.Failure {
	return &domain.Failure{
		ParsedEmailID:     target.ParsedEmailID,
		EmailID:           target.EmailID,
		ExternalMessageID: target.ExternalMessageID,
		Stage:             stage,
		Code:              code,
	}
}
