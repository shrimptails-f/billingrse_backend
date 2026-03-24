package model

import "time"

// ParsedEmail represents the parsed_emails table for mailanalysis history.
type ParsedEmail struct {
	ID                 uint     `gorm:"primaryKey;autoIncrement"`
	UserID             uint     `gorm:"not null;index:idx_parsed_emails_user_email,priority:1"`
	EmailID            uint     `gorm:"not null;index:idx_parsed_emails_user_email,priority:2"`
	AnalysisRunID      string   `gorm:"type:char(36);not null;index:idx_parsed_emails_analysis_run;uniqueIndex:uni_parsed_emails_run_position,priority:1"`
	Position           int      `gorm:"not null;uniqueIndex:uni_parsed_emails_run_position,priority:2"`
	ProductNameRaw     *string  `gorm:"column:product_name_raw;type:text"`
	ProductNameDisplay *string  `gorm:"column:product_name_display;size:255"`
	VendorName         *string  `gorm:"type:text"`
	BillingNumber      *string  `gorm:"size:255"`
	InvoiceNumber      *string  `gorm:"size:14"`
	Amount             *float64 `gorm:"type:decimal(18,3)"`
	Currency           *string  `gorm:"type:char(3)"`
	BillingDate        *time.Time
	PaymentCycle       *string   `gorm:"size:32"`
	ExtractedAt        time.Time `gorm:"not null"`
	PromptVersion      string    `gorm:"size:50;not null"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// TableName specifies the table name for the ParsedEmail model.
func (ParsedEmail) TableName() string {
	return "parsed_emails"
}
