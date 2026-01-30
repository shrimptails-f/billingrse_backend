package domain

import (
	"errors"
	"strings"
)

var (
	// ErrInvoiceNumberEmpty is returned when the invoice number is empty.
	ErrInvoiceNumberEmpty = errors.New("invoice number is empty")
	// ErrInvoiceNumberInvalidFormat is returned when the invoice number has an invalid format.
	ErrInvoiceNumberInvalidFormat = errors.New("invoice number is invalid")
)

// InvoiceNumber represents an invoice number (qualified invoice issuer number, "T" + 13 digits).
type InvoiceNumber string

// NewInvoiceNumber creates an optional invoice number from raw input.
// It returns empty when the input is nil or blank.
func NewInvoiceNumber(value *string) (InvoiceNumber, error) {
	if value == nil {
		return "", nil
	}
	normalized := NormalizeInvoiceNumber(*value)
	if normalized == "" {
		return "", nil
	}
	return InvoiceNumber(normalized), nil
}

// NormalizeInvoiceNumber trims and normalizes the invoice number.
// Validation is handled by InvoiceNumber.Validate.
func NormalizeInvoiceNumber(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

// String returns the raw string value.
func (n InvoiceNumber) String() string {
	return string(n)
}

// IsEmpty reports whether the invoice number is empty.
func (n InvoiceNumber) IsEmpty() bool {
	return strings.TrimSpace(string(n)) == ""
}

// Validate enforces invariants for InvoiceNumber.
func (n InvoiceNumber) Validate() error {
	normalized := NormalizeInvoiceNumber(string(n))
	if normalized == "" {
		return ErrInvoiceNumberEmpty
	}
	if !isValidInvoiceNumber(normalized) {
		return ErrInvoiceNumberInvalidFormat
	}
	return nil
}

func isValidInvoiceNumber(value string) bool {
	if len(value) != 14 {
		return false
	}
	if value[0] != 'T' {
		return false
	}
	for i := 1; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return true
}
