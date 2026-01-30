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
	// ErrBillingDateEmpty is returned when the billing date is missing.
	ErrBillingDateEmpty = errors.New("billing date is empty")
)

// Billing represents the aggregate root for billing.
type Billing struct {
	ID            uint
	UserID        uint
	VendorID      uint
	EmailID       uint
	BillingNumber BillingNumber // Vendor-provided invoice/billing identifier.
	InvoiceNumber InvoiceNumber // Invoice number (qualified invoice issuer number, optional).
	Money         Money
	BillingDate   time.Time // BillingDate is interpreted in JST.
	PaymentCycle  PaymentCycle
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
	billingDate time.Time,
	cycle string,
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
		UserID:        userID,
		VendorID:      vendorID,
		EmailID:       emailID,
		BillingNumber: normalizedBillingNumber,
		InvoiceNumber: normalizedInvoice,
		Money:         money,
		BillingDate:   billingDate,
		PaymentCycle:  normalizedCycle,
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
	if b.BillingDate.IsZero() {
		return ErrBillingDateEmpty
	}
	if err := b.PaymentCycle.Validate(); err != nil {
		return err
	}
	return nil
}
