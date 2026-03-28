package domain

import (
	"errors"
	"testing"
)

func TestVendorValidate(t *testing.T) {
	t.Parallel()

	if err := (Vendor{UserID: 1, Name: "AWS"}).Validate(); err != nil {
		t.Fatalf("expected vendor to be valid, got %v", err)
	}

	if err := (Vendor{Name: "AWS"}).Validate(); !errors.Is(err, ErrVendorUserIDEmpty) {
		t.Fatalf("expected ErrVendorUserIDEmpty, got %v", err)
	}

	if err := (Vendor{UserID: 1, Name: "  "}).Validate(); !errors.Is(err, ErrVendorNameEmpty) {
		t.Fatalf("expected ErrVendorNameEmpty, got %v", err)
	}
}
