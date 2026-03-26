package model

import (
	"time"
)

// Billing represents the billings table for persisted billing aggregates.
type Billing struct {
	ID                 uint    `gorm:"primaryKey;autoIncrement;index:idx_billings_user_summary_date_id,priority:3"`
	UserID             uint    `gorm:"not null;uniqueIndex:uni_billings_user_vendor_number,priority:1;index:idx_billings_user_summary_date_id,priority:1;index:idx_billings_user_email_id,priority:1"`
	VendorID           uint    `gorm:"not null;uniqueIndex:uni_billings_user_vendor_number,priority:2"`
	EmailID            uint    `gorm:"not null;index:idx_billings_user_email_id,priority:2"`
	ProductNameDisplay *string `gorm:"column:product_name_display;size:255"`
	BillingNumber      string  `gorm:"size:255;not null;uniqueIndex:uni_billings_user_vendor_number,priority:3"`
	InvoiceNumber      *string `gorm:"size:14"`
	BillingDate        *time.Time
	BillingSummaryDate time.Time `gorm:"column:billing_summary_date;not null;index:idx_billings_user_summary_date_id,priority:2"`
	PaymentCycle       string    `gorm:"size:32;not null"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// TableName specifies the table name for the Billing model.
func (Billing) TableName() string {
	return "billings"
}
