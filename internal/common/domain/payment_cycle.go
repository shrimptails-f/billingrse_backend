package domain

import (
	"errors"
	"strings"
)

var (
	// ErrPaymentCycleEmpty is returned when the payment cycle is empty.
	ErrPaymentCycleEmpty = errors.New("payment cycle is empty")
	// ErrPaymentCycleInvalid is returned when the payment cycle is invalid.
	ErrPaymentCycleInvalid = errors.New("payment cycle is invalid")
)

// PaymentCycle represents whether a billing is one-time or recurring.
type PaymentCycle string

const (
	PaymentCycleOneTime   PaymentCycle = "one_time"
	PaymentCycleRecurring PaymentCycle = "recurring"
)

// String returns the raw string value.
func (c PaymentCycle) String() string {
	return string(c)
}

// NewPaymentCycle creates a normalized payment cycle from raw input.
func NewPaymentCycle(value string) (PaymentCycle, error) {
	normalized := NormalizePaymentCycle(value)
	cycle := PaymentCycle(normalized)
	if err := cycle.Validate(); err != nil {
		return "", err
	}
	return cycle, nil
}

// NormalizePaymentCycle trims and lowercases the input.
func NormalizePaymentCycle(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// IsZero reports whether the payment cycle is empty.
func (c PaymentCycle) IsZero() bool {
	return strings.TrimSpace(string(c)) == ""
}

// Validate enforces invariants for PaymentCycle.
func (c PaymentCycle) Validate() error {
	normalized := NormalizePaymentCycle(string(c))
	if normalized == "" {
		return ErrPaymentCycleEmpty
	}
	if normalized != string(PaymentCycleOneTime) && normalized != string(PaymentCycleRecurring) {
		return ErrPaymentCycleInvalid
	}
	return nil
}
