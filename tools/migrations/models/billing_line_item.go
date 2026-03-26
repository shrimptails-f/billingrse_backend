package model

import "time"

// BillingLineItem represents billing_line_items table for billing detail rows.
type BillingLineItem struct {
	ID                 uint     `gorm:"primaryKey;autoIncrement"`
	BillingID          uint     `gorm:"not null;uniqueIndex:uni_billing_line_items_billing_position,priority:1;index:idx_billing_line_items_user_billing,priority:2"`
	UserID             uint     `gorm:"not null;index:idx_billing_line_items_user_billing,priority:1"`
	Position           int      `gorm:"not null;uniqueIndex:uni_billing_line_items_billing_position,priority:2"`
	ProductNameRaw     *string  `gorm:"column:product_name_raw;type:text"`
	ProductNameDisplay *string  `gorm:"column:product_name_display;size:255"`
	Amount             *float64 `gorm:"type:decimal(18,3)"`
	Currency           *string  `gorm:"type:char(3)"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// TableName specifies the table name for the BillingLineItem model.
func (BillingLineItem) TableName() string {
	return "billing_line_items"
}
