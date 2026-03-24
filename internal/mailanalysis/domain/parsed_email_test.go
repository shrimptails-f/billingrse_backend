package domain

import (
	commondomain "business/internal/common/domain"
	"strings"
	"testing"
	"time"
)

func TestAnalysisOutputNormalize_NormalizesAndFiltersEmptyParsedEmails(t *testing.T) {
	t.Parallel()

	output := AnalysisOutput{
		ParsedEmails: []commondomain.ParsedEmail{
			{
				ProductNameRaw:     stringPtr(" Example Product Full Name "),
				ProductNameDisplay: stringPtr(" Example Product "),
				VendorName:         stringPtr(" Example Vendor "),
				BillingNumber:      stringPtr(" INV-001 "),
				InvoiceNumber:      stringPtr(" inv-001 "),
				Currency:           stringPtr(" jpy "),
				PaymentCycle:       stringPtr(" one time "),
			},
			{
				ProductNameDisplay: stringPtr(" "),
			},
		},
		PromptVersion: " emailanalysis_v1 ",
	}.Normalize()

	if output.PromptVersion != "emailanalysis_v1" {
		t.Fatalf("unexpected metadata: %+v", output)
	}
	if len(output.ParsedEmails) != 1 {
		t.Fatalf("unexpected parsed emails: %+v", output.ParsedEmails)
	}

	parsed := output.ParsedEmails[0]
	if got := *parsed.ProductNameRaw; got != "Example Product Full Name" {
		t.Fatalf("unexpected raw product name: %q", got)
	}
	if got := *parsed.ProductNameDisplay; got != "Example Product" {
		t.Fatalf("unexpected display product name: %q", got)
	}
	if got := *parsed.VendorName; got != "Example Vendor" {
		t.Fatalf("unexpected vendor name: %q", got)
	}
	if got := *parsed.BillingNumber; got != "INV-001" {
		t.Fatalf("unexpected billing number: %q", got)
	}
	if got := *parsed.InvoiceNumber; got != "inv-001" {
		t.Fatalf("unexpected invoice number: %q", got)
	}
	if got := *parsed.Currency; got != "JPY" {
		t.Fatalf("unexpected currency: %q", got)
	}
	if got := *parsed.PaymentCycle; got != "one_time" {
		t.Fatalf("unexpected payment cycle: %q", got)
	}
}

func TestSaveInputNormalize_NormalizesParsedEmailsAndExtractedAt(t *testing.T) {
	t.Parallel()

	extractedAt := time.Date(2026, 3, 24, 12, 0, 0, 0, time.FixedZone("JST", 9*60*60))
	input := SaveInput{
		AnalysisRunID: " run-1 ",
		ExtractedAt:   extractedAt,
		PromptVersion: " emailanalysis_v1 ",
		ParsedEmails: []commondomain.ParsedEmail{
			{ProductNameDisplay: stringPtr(" Example Product ")},
			{ProductNameDisplay: stringPtr(" ")},
		},
	}.Normalize()

	if input.AnalysisRunID != "run-1" || input.PromptVersion != "emailanalysis_v1" {
		t.Fatalf("unexpected normalized metadata: %+v", input)
	}
	if input.ExtractedAt.Location() != time.UTC {
		t.Fatalf("expected UTC extracted_at, got %+v", input.ExtractedAt)
	}
	if len(input.ParsedEmails) != 1 {
		t.Fatalf("unexpected parsed emails: %+v", input.ParsedEmails)
	}
	if got := *input.ParsedEmails[0].ProductNameDisplay; got != "Example Product" {
		t.Fatalf("unexpected display product name: %q", got)
	}
}

func TestSaveInputValidate_InvoiceNumberChecksLengthOnly(t *testing.T) {
	t.Parallel()

	input := SaveInput{
		UserID:        1,
		EmailID:       10,
		AnalysisRunID: "run-1",
		ExtractedAt:   time.Date(2026, 3, 24, 10, 0, 0, 0, time.UTC),
		PromptVersion: "emailanalysis_v1",
		ParsedEmails: []commondomain.ParsedEmail{
			{InvoiceNumber: stringPtr("inv-anything")},
		},
	}.Normalize()

	if err := input.Validate(); err != nil {
		t.Fatalf("expected invoice number length-only validation, got %v", err)
	}
}

func TestSaveInputValidate_RejectsOverflowingFields(t *testing.T) {
	t.Parallel()

	base := SaveInput{
		UserID:        1,
		EmailID:       10,
		AnalysisRunID: "run-1",
		ExtractedAt:   time.Date(2026, 3, 24, 10, 0, 0, 0, time.UTC),
		PromptVersion: "emailanalysis_v1",
	}

	testCases := []struct {
		name   string
		parsed commondomain.ParsedEmail
	}{
		{
			name: "product_name_display too long",
			parsed: commondomain.ParsedEmail{
				ProductNameDisplay: stringPtr(strings.Repeat("a", parsedEmailProductDisplayMaxBytes+1)),
			},
		},
		{
			name: "invoice_number too long",
			parsed: commondomain.ParsedEmail{
				InvoiceNumber: stringPtr(strings.Repeat("1", parsedEmailInvoiceNumberMaxBytes+1)),
			},
		},
		{
			name: "amount too large",
			parsed: commondomain.ParsedEmail{
				Amount: float64Ptr(parsedEmailAmountMaxAbs + 1),
			},
		},
		{
			name: "amount too many decimals",
			parsed: commondomain.ParsedEmail{
				Amount: float64Ptr(123.4567),
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			input := base
			input.ParsedEmails = []commondomain.ParsedEmail{tc.parsed}
			if err := input.Normalize().Validate(); err == nil {
				t.Fatal("expected validation error, got nil")
			}
		})
	}
}

func stringPtr(value string) *string {
	return &value
}

func float64Ptr(value float64) *float64 {
	return &value
}
