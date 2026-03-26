package infrastructure

import (
	"business/internal/library/logger"
	"business/internal/library/timewrapper"
	"gorm.io/gorm"
	"time"
)

type billingRecord struct {
	ID                 uint       `gorm:"column:id;primaryKey;autoIncrement;index:idx_billings_user_summary_date_id,priority:3"`
	UserID             uint       `gorm:"column:user_id;not null;uniqueIndex:uni_billings_user_vendor_number,priority:1;index:idx_billings_user_summary_date_id,priority:1;index:idx_billings_user_email_id,priority:1"`
	VendorID           uint       `gorm:"column:vendor_id;not null;uniqueIndex:uni_billings_user_vendor_number,priority:2"`
	EmailID            uint       `gorm:"column:email_id;not null;index:idx_billings_user_email_id,priority:2"`
	ProductNameDisplay *string    `gorm:"column:product_name_display;size:255"`
	BillingNumber      string     `gorm:"column:billing_number;size:255;not null;uniqueIndex:uni_billings_user_vendor_number,priority:3"`
	InvoiceNumber      *string    `gorm:"column:invoice_number;size:14"`
	BillingDate        *time.Time `gorm:"column:billing_date"`
	BillingSummaryDate time.Time  `gorm:"column:billing_summary_date;not null;index:idx_billings_user_summary_date_id,priority:2"`
	PaymentCycle       string     `gorm:"column:payment_cycle;size:32;not null"`
	CreatedAt          time.Time  `gorm:"column:created_at;not null"`
	UpdatedAt          time.Time  `gorm:"column:updated_at;not null"`
}

func (billingRecord) TableName() string {
	return "billings"
}

type billingLineItemRecord struct {
	ID                 uint      `gorm:"column:id;primaryKey;autoIncrement"`
	BillingID          uint      `gorm:"column:billing_id;not null;index:idx_billing_line_items_user_billing,priority:2;uniqueIndex:uni_billing_line_items_billing_position,priority:1"`
	UserID             uint      `gorm:"column:user_id;not null;index:idx_billing_line_items_user_billing,priority:1"`
	Position           int       `gorm:"column:position;not null;uniqueIndex:uni_billing_line_items_billing_position,priority:2"`
	ProductNameRaw     *string   `gorm:"column:product_name_raw;type:text"`
	ProductNameDisplay *string   `gorm:"column:product_name_display;size:255"`
	Amount             *float64  `gorm:"column:amount;type:decimal(18,3)"`
	Currency           *string   `gorm:"column:currency;type:char(3)"`
	CreatedAt          time.Time `gorm:"column:created_at;not null"`
	UpdatedAt          time.Time `gorm:"column:updated_at;not null"`
}

func (billingLineItemRecord) TableName() string {
	return "billing_line_items"
}

// BillingQueryRepository loads billing read models from MySQL.
type BillingQueryRepository struct {
	db    *gorm.DB
	clock timewrapper.ClockInterface
	log   logger.Interface
}

// NewBillingQueryRepository creates a billing query repository backed by MySQL.
func NewBillingQueryRepository(
	db *gorm.DB,
	clock timewrapper.ClockInterface,
	log logger.Interface,
) *BillingQueryRepository {
	if clock == nil {
		clock = timewrapper.NewClock()
	}
	if log == nil {
		log = logger.NewNop()
	}

	return &BillingQueryRepository{
		db:    db,
		clock: clock,
		log:   log.With(logger.Component("billing_query_repository")),
	}
}

func cloneBillingDate(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := value.UTC()
	return &cloned
}

func cloneOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
