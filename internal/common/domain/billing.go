package domain

import (
	"errors"
	"time"
)

var (
	// ErrBillingUserIDEmpty is returned when the user ID is missing.
	ErrBillingUserIDEmpty = errors.New("billing user id is empty")
	// ErrBillingVendorIDEmpty is returned when the vendor ID is missing.
	ErrBillingVendorIDEmpty = errors.New("billing vendor id is empty")
	// ErrBillingEmailIDEmpty is returned when the source email ID is missing.
	ErrBillingEmailIDEmpty = errors.New("billing email id is empty")
	// ErrBillingDateInvalid is returned when a provided billing date is invalid.
	ErrBillingDateInvalid = errors.New("billing date is invalid")
	// ErrBillingLineItemsEmpty is returned when billing has no detail rows.
	ErrBillingLineItemsEmpty = errors.New("billing line items are empty")
)

// Billing represents the aggregate root for billing.
type Billing struct {
	ID                 uint
	UserID             uint
	VendorID           uint
	EmailID            uint
	ProductNameDisplay *string
	BillingNumber      BillingNumber // Vendor-provided invoice/billing identifier.
	InvoiceNumber      InvoiceNumber // Invoice number (qualified invoice issuer number, optional).
	Money              Money
	BillingDate        *time.Time // BillingDate is optional when the source mail does not include it.
	PaymentCycle       PaymentCycle
	LineItems          []BillingLineItem
}

// NewBilling constructs a Billing with normalized values.
func NewBilling(
	userID uint,
	vendorID uint,
	emailID uint,
	billingNumber string,
	invoiceNumber *string,
	amount float64,
	currency string,
	billingDate *time.Time,
	cycle string,
	productNameDisplay *string,
	lineItems []BillingLineItemInput,
) (Billing, error) {
	normalizedBillingNumber, err := NewBillingNumber(billingNumber)
	if err != nil {
		return Billing{}, err
	}

	normalizedInvoice, err := NewInvoiceNumber(invoiceNumber)
	if err != nil {
		return Billing{}, err
	}

	money, err := NewMoney(amount, currency)
	if err != nil {
		return Billing{}, err
	}

	normalizedCycle, err := NewPaymentCycle(cycle)
	if err != nil {
		return Billing{}, err
	}

	billing := Billing{
		UserID:             userID,
		VendorID:           vendorID,
		EmailID:            emailID,
		ProductNameDisplay: normalizeOptionalString(productNameDisplay),
		BillingNumber:      normalizedBillingNumber,
		InvoiceNumber:      normalizedInvoice,
		Money:              money,
		BillingDate:        cloneBillingDate(billingDate),
		PaymentCycle:       normalizedCycle,
		LineItems:          resolveBillingLineItems(lineItems, productNameDisplay, amount, currency),
	}

	if err := billing.Validate(); err != nil {
		return Billing{}, err
	}
	return billing, nil
}

// Validate enforces invariants for Billing.
func (b Billing) Validate() error {
	if b.UserID == 0 {
		return ErrBillingUserIDEmpty
	}
	if b.VendorID == 0 {
		return ErrBillingVendorIDEmpty
	}
	if b.EmailID == 0 {
		return ErrBillingEmailIDEmpty
	}
	if err := b.BillingNumber.Validate(); err != nil {
		return err
	}
	if !b.InvoiceNumber.IsEmpty() {
		if err := b.InvoiceNumber.Validate(); err != nil {
			return err
		}
	}
	if err := b.Money.Validate(); err != nil {
		return err
	}
	if b.BillingDate != nil && b.BillingDate.IsZero() {
		return ErrBillingDateInvalid
	}
	if err := b.PaymentCycle.Validate(); err != nil {
		return err
	}
	if len(normalizeBillingLineItems(b.LineItems)) == 0 {
		return ErrBillingLineItemsEmpty
	}
	return nil
}

func resolveBillingLineItems(items []BillingLineItemInput, productNameDisplay *string, amount float64, currency string) []BillingLineItem {
	normalizedInputs := normalizeBillingLineItemInputs(items)
	if len(normalizedInputs) > 0 {
		lineItems := make([]BillingLineItem, 0, len(normalizedInputs))
		for _, item := range normalizedInputs {
			lineItems = append(lineItems, item.ToBillingLineItem())
		}
		return lineItems
	}

	fallbackAmount := amount
	fallbackCurrency := currency
	fallback := BillingLineItemInput{
		ProductNameDisplay: normalizeOptionalString(productNameDisplay),
		Amount:             &fallbackAmount,
		Currency:           normalizeOptionalUpperString(&fallbackCurrency),
	}.Normalize()
	if fallback.IsEmpty() {
		return nil
	}

	return []BillingLineItem{fallback.ToBillingLineItem()}
}

func cloneBillingDate(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	cloned := value.UTC()
	return &cloned
}
