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

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	lineItems []billingapp.CreationLineItem,
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

	billingSummaryDate, err := r.resolveBillingSummaryDate(ctx, billing)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			reqLog.Error("db_query_failed",
				logger.String("db_system", "mysql"),
				logger.String("table", "emails"),
				logger.String("operation", "find_source_email_for_billing"),
				logger.Err(err),
			)
		}
		return billingapp.SaveResult{}, fmt.Errorf("failed to resolve billing summary date: %w", err)
	}

	now := r.clock.Now().UTC()
	hasAmountColumn := r.db.Migrator().HasColumn("billings", "amount")
	hasCurrencyColumn := r.db.Migrator().HasColumn("billings", "currency")
	if hasAmountColumn != hasCurrencyColumn {
		return billingapp.SaveResult{}, fmt.Errorf("billings schema mismatch: amount and currency columns must be both present or both absent")
	}

	recordValues := map[string]any{
		"user_id":              billing.UserID,
		"vendor_id":            billing.VendorID,
		"email_id":             billing.EmailID,
		"product_name_display": cloneOptionalString(billing.ProductNameDisplay),
		"billing_number":       billing.BillingNumber.String(),
		"invoice_number":       invoiceNumberPtr(billing.InvoiceNumber),
		"billing_date":         cloneBillingDate(billing.BillingDate),
		"billing_summary_date": billingSummaryDate,
		"payment_cycle":        billing.PaymentCycle.String(),
		"created_at":           now,
		"updated_at":           now,
	}
	if hasAmountColumn {
		recordValues["amount"] = billing.Money.Amount
		recordValues["currency"] = billing.Money.Currency
	}

	createResult := r.db.WithContext(ctx).
		Table("billings").
		Clauses(clause.Insert{Modifier: "IGNORE"}).
		Create(recordValues)
	if err := createResult.Error; err != nil {
		reqLog.Error("db_query_failed",
			logger.String("db_system", "mysql"),
			logger.String("table", "billings"),
			logger.String("operation", "create"),
			logger.Err(err),
		)
		return billingapp.SaveResult{}, fmt.Errorf("failed to create billing: %w", err)
	}

	existing, err := r.findByIdentity(ctx, billing.UserID, billing.VendorID, billing.BillingNumber.String())
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			reqLog.Error("db_query_failed",
				logger.String("db_system", "mysql"),
				logger.String("table", "billings"),
				logger.String("operation", "find_by_identity"),
				logger.Err(err),
			)
		}
		return billingapp.SaveResult{}, fmt.Errorf("failed to find billing by identity: %w", err)
	}

	if createResult.RowsAffected > 0 {
		if err := r.saveLineItems(ctx, existing.ID, billing.UserID, lineItems, now); err != nil {
			reqLog.Error("db_query_failed",
				logger.String("db_system", "mysql"),
				logger.String("table", "billing_line_items"),
				logger.String("operation", "create"),
				logger.Err(err),
			)
			return billingapp.SaveResult{}, fmt.Errorf("failed to create billing line items: %w", err)
		}
		return billingapp.SaveResult{
			BillingID: existing.ID,
			Duplicate: false,
		}, nil
	}

	return billingapp.SaveResult{
		BillingID: existing.ID,
		Duplicate: true,
	}, nil
}

func (r *BillingRepository) saveLineItems(
	ctx context.Context,
	billingID uint,
	userID uint,
	lineItems []billingapp.CreationLineItem,
	now time.Time,
) error {
	if len(lineItems) == 0 {
		return nil
	}

	records := make([]billingLineItemRecord, 0, len(lineItems))
	for idx, item := range lineItems {
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

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "billing_id"},
			{Name: "position"},
		},
		DoNothing: true,
	}).Create(&records).Error
}

func (r *BillingRepository) resolveBillingSummaryDate(ctx context.Context, billing commondomain.Billing) (time.Time, error) {
	if billing.BillingDate != nil {
		return billing.BillingDate.UTC(), nil
	}

	var sourceEmail billingSourceEmailRecord
	err := r.db.WithContext(ctx).
		Select("id", "user_id", "received_at").
		Where("id = ? AND user_id = ?", billing.EmailID, billing.UserID).
		Take(&sourceEmail).
		Error
	if err != nil {
		return time.Time{}, err
	}

	return sourceEmail.ReceivedAt.UTC(), nil
}

func (r *BillingRepository) findByIdentity(ctx context.Context, userID, vendorID uint, billingNumber string) (billingRecord, error) {
	var record billingRecord
	err := r.db.WithContext(ctx).
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
