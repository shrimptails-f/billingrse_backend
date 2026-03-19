package domain

import "errors"

var (
	// ErrVendorResolutionUnresolved is returned when no canonical vendor has been resolved yet.
	ErrVendorResolutionUnresolved = errors.New("vendor resolution is unresolved")
)

// VendorResolution represents the outcome of vendor normalization.
// ParsedEmail may contain a candidate vendor name, but BillingEligibility
// must consume a resolved canonical Vendor through this model.
type VendorResolution struct {
	ResolvedVendor *Vendor
}

// IsResolved reports whether a canonical vendor has been resolved.
func (r VendorResolution) IsResolved() bool {
	return r.ResolvedVendor != nil
}

// Validate enforces invariants for a resolved vendor outcome.
func (r VendorResolution) Validate() error {
	if !r.IsResolved() {
		return ErrVendorResolutionUnresolved
	}
	return r.ResolvedVendor.Validate()
}
