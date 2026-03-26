package infrastructure

import (
	billingapp "business/internal/billing/application"
	commondomain "business/internal/common/domain"
	"business/internal/library/logger"
	"business/internal/library/timewrapper"
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
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
	ID                 uint             `gorm:"column:id;primaryKey;autoIncrement"`
	BillingID          uint             `gorm:"column:billing_id;not null;index:idx_billing_line_items_user_billing,priority:2;uniqueIndex:uni_billing_line_items_billing_position,priority:1"`
	UserID             uint             `gorm:"column:user_id;not null;index:idx_billing_line_items_user_billing,priority:1"`
	Position           int              `gorm:"column:position;not null;uniqueIndex:uni_billing_line_items_billing_position,priority:2"`
	ProductNameRaw     *string          `gorm:"column:product_name_raw;type:text"`
	ProductNameDisplay *string          `gorm:"column:product_name_display;size:255"`
	Amount             *decimal.Decimal `gorm:"column:amount;type:decimal(18,3)"`
	Currency           *string          `gorm:"column:currency;type:char(3)"`
	CreatedAt          time.Time        `gorm:"column:created_at;not null"`
	UpdatedAt          time.Time        `gorm:"column:updated_at;not null"`
}

func (billingLineItemRecord) TableName() string {
	return "billing_line_items"
}

type billingSourceEmailRecord struct {
	ID         uint      `gorm:"column:id;primaryKey;autoIncrement"`
	UserID     uint      `gorm:"column:user_id;not null"`
	ReceivedAt time.Time `gorm:"column:received_at;not null"`
}

func (billingSourceEmailRecord) TableName() string {
	return "emails"
}

// BillingRepository persists billings into MySQL.
type BillingRepository struct {
	db    *gorm.DB
	clock timewrapper.ClockInterface
	log   logger.Interface
}

// NewBillingRepository creates a billing repository backed by MySQL.
func NewBillingRepository(
	db *gorm.DB,
	clock timewrapper.ClockInterface,
	log logger.Interface,
) *BillingRepository {
	if clock == nil {
		clock = timewrapper.NewClock()
	}
	if log == nil {
		log = logger.NewNop()
	}

	return &BillingRepository{
		db:    db,
		clock: clock,
		log:   log.With(logger.Component("billing_repository")),
	}
}

// SaveIfAbsent creates a billing once per user/vendor/billing-number identity.
// For newly created billings, line-items are inserted into billing_line_items.
func (r *BillingRepository) SaveIfAbsent(
	ctx context.Context,
	billing commondomain.Billing,
) (billingapp.SaveResult, error) {
	if ctx == nil {
		return billingapp.SaveResult{}, logger.ErrNilContext
	}
	if r.db == nil {
		return billingapp.SaveResult{}, fmt.Errorf("gorm db is not configured")
	}
	if err := billing.Validate(); err != nil {
		return billingapp.SaveResult{}, err
	}

	reqLog := r.log
	if withContext, err := r.log.WithContext(ctx); err == nil {
		reqLog = withContext
	}

	now := r.clock.Now().UTC()
	result := billingapp.SaveResult{}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		billingSummaryDate, err := r.resolveBillingSummaryDate(tx, billing)
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				reqLog.Error("db_query_failed",
					logger.String("db_system", "mysql"),
					logger.String("table", "emails"),
					logger.String("operation", "find_source_email_for_billing"),
					logger.Err(err),
				)
			}
			return fmt.Errorf("failed to resolve billing summary date: %w", err)
		}

		record := billingRecord{
			UserID:             billing.UserID,
			VendorID:           billing.VendorID,
			EmailID:            billing.EmailID,
			ProductNameDisplay: cloneOptionalString(billing.ProductNameDisplay),
			BillingNumber:      billing.BillingNumber.String(),
			InvoiceNumber:      invoiceNumberPtr(billing.InvoiceNumber),
			BillingDate:        cloneBillingDate(billing.BillingDate),
			BillingSummaryDate: billingSummaryDate,
			PaymentCycle:       billing.PaymentCycle.String(),
			CreatedAt:          now,
			UpdatedAt:          now,
		}

		if err := tx.Create(&record).Error; err != nil {
			if isDuplicatedKeyError(err) {
				return gorm.ErrDuplicatedKey
			}
			reqLog.Error("db_query_failed",
				logger.String("db_system", "mysql"),
				logger.String("table", "billings"),
				logger.String("operation", "create"),
				logger.Err(err),
			)
			return fmt.Errorf("failed to create billing: %w", err)
		}

		result = billingapp.SaveResult{
			BillingID: record.ID,
			Duplicate: false,
		}

		if err := r.saveLineItems(tx, record.ID, billing.UserID, billing.LineItems, now); err != nil {
			reqLog.Error("db_query_failed",
				logger.String("db_system", "mysql"),
				logger.String("table", "billing_line_items"),
				logger.String("operation", "create"),
				logger.Err(err),
			)
			return fmt.Errorf("failed to create billing line items: %w", err)
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			existing, findErr := r.findByIdentity(r.db.WithContext(ctx), billing.UserID, billing.VendorID, billing.BillingNumber.String())
			if findErr != nil {
				if !errors.Is(findErr, gorm.ErrRecordNotFound) {
					reqLog.Error("db_query_failed",
						logger.String("db_system", "mysql"),
						logger.String("table", "billings"),
						logger.String("operation", "find_by_identity"),
						logger.Err(findErr),
					)
				}
				return billingapp.SaveResult{}, fmt.Errorf("failed to find billing by identity: %w", findErr)
			}
			return billingapp.SaveResult{
				BillingID: existing.ID,
				Duplicate: true,
			}, nil
		}
		return billingapp.SaveResult{}, err
	}

	return result, nil
}

func (r *BillingRepository) saveLineItems(
	tx *gorm.DB,
	billingID uint,
	userID uint,
	lineItems []commondomain.BillingLineItem,
	now time.Time,
) error {
	if len(lineItems) == 0 {
		return nil
	}

	records := make([]billingLineItemRecord, 0, len(lineItems))
	for idx, item := range lineItems {
		item = item.Normalize()
		if item.IsEmpty() {
			continue
		}
		amount, err := decimalPtr(item.Amount)
		if err != nil {
			return fmt.Errorf("line_items[%d].amount: %w", idx, err)
		}
		records = append(records, billingLineItemRecord{
			BillingID:          billingID,
			UserID:             userID,
			Position:           idx,
			ProductNameRaw:     cloneOptionalString(item.ProductNameRaw),
			ProductNameDisplay: cloneOptionalString(item.ProductNameDisplay),
			Amount:             amount,
			Currency:           normalizeOptionalCurrency(item.Currency),
			CreatedAt:          now,
			UpdatedAt:          now,
		})
	}

	if len(records) == 0 {
		return nil
	}

	return tx.Create(&records).Error
}

func (r *BillingRepository) resolveBillingSummaryDate(tx *gorm.DB, billing commondomain.Billing) (time.Time, error) {
	if billing.BillingDate != nil {
		return billing.BillingDate.UTC(), nil
	}

	var sourceEmail billingSourceEmailRecord
	err := tx.
		Select("id", "user_id", "received_at").
		Where("id = ? AND user_id = ?", billing.EmailID, billing.UserID).
		Take(&sourceEmail).
		Error
	if err != nil {
		return time.Time{}, err
	}

	return sourceEmail.ReceivedAt.UTC(), nil
}

func (r *BillingRepository) findByIdentity(tx *gorm.DB, userID, vendorID uint, billingNumber string) (billingRecord, error) {
	var record billingRecord
	err := tx.
		Where("user_id = ? AND vendor_id = ? AND billing_number = ?", userID, vendorID, billingNumber).
		Take(&record).
		Error
	if err != nil {
		return billingRecord{}, err
	}
	return record, nil
}

func invoiceNumberPtr(value commondomain.InvoiceNumber) *string {
	if value.IsEmpty() {
		return nil
	}
	stringValue := value.String()
	return &stringValue
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
	cloned := strings.TrimSpace(*value)
	if cloned == "" {
		return nil
	}
	return &cloned
}

func normalizeOptionalCurrency(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	upper := strings.ToUpper(trimmed)
	return &upper
}

func decimalPtr(value *float64) (*decimal.Decimal, error) {
	if value == nil {
		return nil, nil
	}
	if math.IsNaN(*value) || math.IsInf(*value, 0) {
		return nil, fmt.Errorf("amount must be finite")
	}
	raw := strconv.FormatFloat(*value, 'f', -1, 64)
	parsed, err := decimal.NewFromString(raw)
	if err != nil {
		return nil, fmt.Errorf("amount must be a decimal number: %w", err)
	}
	return &parsed, nil
}

func isDuplicatedKeyError(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}

	var mysqlErr *mysqlDriver.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
