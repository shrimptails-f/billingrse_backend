package application

import (
	commondomain "business/internal/common/domain"
	"business/internal/library/logger"
	"business/internal/vendorresolution/domain"
	"context"
)

// ensureVendorByCandidateName は unresolved のときだけ policy の登録計画に従って master を補完する。
func (uc *useCase) ensureVendorByCandidateName(
	ctx context.Context,
	userID uint,
	target ResolutionTarget,
	input commondomain.VendorResolutionInput,
	decision domain.ResolutionDecision,
	reqLog logger.Interface,
) (domain.ResolutionDecision, error) {
	plan := uc.policy.BuildRegistrationPlan(input, decision)
	if plan == nil {
		return decision, nil
	}

	vendor, err := uc.registrationRepository.EnsureByPlan(ctx, *plan)
	if err != nil {
		return domain.ResolutionDecision{}, err
	}
	if vendor == nil {
		return decision, nil
	}

	// 自動登録して解決できたケースは監査できるように専用ログを出す。
	reqLog.Info("vendor_resolution_auto_registered",
		logger.UserID(userID),
		logger.Uint("parsed_email_id", target.ParsedEmailID),
		logger.Uint("email_id", target.EmailID),
		logger.String("external_message_id", target.ExternalMessageID),
		logger.String("candidate_vendor_name", plan.VendorName),
		logger.Uint("vendor_id", vendor.ID),
		logger.String("vendor_name", vendor.Name),
	)

	return uc.policy.ResolveRegisteredVendor(*vendor), nil
}
