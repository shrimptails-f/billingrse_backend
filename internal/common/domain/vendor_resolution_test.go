package domain

import (
	"errors"
	"testing"
)

func TestVendorResolutionValidate(t *testing.T) {
	t.Parallel()

	t.Run("resolved vendor is valid", func(t *testing.T) {
		t.Parallel()

		resolution := VendorResolution{
			ResolvedVendor: &Vendor{Name: "AWS"},
		}

		if !resolution.IsResolved() {
			t.Fatalf("expected resolution to be resolved")
		}
		if err := resolution.Validate(); err != nil {
			t.Fatalf("expected resolution to be valid, got %v", err)
		}
	})

	t.Run("unresolved vendor returns error", func(t *testing.T) {
		t.Parallel()

		var resolution VendorResolution
		if resolution.IsResolved() {
			t.Fatalf("expected resolution to be unresolved")
		}
		if err := resolution.Validate(); !errors.Is(err, ErrVendorResolutionUnresolved) {
			t.Fatalf("expected ErrVendorResolutionUnresolved, got %v", err)
		}
	})

	t.Run("invalid resolved vendor returns vendor error", func(t *testing.T) {
		t.Parallel()

		resolution := VendorResolution{
			ResolvedVendor: &Vendor{Name: "  "},
		}

		if err := resolution.Validate(); !errors.Is(err, ErrVendorNameEmpty) {
			t.Fatalf("expected ErrVendorNameEmpty, got %v", err)
		}
	})
}
