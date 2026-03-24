package domain

import (
	commondomain "business/internal/common/domain"
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	parsedEmailTextMaxBytes                   = 65535
	parsedEmailProductDisplayMaxBytes         = 255
	parsedEmailBillingNumberMaxBytes          = 255
	parsedEmailInvoiceNumberMaxBytes          = 14
	parsedEmailCurrencyMaxBytes               = 3
	parsedEmailPaymentCycleMaxBytes           = 32
	parsedEmailAmountScale                    = 3
	parsedEmailAmountMaxAbs           float64 = 999999999999999.999
)

// AnalysisOutput is the analyzer result returned to the application layer.
type AnalysisOutput struct {
	ParsedEmails  []commondomain.ParsedEmail
	PromptVersion string
}

// Normalize trims prompt metadata and normalizes all drafts.
func (o AnalysisOutput) Normalize() AnalysisOutput {
	o.PromptVersion = strings.TrimSpace(o.PromptVersion)

	normalizedParsedEmails := make([]commondomain.ParsedEmail, 0, len(o.ParsedEmails))
	for _, parsedEmail := range o.ParsedEmails {
		parsedEmail = parsedEmail.Normalize()
		if parsedEmail.IsEmpty() {
			continue
		}
		normalizedParsedEmails = append(normalizedParsedEmails, parsedEmail)
	}
	o.ParsedEmails = normalizedParsedEmails

	return o
}

// SaveInput is the repository input for appending parsed-email history.
type SaveInput struct {
	UserID        uint
	EmailID       uint
	AnalysisRunID string
	PositionBase  int
	ExtractedAt   time.Time
	PromptVersion string
	ParsedEmails  []commondomain.ParsedEmail
}

// Normalize trims metadata and normalizes all drafts.
func (in SaveInput) Normalize() SaveInput {
	in.AnalysisRunID = strings.TrimSpace(in.AnalysisRunID)
	in.PromptVersion = strings.TrimSpace(in.PromptVersion)
	if !in.ExtractedAt.IsZero() {
		in.ExtractedAt = in.ExtractedAt.UTC()
	}

	normalizedParsedEmails := make([]commondomain.ParsedEmail, 0, len(in.ParsedEmails))
	for _, parsedEmail := range in.ParsedEmails {
		parsedEmail = parsedEmail.Normalize()
		if parsedEmail.IsEmpty() {
			continue
		}
		normalizedParsedEmails = append(normalizedParsedEmails, parsedEmail)
	}
	in.ParsedEmails = normalizedParsedEmails

	return in
}

// Validate enforces repository-level save requirements.
func (in SaveInput) Validate() error {
	if in.UserID == 0 {
		return fmt.Errorf("user_id is required")
	}
	if in.EmailID == 0 {
		return fmt.Errorf("email_id is required")
	}
	if strings.TrimSpace(in.AnalysisRunID) == "" {
		return fmt.Errorf("analysis_run_id is required")
	}
	if in.PositionBase < 0 {
		return fmt.Errorf("position_base must be greater than or equal to zero")
	}
	if in.ExtractedAt.IsZero() {
		return fmt.Errorf("extracted_at is required")
	}
	if strings.TrimSpace(in.PromptVersion) == "" {
		return fmt.Errorf("prompt_version is required")
	}
	for idx, parsedEmail := range in.ParsedEmails {
		if err := validateParsedEmailBounds(parsedEmail); err != nil {
			return fmt.Errorf("parsed_emails[%d]: %w", idx, err)
		}
	}

	return nil
}

// ParsedEmailRecord identifies a stored ParsedEmail.
type ParsedEmailRecord struct {
	ID      uint
	EmailID uint
}

func validateParsedEmailBounds(parsed commondomain.ParsedEmail) error {
	if err := validateOptionalStringBytes("product_name_raw", parsed.ProductNameRaw, parsedEmailTextMaxBytes); err != nil {
		return err
	}
	if err := validateOptionalStringBytes("product_name_display", parsed.ProductNameDisplay, parsedEmailProductDisplayMaxBytes); err != nil {
		return err
	}
	if err := validateOptionalStringBytes("vendor_name", parsed.VendorName, parsedEmailTextMaxBytes); err != nil {
		return err
	}
	if err := validateOptionalStringBytes("billing_number", parsed.BillingNumber, parsedEmailBillingNumberMaxBytes); err != nil {
		return err
	}
	if err := validateOptionalStringBytes("invoice_number", parsed.InvoiceNumber, parsedEmailInvoiceNumberMaxBytes); err != nil {
		return err
	}
	if err := validateOptionalStringBytes("currency", parsed.Currency, parsedEmailCurrencyMaxBytes); err != nil {
		return err
	}
	if err := validateOptionalStringBytes("payment_cycle", parsed.PaymentCycle, parsedEmailPaymentCycleMaxBytes); err != nil {
		return err
	}
	if err := validateOptionalAmount(parsed.Amount); err != nil {
		return err
	}

	return nil
}

func validateOptionalStringBytes(fieldName string, value *string, maxBytes int) error {
	if value == nil {
		return nil
	}
	if len(*value) > maxBytes {
		return fmt.Errorf("%s exceeds max length %d bytes", fieldName, maxBytes)
	}
	return nil
}

func validateOptionalAmount(value *float64) error {
	if value == nil {
		return nil
	}
	if math.IsNaN(*value) || math.IsInf(*value, 0) {
		return fmt.Errorf("amount must be finite")
	}
	if math.Abs(*value) > parsedEmailAmountMaxAbs {
		return fmt.Errorf("amount exceeds max absolute value %.3f", parsedEmailAmountMaxAbs)
	}

	scaled := *value * 1000
	if math.Abs(scaled-math.Round(scaled)) > 1e-9 {
		return fmt.Errorf("amount must have at most %d decimal places", parsedEmailAmountScale)
	}

	return nil
}
