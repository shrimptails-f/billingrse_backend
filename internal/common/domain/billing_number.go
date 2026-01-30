package domain

import (
	"errors"
	"strings"
)

var (
	// ErrBillingNumberEmpty is returned when the billing number is empty.
	ErrBillingNumberEmpty = errors.New("billing number is empty")
)

// BillingNumber represents a vendor-provided invoice/billing identifier.
type BillingNumber string

// NewBillingNumber creates an optional billing number from raw input.
// It returns empty when the input is nil or blank.
func NewBillingNumber(value *string) (BillingNumber, error) {
	if value == nil {
		return "", ErrBillingNumberEmpty
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return "", ErrBillingNumberEmpty
	}
	return BillingNumber(trimmed), nil
}

// String returns the raw string value.
func (n BillingNumber) String() string {
	return string(n)
}

// IsEmpty reports whether the billing number is empty.
func (n BillingNumber) IsEmpty() bool {
	return strings.TrimSpace(string(n)) == ""
}

// Validate enforces invariants for BillingNumber.
func (n BillingNumber) Validate() error {
	if n.IsEmpty() {
		return ErrBillingNumberEmpty
	}
	return nil
}
