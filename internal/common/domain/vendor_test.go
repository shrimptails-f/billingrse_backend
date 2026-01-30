package domain

import (
	"errors"
	"testing"
)

func TestVendorValidate(t *testing.T) {
	t.Parallel()

	if err := (Vendor{Name: "AWS"}).Validate(); err != nil {
		t.Fatalf("expected vendor to be valid, got %v", err)
	}

	if err := (Vendor{Name: "  "}).Validate(); !errors.Is(err, ErrVendorNameEmpty) {
		t.Fatalf("expected ErrVendorNameEmpty, got %v", err)
	}
}
