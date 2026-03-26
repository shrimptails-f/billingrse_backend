package application

import (
	"business/internal/billingeligibility/domain"
	commondomain "business/internal/common/domain"
	"business/internal/library/logger"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestUseCaseExecute(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)
	amount := 1200.0
	currency := " jpy "
	productNameRaw := " Example Product Full Name "
	billingNumber := " INV-001 "
	invoiceNumber := " T1234567890123 "
	paymentCycle := " one time "
	productNameDisplay := " Example Product "
	validData := commondomain.ParsedEmail{
		ProductNameDisplay: &productNameDisplay,
		ProductNameRaw:     &productNameRaw,
		BillingNumber:      &billingNumber,
		InvoiceNumber:      &invoiceNumber,
		Amount:             &amount,
		Currency:           &currency,
		BillingDate:        &now,
		PaymentCycle:       &paymentCycle,
		LineItems: []commondomain.ParsedEmailLineItem{
			{
				ProductNameDisplay: &productNameDisplay,
				Amount:             &amount,
				Currency:           &currency,
			},
		},
	}

	missingCurrencyData := validData
	missingCurrencyData.Currency = nil

	nilBillingDateData := validData
	nilBillingDateData.BillingDate = nil
	nilBillingDateData.ProductNameDisplay = nil

	uc := NewUseCase(logger.NewNop())

	result, err := uc.Execute(context.Background(), Command{
		UserID: 10,
		ResolvedItems: []EligibilityTarget{
			{
				ParsedEmailID:     9001,
				EmailID:           101,
				ExternalMessageID: "msg-1",
				VendorID:          3001,
				VendorName:        " Acme ",
				MatchedBy:         " name_exact ",
				Data:              validData,
			},
			{
				ParsedEmailID:     9002,
				EmailID:           102,
				ExternalMessageID: "msg-2",
				VendorID:          3001,
				VendorName:        "Acme",
				MatchedBy:         "name_exact",
				Data:              missingCurrencyData,
			},
			{
				ParsedEmailID:     0,
				EmailID:           103,
				ExternalMessageID: "msg-3",
				VendorID:          3002,
				VendorName:        "Broken",
				MatchedBy:         "name_exact",
				Data:              validData,
			},
			{
				ParsedEmailID:     9004,
				EmailID:           104,
				ExternalMessageID: "msg-4",
				VendorID:          3003,
				VendorName:        "Acme",
				MatchedBy:         "name_exact",
				Data:              nilBillingDateData,
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if result.EligibleCount != 2 {
		t.Fatalf("expected 2 eligible items, got %+v", result)
	}
	if result.IneligibleCount != 1 {
		t.Fatalf("expected 1 ineligible item, got %+v", result)
	}
	if len(result.Failures) != 1 {
		t.Fatalf("expected 1 failure, got %+v", result)
	}

	first := result.EligibleItems[0]
	if first.VendorName != "Acme" || first.MatchedBy != "name_exact" {
		t.Fatalf("expected normalized vendor fields, got %+v", first)
	}
	if first.BillingNumber != "INV-001" {
		t.Fatalf("expected normalized billing number, got %+v", first)
	}
	if first.InvoiceNumber == nil || *first.InvoiceNumber != "T1234567890123" {
		t.Fatalf("expected invoice number to be preserved, got %+v", first)
	}
	if first.Currency != "JPY" || first.PaymentCycle != "one_time" {
		t.Fatalf("expected normalized currency/payment cycle, got %+v", first)
	}
	if first.BillingDate == nil || !first.BillingDate.Equal(now) {
		t.Fatalf("expected billing date to be preserved, got %+v", first)
	}
	if first.ProductNameDisplay == nil || *first.ProductNameDisplay != "Example Product" {
		t.Fatalf("expected product name display to be preserved, got %+v", first)
	}
	if len(first.LineItems) != 1 {
		t.Fatalf("expected line items to be preserved, got %+v", first)
	}
	if first.LineItems[0].ProductNameDisplay == nil || *first.LineItems[0].ProductNameDisplay != "Example Product" {
		t.Fatalf("expected line item display to be preserved, got %+v", first.LineItems[0])
	}

	second := result.EligibleItems[1]
	if second.BillingDate != nil {
		t.Fatalf("expected nil billing date to remain allowed, got %+v", second)
	}
	if second.ProductNameDisplay == nil || *second.ProductNameDisplay != "Example Product Full Name" {
		t.Fatalf("expected product name raw to be used as fallback, got %+v", second)
	}

	if result.IneligibleItems[0].ReasonCode != domain.ReasonCodeCurrencyEmpty {
		t.Fatalf("expected currency_empty reason, got %+v", result.IneligibleItems[0])
	}
	if result.IneligibleItems[0].Message == "" || result.IneligibleItems[0].Message == result.IneligibleItems[0].ReasonCode {
		t.Fatalf("expected human-readable ineligible message, got %+v", result.IneligibleItems[0])
	}
	if !strings.Contains(result.IneligibleItems[0].Message, "Acme") || !strings.Contains(result.IneligibleItems[0].Message, "msg-2") {
		t.Fatalf("expected vendor name and message id in ineligible message, got %+v", result.IneligibleItems[0])
	}

	if result.Failures[0].Code != domain.FailureCodeInvalidEligibilityTarget {
		t.Fatalf("unexpected failure: %+v", result.Failures[0])
	}
	if result.Failures[0].Message == "" {
		t.Fatalf("expected human-readable failure message, got %+v", result.Failures[0])
	}
	if !strings.Contains(result.Failures[0].Message, "Broken") || !strings.Contains(result.Failures[0].Message, "msg-3") {
		t.Fatalf("expected vendor name and message id in failure message, got %+v", result.Failures[0])
	}
}

func TestUseCaseExecute_InvalidCommand(t *testing.T) {
	t.Parallel()

	uc := NewUseCase(logger.NewNop())

	_, err := uc.Execute(context.Background(), Command{})
	if !errors.Is(err, domain.ErrInvalidCommand) {
		t.Fatalf("expected ErrInvalidCommand, got %v", err)
	}
}

func TestUseCaseExecute_ProductNameMissingBecomesIneligible(t *testing.T) {
	t.Parallel()

	amount := 1200.0
	currency := "JPY"
	billingNumber := "INV-001"
	paymentCycle := "one_time"

	uc := NewUseCase(logger.NewNop())
	result, err := uc.Execute(context.Background(), Command{
		UserID: 10,
		ResolvedItems: []EligibilityTarget{
			{
				ParsedEmailID:     9001,
				EmailID:           101,
				ExternalMessageID: "msg-1",
				VendorID:          3001,
				VendorName:        "Acme",
				MatchedBy:         "name_exact",
				Data: commondomain.ParsedEmail{
					BillingNumber: &billingNumber,
					Amount:        &amount,
					Currency:      &currency,
					PaymentCycle:  &paymentCycle,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if result.EligibleCount != 0 || result.IneligibleCount != 1 {
		t.Fatalf("expected one ineligible item, got %+v", result)
	}
	if result.IneligibleItems[0].ReasonCode != domain.ReasonCodeProductNameEmpty {
		t.Fatalf("expected product_name_empty, got %+v", result.IneligibleItems[0])
	}
	if result.IneligibleItems[0].Message == "" {
		t.Fatalf("expected human-readable ineligible message, got %+v", result.IneligibleItems[0])
	}
	if !strings.Contains(result.IneligibleItems[0].Message, "Acme") || !strings.Contains(result.IneligibleItems[0].Message, "msg-1") {
		t.Fatalf("expected vendor name and message id in ineligible message, got %+v", result.IneligibleItems[0])
	}
}

func TestUseCaseExecute_NilContext(t *testing.T) {
	t.Parallel()

	uc := NewUseCase(logger.NewNop())

	_, err := uc.Execute(nil, Command{UserID: 1})
	if !errors.Is(err, logger.ErrNilContext) {
		t.Fatalf("expected logger.ErrNilContext, got %v", err)
	}
}
