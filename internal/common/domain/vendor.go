package domain

import (
	"errors"
	"strings"
)

var (
	// ErrVendorNameEmpty is returned when the vendor name is empty.
	ErrVendorNameEmpty = errors.New("vendor name is empty")
	// ErrVendorUserIDEmpty is returned when the vendor owner is missing.
	ErrVendorUserIDEmpty = errors.New("vendor user_id is empty")
)

// Vendor represents a canonical, normalized billing vendor/service.
type Vendor struct {
	ID     uint
	UserID uint
	Name   string
}

// Validate enforces basic invariants for Vendor.
func (v Vendor) Validate() error {
	if v.UserID == 0 {
		return ErrVendorUserIDEmpty
	}
	if strings.TrimSpace(v.Name) == "" {
		return ErrVendorNameEmpty
	}
	return nil
}
