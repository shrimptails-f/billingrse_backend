package infrastructure

import (
	billingapp "business/internal/billing/application"
	manualapp "business/internal/manualmailworkflow/application"
	"context"
	"errors"
)

// DirectBillingAdapter directly calls the billing usecase.
type DirectBillingAdapter struct {
	usecase billingapp.UseCase
}

// NewDirectBillingAdapter creates a direct billing adapter.
func NewDirectBillingAdapter(usecase billingapp.UseCase) *DirectBillingAdapter {
	return &DirectBillingAdapter{usecase: usecase}
}

// Execute runs the billing stage and converts the result to workflow-owned types.
func (a *DirectBillingAdapter) Execute(ctx context.Context, cmd manualapp.BillingCommand) (manualapp.BillingResult, error) {
	if a.usecase == nil {
		return manualapp.BillingResult{}, errors.New("billing usecase is not configured")
	}

	targets := make([]billingapp.CreationTarget, 0, len(cmd.EligibleItems))
	for _, item := range cmd.EligibleItems {
		targets = append(targets, billingapp.CreationTarget{
			ParsedEmailID:      item.ParsedEmailID,
			EmailID:            item.EmailID,
			ExternalMessageID:  item.ExternalMessageID,
			VendorID:           item.VendorID,
			VendorName:         item.VendorName,
			MatchedBy:          item.MatchedBy,
			ProductNameDisplay: cloneString(item.ProductNameDisplay),
			BillingNumber:      item.BillingNumber,
			InvoiceNumber:      cloneString(item.InvoiceNumber),
			Amount:             item.Amount,
			Currency:           item.Currency,
			BillingDate:        cloneTime(item.BillingDate),
			PaymentCycle:       item.PaymentCycle,
		})
	}

	result, err := a.usecase.Execute(ctx, billingapp.Command{
		UserID:        cmd.UserID,
		EligibleItems: targets,
	})
	if err != nil {
		return manualapp.BillingResult{}, err
	}

	createdItems := make([]manualapp.BillingCreatedItem, 0, len(result.CreatedItems))
	for _, item := range result.CreatedItems {
		createdItems = append(createdItems, manualapp.BillingCreatedItem{
			BillingID:         item.BillingID,
			ParsedEmailID:     item.ParsedEmailID,
			EmailID:           item.EmailID,
			ExternalMessageID: item.ExternalMessageID,
			VendorID:          item.VendorID,
			VendorName:        item.VendorName,
			BillingNumber:     item.BillingNumber,
		})
	}

	duplicateItems := make([]manualapp.BillingDuplicateItem, 0, len(result.DuplicateItems))
	for _, item := range result.DuplicateItems {
		duplicateItems = append(duplicateItems, manualapp.BillingDuplicateItem{
			ExistingBillingID: item.ExistingBillingID,
			ParsedEmailID:     item.ParsedEmailID,
			EmailID:           item.EmailID,
			ExternalMessageID: item.ExternalMessageID,
			VendorID:          item.VendorID,
			VendorName:        item.VendorName,
			BillingNumber:     item.BillingNumber,
		})
	}

	failures := make([]manualapp.BillingFailure, 0, len(result.Failures))
	for _, failure := range result.Failures {
		failures = append(failures, manualapp.BillingFailure{
			ParsedEmailID:     failure.ParsedEmailID,
			EmailID:           failure.EmailID,
			ExternalMessageID: failure.ExternalMessageID,
			Stage:             failure.Stage,
			Code:              failure.Code,
		})
	}

	return manualapp.BillingResult{
		CreatedItems:   createdItems,
		CreatedCount:   result.CreatedCount,
		DuplicateItems: duplicateItems,
		DuplicateCount: result.DuplicateCount,
		Failures:       failures,
	}, nil
}
