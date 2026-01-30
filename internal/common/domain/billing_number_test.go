package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBillingNumber(t *testing.T) {
	t.Parallel()

	raw := "  INV-001  "
	number, err := NewBillingNumber(raw)
	if err != nil {
		t.Fatalf("NewBillingNumber returned error: %v", err)
	}
	if number.String() != "INV-001" {
		t.Fatalf("expected INV-001, got %s", number.String())
	}

	empty := "  "
	number, err = NewBillingNumber(empty)
	assert.ErrorIs(t, err, ErrBillingNumberEmpty)
}

func TestBillingNumberValidate(t *testing.T) {
	t.Parallel()

	if err := (BillingNumber("INV-002")).Validate(); err != nil {
		t.Fatalf("expected billing number to be valid, got %v", err)
	}
	if err := (BillingNumber("  ")).Validate(); !errors.Is(err, ErrBillingNumberEmpty) {
		t.Fatalf("expected ErrBillingNumberEmpty, got %v", err)
	}
}
