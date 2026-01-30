package domain

import (
	"errors"
	"testing"
)

func TestNewInvoiceNumber(t *testing.T) {
	t.Parallel()

	raw := " inv-001 "
	number, err := NewInvoiceNumber(&raw)
	if err != nil {
		t.Fatalf("NewInvoiceNumber returned error: %v", err)
	}
	if number.String() != "INV-001" {
		t.Fatalf("expected INV-001, got %s", number.String())
	}
	if err := number.Validate(); !errors.Is(err, ErrInvoiceNumberInvalidFormat) {
		t.Fatalf("expected ErrInvoiceNumberInvalidFormat, got %v", err)
	}

	raw = "t1234567890123"
	number, err = NewInvoiceNumber(&raw)
	if err != nil {
		t.Fatalf("NewInvoiceNumber returned error: %v", err)
	}
	if number.String() != "T1234567890123" {
		t.Fatalf("expected T1234567890123, got %s", number.String())
	}
	if err := number.Validate(); err != nil {
		t.Fatalf("expected valid invoice number, got %v", err)
	}

	empty := "  "
	number, err = NewInvoiceNumber(&empty)
	if err != nil {
		t.Fatalf("NewInvoiceNumber returned error: %v", err)
	}
	if !number.IsEmpty() {
		t.Fatalf("expected empty invoice number")
	}

	number, err = NewInvoiceNumber(nil)
	if err != nil {
		t.Fatalf("NewInvoiceNumber returned error: %v", err)
	}
	if !number.IsEmpty() {
		t.Fatalf("expected empty invoice number for nil input")
	}
}

func TestInvoiceNumberValidate(t *testing.T) {
	t.Parallel()

	if err := (InvoiceNumber("T1234567890123")).Validate(); err != nil {
		t.Fatalf("expected invoice number to be valid, got %v", err)
	}

	if err := (InvoiceNumber("INV-002")).Validate(); !errors.Is(err, ErrInvoiceNumberInvalidFormat) {
		t.Fatalf("expected ErrInvoiceNumberInvalidFormat, got %v", err)
	}

	if err := (InvoiceNumber("  ")).Validate(); !errors.Is(err, ErrInvoiceNumberEmpty) {
		t.Fatalf("expected ErrInvoiceNumberEmpty, got %v", err)
	}
}
