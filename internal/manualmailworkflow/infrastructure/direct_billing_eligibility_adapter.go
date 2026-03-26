package infrastructure

import (
	beapp "business/internal/billingeligibility/application"
	bedomain "business/internal/billingeligibility/domain"
	manualapp "business/internal/manualmailworkflow/application"
	"context"
	"errors"
	"time"
)

// DirectBillingEligibilityAdapter directly calls the billingeligibility usecase.
type DirectBillingEligibilityAdapter struct {
	usecase beapp.UseCase
}

// NewDirectBillingEligibilityAdapter creates a direct billingeligibility adapter.
func NewDirectBillingEligibilityAdapter(usecase beapp.UseCase) *DirectBillingEligibilityAdapter {
	return &DirectBillingEligibilityAdapter{usecase: usecase}
}

// Execute runs the billingeligibility stage and converts the result to workflow-owned types.
func (a *DirectBillingEligibilityAdapter) Execute(ctx context.Context, cmd manualapp.BillingEligibilityCommand) (manualapp.BillingEligibilityResult, error) {
	if a.usecase == nil {
		return manualapp.BillingEligibilityResult{}, errors.New("billingeligibility usecase is not configured")
	}

	targets := make([]beapp.EligibilityTarget, 0, len(cmd.ResolvedItems))
	for _, item := range cmd.ResolvedItems {
		targets = append(targets, beapp.EligibilityTarget{
			ParsedEmailID:     item.ParsedEmailID,
			EmailID:           item.EmailID,
			ExternalMessageID: item.ExternalMessageID,
			VendorID:          item.VendorID,
			VendorName:        item.VendorName,
			MatchedBy:         item.MatchedBy,
			Data:              item.Data,
		})
	}

	result, err := a.usecase.Execute(ctx, beapp.Command{
		UserID:        cmd.UserID,
		ResolvedItems: targets,
	})
	if err != nil {
		return manualapp.BillingEligibilityResult{}, err
	}

	eligibleItems := make([]manualapp.EligibleItem, 0, len(result.EligibleItems))
	for _, item := range result.EligibleItems {
		eligibleItems = append(eligibleItems, manualapp.EligibleItem{
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
			LineItems:          toEligibleLineItems(item.LineItems),
		})
	}

	ineligibleItems := make([]manualapp.IneligibleItem, 0, len(result.IneligibleItems))
	for _, item := range result.IneligibleItems {
		ineligibleItems = append(ineligibleItems, manualapp.IneligibleItem{
			ParsedEmailID:     item.ParsedEmailID,
			EmailID:           item.EmailID,
			ExternalMessageID: item.ExternalMessageID,
			VendorID:          item.VendorID,
			VendorName:        item.VendorName,
			MatchedBy:         item.MatchedBy,
			ReasonCode:        item.ReasonCode,
			Message:           item.Message,
		})
	}

	failures := make([]manualapp.BillingEligibilityFailure, 0, len(result.Failures))
	for _, failure := range result.Failures {
		failures = append(failures, manualapp.BillingEligibilityFailure{
			ParsedEmailID:     failure.ParsedEmailID,
			EmailID:           failure.EmailID,
			ExternalMessageID: failure.ExternalMessageID,
			Code:              failure.Code,
			Message:           failure.Message,
		})
	}

	return manualapp.BillingEligibilityResult{
		EligibleItems:   eligibleItems,
		EligibleCount:   result.EligibleCount,
		IneligibleItems: ineligibleItems,
		IneligibleCount: result.IneligibleCount,
		Failures:        failures,
	}, nil
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneFloat64(value *float64) *float64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func toEligibleLineItems(items []bedomain.LineItem) []manualapp.EligibleLineItem {
	if len(items) == 0 {
		return nil
	}

	lineItems := make([]manualapp.EligibleLineItem, 0, len(items))
	for _, item := range items {
		lineItems = append(lineItems, manualapp.EligibleLineItem{
			ProductNameRaw:     cloneString(item.ProductNameRaw),
			ProductNameDisplay: cloneString(item.ProductNameDisplay),
			Amount:             cloneFloat64(item.Amount),
			Currency:           cloneString(item.Currency),
		})
	}
	return lineItems
}
