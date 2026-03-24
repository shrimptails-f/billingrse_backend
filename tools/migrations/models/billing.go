package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// Billing represents the billings table for persisted billing aggregates.
type Billing struct {
	ID            uint            `gorm:"primaryKey;autoIncrement"`
	UserID        uint            `gorm:"not null;uniqueIndex:uni_billings_user_vendor_number,priority:1"`
	VendorID      uint            `gorm:"not null;uniqueIndex:uni_billings_user_vendor_number,priority:2"`
	EmailID       uint            `gorm:"not null"`
	BillingNumber string          `gorm:"size:255;not null;uniqueIndex:uni_billings_user_vendor_number,priority:3"`
	InvoiceNumber *string         `gorm:"size:14"`
	Amount        decimal.Decimal `gorm:"type:decimal(18,3);not null"`
	Currency      string          `gorm:"type:char(3);not null"`
	BillingDate   *time.Time
	PaymentCycle  string `gorm:"size:32;not null"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// TableName specifies the table name for the Billing model.
func (Billing) TableName() string {
	return "billings"
}
