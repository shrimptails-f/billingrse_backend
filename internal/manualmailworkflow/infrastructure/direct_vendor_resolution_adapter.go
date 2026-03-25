package infrastructure

import (
	manualapp "business/internal/manualmailworkflow/application"
	vrapp "business/internal/vendorresolution/application"
	"context"
	"errors"
)

// DirectVendorResolutionAdapter は vendorresolution usecase を直接呼び出す。
type DirectVendorResolutionAdapter struct {
	usecase vrapp.UseCase
}

// NewDirectVendorResolutionAdapter は direct な vendorresolution adapter を生成する。
func NewDirectVendorResolutionAdapter(usecase vrapp.UseCase) *DirectVendorResolutionAdapter {
	return &DirectVendorResolutionAdapter{usecase: usecase}
}

// Execute は vendorresolution stage を実行し、workflow 側の型へ変換する。
func (a *DirectVendorResolutionAdapter) Execute(ctx context.Context, cmd manualapp.VendorResolutionCommand) (manualapp.VendorResolutionResult, error) {
	if a.usecase == nil {
		return manualapp.VendorResolutionResult{}, errors.New("vendorresolution usecase is not configured")
	}

	parsedEmailMap := make(map[uint]manualapp.ParsedEmail, len(cmd.ParsedEmails))

	result, err := a.usecase.Execute(ctx, vrapp.Command{
		UserID: cmd.UserID,
		ParsedEmails: func() []vrapp.ResolutionTarget {
			targets := make([]vrapp.ResolutionTarget, 0, len(cmd.ParsedEmails))
			for _, parsedEmail := range cmd.ParsedEmails {
				parsedEmailMap[parsedEmail.ParsedEmailID] = parsedEmail
				targets = append(targets, vrapp.ResolutionTarget{
					ParsedEmailID:     parsedEmail.ParsedEmailID,
					EmailID:           parsedEmail.EmailID,
					ExternalMessageID: parsedEmail.ExternalMessageID,
					Subject:           parsedEmail.Subject,
					From:              parsedEmail.From,
					To:                append([]string{}, parsedEmail.To...),
					BodyDigest:        parsedEmail.BodyDigest,
					ParsedEmail:       parsedEmail.Data,
				})
			}
			return targets
		}(),
	})
	if err != nil {
		return manualapp.VendorResolutionResult{}, err
	}

	resolvedItems := make([]manualapp.ResolvedItem, 0, len(result.ResolvedItems))
	for _, item := range result.ResolvedItems {
		data := parsedEmailMap[item.ParsedEmailID]
		resolvedItems = append(resolvedItems, manualapp.ResolvedItem{
			ParsedEmailID:     item.ParsedEmailID,
			EmailID:           item.EmailID,
			ExternalMessageID: item.ExternalMessageID,
			VendorID:          item.VendorID,
			VendorName:        item.VendorName,
			MatchedBy:         item.MatchedBy,
			Data:              data.Data,
		})
	}

	unresolvedItems := make([]manualapp.UnresolvedItem, 0, len(result.UnresolvedItems))
	for _, item := range result.UnresolvedItems {
		unresolvedItems = append(unresolvedItems, manualapp.UnresolvedItem{
			ParsedEmailID:       item.ParsedEmailID,
			EmailID:             item.EmailID,
			ExternalMessageID:   item.ExternalMessageID,
			ReasonCode:          item.ReasonCode,
			Message:             item.Message,
			CandidateVendorName: item.CandidateVendorName,
		})
	}

	failures := make([]manualapp.VendorResolutionFailure, 0, len(result.Failures))
	for _, failure := range result.Failures {
		failures = append(failures, manualapp.VendorResolutionFailure{
			ParsedEmailID:     failure.ParsedEmailID,
			EmailID:           failure.EmailID,
			ExternalMessageID: failure.ExternalMessageID,
			Stage:             failure.Stage,
			Code:              failure.Code,
			Message:           failure.Message,
		})
	}

	return manualapp.VendorResolutionResult{
		ResolvedItems:   resolvedItems,
		ResolvedCount:   result.ResolvedCount,
		UnresolvedItems: unresolvedItems,
		UnresolvedCount: result.UnresolvedCount,
		Failures:        failures,
	}, nil
}
