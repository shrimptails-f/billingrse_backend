package infrastructure

import (
	billingapp "business/internal/billing/application"
	commondomain "business/internal/common/domain"
	"business/internal/library/logger"
	"business/internal/library/timewrapper"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type billingRecord struct {
	ID                 uint            `gorm:"column:id;primaryKey;autoIncrement"`
	UserID             uint            `gorm:"column:user_id;not null;uniqueIndex:uni_billings_user_vendor_number,priority:1"`
	VendorID           uint            `gorm:"column:vendor_id;not null;uniqueIndex:uni_billings_user_vendor_number,priority:2"`
	EmailID            uint            `gorm:"column:email_id;not null"`
	ProductNameDisplay *string         `gorm:"column:product_name_display;size:255"`
	BillingNumber      string          `gorm:"column:billing_number;size:255;not null;uniqueIndex:uni_billings_user_vendor_number,priority:3"`
	InvoiceNumber      *string         `gorm:"column:invoice_number;size:14"`
	Amount             decimal.Decimal `gorm:"column:amount;type:decimal(18,3);not null"`
	Currency           string          `gorm:"column:currency;type:char(3);not null"`
	BillingDate        *time.Time      `gorm:"column:billing_date"`
	PaymentCycle       string          `gorm:"column:payment_cycle;size:32;not null"`
	CreatedAt          time.Time       `gorm:"column:created_at;not null"`
	UpdatedAt          time.Time       `gorm:"column:updated_at;not null"`
}

func (billingRecord) TableName() string {
	return "billings"
}

// GormBillingRepository persists billings into MySQL.
type GormBillingRepository struct {
	db    *gorm.DB
	clock timewrapper.ClockInterface
	log   logger.Interface
}

// NewGormBillingRepository creates a Gorm-backed billing repository.
func NewGormBillingRepository(
	db *gorm.DB,
	clock timewrapper.ClockInterface,
	log logger.Interface,
) *GormBillingRepository {
	if clock == nil {
		clock = timewrapper.NewClock()
	}
	if log == nil {
		log = logger.NewNop()
	}

	return &GormBillingRepository{
		db:    db,
		clock: clock,
		log:   log.With(logger.Component("billing_repository")),
	}
}

// SaveIfAbsent creates a billing once per user/vendor/billing-number identity.
func (r *GormBillingRepository) SaveIfAbsent(ctx context.Context, billing commondomain.Billing) (billingapp.SaveResult, error) {
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
	record := billingRecord{
		UserID:             billing.UserID,
		VendorID:           billing.VendorID,
		EmailID:            billing.EmailID,
		ProductNameDisplay: cloneOptionalString(billing.ProductNameDisplay),
		BillingNumber:      billing.BillingNumber.String(),
		InvoiceNumber:      invoiceNumberPtr(billing.InvoiceNumber),
		Amount:             billing.Money.Amount,
		Currency:           billing.Money.Currency,
		BillingDate:        cloneBillingDate(billing.BillingDate),
		PaymentCycle:       billing.PaymentCycle.String(),
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	tx := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "vendor_id"},
			{Name: "billing_number"},
		},
		DoNothing: true,
	})
	if err := tx.Create(&record).Error; err != nil {
		reqLog.Error("db_query_failed",
			logger.String("db_system", "mysql"),
			logger.String("table", "billings"),
			logger.String("operation", "create"),
			logger.Err(err),
		)
		return billingapp.SaveResult{}, fmt.Errorf("failed to create billing: %w", err)
	}

	if record.ID != 0 {
		return billingapp.SaveResult{
			BillingID: record.ID,
			Duplicate: false,
		}, nil
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

	return billingapp.SaveResult{
		BillingID: existing.ID,
		Duplicate: true,
	}, nil
}

func (r *GormBillingRepository) findByIdentity(ctx context.Context, userID, vendorID uint, billingNumber string) (billingRecord, error) {
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
	cloned := *value
	return &cloned
}
