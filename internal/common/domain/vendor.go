package domain

import (
	"errors"
	"strings"
)

var (
	// ErrVendorNameEmpty is returned when the vendor name is empty.
	ErrVendorNameEmpty = errors.New("vendor name is empty")
)

// Vendor represents a normalized billing vendor/service.
type Vendor struct {
	ID   uint
	Name string
}

// Validate enforces basic invariants for Vendor.
func (v Vendor) Validate() error {
	if strings.TrimSpace(v.Name) == "" {
		return ErrVendorNameEmpty
	}
	return nil
}
