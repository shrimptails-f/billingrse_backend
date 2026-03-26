package domain

import "time"

const (
	// FailureCodeInvalidEligibilityTarget indicates the workflow passed an invalid target.
	FailureCodeInvalidEligibilityTarget = "invalid_eligibility_target"
	// FailureCodeBillingEligibilityFail indicates the policy returned an unexpected failure.
	FailureCodeBillingEligibilityFail = "billing_eligibility_failed"

	// ReasonCodeAmountEmpty indicates amount is missing.
	ReasonCodeAmountEmpty = "amount_empty"
	// ReasonCodeProductNameEmpty indicates both display/raw product names are missing.
	ReasonCodeProductNameEmpty = "product_name_empty"
	// ReasonCodeAmountInvalid indicates amount is invalid.
	ReasonCodeAmountInvalid = "amount_invalid"
	// ReasonCodeCurrencyEmpty indicates currency is missing.
	ReasonCodeCurrencyEmpty = "currency_empty"
	// ReasonCodeCurrencyInvalid indicates currency is invalid.
	ReasonCodeCurrencyInvalid = "currency_invalid"
	// ReasonCodeBillingNumberEmpty indicates billing number is missing.
	ReasonCodeBillingNumberEmpty = "billing_number_empty"
	// ReasonCodePaymentCycleEmpty indicates payment cycle is missing.
	ReasonCodePaymentCycleEmpty = "payment_cycle_empty"
	// ReasonCodePaymentCycleInvalid indicates payment cycle is invalid.
	ReasonCodePaymentCycleInvalid = "payment_cycle_invalid"
)

// EligibleItem is a billing-ready item that can be passed to the next stage.
type EligibleItem struct {
	ParsedEmailID      uint
	EmailID            uint
	ExternalMessageID  string
	VendorID           uint
	VendorName         string
	MatchedBy          string
	ProductNameDisplay *string
	BillingNumber      string
	InvoiceNumber      *string
	Amount             float64
	Currency           string
	BillingDate        *time.Time
	PaymentCycle       string
	LineItems          []LineItem
}

// LineItem is one billing detail row extracted under the same billing number.
type LineItem struct {
	ProductNameRaw     *string
	ProductNameDisplay *string
	Amount             *float64
	Currency           *string
}

// IneligibleItem is a business-level non-eligible result with a stable reason code.
type IneligibleItem struct {
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	VendorID          uint
	VendorName        string
	MatchedBy         string
	ReasonCode        string
	Message           string
}

// Failure is a technical or contract failure for a single target.
type Failure struct {
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	Code              string
	Message           string
}
