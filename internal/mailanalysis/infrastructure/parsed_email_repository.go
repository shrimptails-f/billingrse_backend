package infrastructure

import (
	"business/internal/library/logger"
	madomain "business/internal/mailanalysis/domain"
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type parsedEmailRecord struct {
	ID                 uint       `gorm:"column:id;primaryKey;autoIncrement"`
	UserID             uint       `gorm:"column:user_id;not null;index:idx_parsed_emails_user_email,priority:1"`
	EmailID            uint       `gorm:"column:email_id;not null;index:idx_parsed_emails_user_email,priority:2"`
	AnalysisRunID      string     `gorm:"column:analysis_run_id;type:char(36);not null;index:idx_parsed_emails_analysis_run;uniqueIndex:uni_parsed_emails_run_position,priority:1"`
	Position           int        `gorm:"column:position;not null;uniqueIndex:uni_parsed_emails_run_position,priority:2"`
	ProductNameRaw     *string    `gorm:"column:product_name_raw;type:text"`
	ProductNameDisplay *string    `gorm:"column:product_name_display;size:255"`
	VendorName         *string    `gorm:"column:vendor_name;type:text"`
	BillingNumber      *string    `gorm:"column:billing_number;size:255"`
	InvoiceNumber      *string    `gorm:"column:invoice_number;size:14"`
	Amount             *float64   `gorm:"column:amount;type:decimal(18,3)"`
	Currency           *string    `gorm:"column:currency;type:char(3)"`
	BillingDate        *time.Time `gorm:"column:billing_date"`
	PaymentCycle       *string    `gorm:"column:payment_cycle;size:32"`
	ExtractedAt        time.Time  `gorm:"column:extracted_at;not null"`
	PromptVersion      string     `gorm:"column:prompt_version;size:50;not null"`
	CreatedAt          time.Time  `gorm:"column:created_at;not null"`
	UpdatedAt          time.Time  `gorm:"column:updated_at;not null"`
}

func (parsedEmailRecord) TableName() string {
	return "parsed_emails"
}

// GormParsedEmailRepositoryAdapter persists ParsedEmail history into MySQL.
type GormParsedEmailRepositoryAdapter struct {
	db  *gorm.DB
	log logger.Interface
}

// NewGormParsedEmailRepositoryAdapter creates a Gorm-backed ParsedEmail repository.
func NewGormParsedEmailRepositoryAdapter(db *gorm.DB, log logger.Interface) *GormParsedEmailRepositoryAdapter {
	if log == nil {
		log = logger.NewNop()
	}

	return &GormParsedEmailRepositoryAdapter{
		db:  db,
		log: log.With(logger.Component("parsed_email_repository")),
	}
}

// SaveAll appends ParsedEmail history rows for a single analysis run.
func (r *GormParsedEmailRepositoryAdapter) SaveAll(ctx context.Context, input madomain.SaveInput) ([]madomain.ParsedEmailRecord, error) {
	if ctx == nil {
		return nil, logger.ErrNilContext
	}
	if r.db == nil {
		return nil, fmt.Errorf("gorm db is not configured")
	}

	input = input.Normalize()
	if len(input.ParsedEmails) == 0 {
		return nil, nil
	}
	if err := input.Validate(); err != nil {
		return nil, err
	}

	reqLog := r.log
	if withContext, err := r.log.WithContext(ctx); err == nil {
		reqLog = withContext
	}

	now := time.Now().UTC()
	records := make([]parsedEmailRecord, 0, len(input.ParsedEmails))
	for idx, parsedEmail := range input.ParsedEmails {
		parsed := parsedEmail.WithExtractedAt(input.ExtractedAt)
		records = append(records, parsedEmailRecord{
			UserID:             input.UserID,
			EmailID:            input.EmailID,
			AnalysisRunID:      input.AnalysisRunID,
			Position:           input.PositionBase + idx,
			ProductNameRaw:     parsed.ProductNameRaw,
			ProductNameDisplay: parsed.ProductNameDisplay,
			VendorName:         parsed.VendorName,
			BillingNumber:      parsed.BillingNumber,
			InvoiceNumber:      parsed.InvoiceNumber,
			Amount:             parsed.Amount,
			Currency:           parsed.Currency,
			BillingDate:        parsed.BillingDate,
			PaymentCycle:       parsed.PaymentCycle,
			ExtractedAt:        parsed.ExtractedAt,
			PromptVersion:      input.PromptVersion,
			CreatedAt:          now,
			UpdatedAt:          now,
		})
	}

	if err := r.db.WithContext(ctx).Create(&records).Error; err != nil {
		reqLog.Error("db_query_failed",
			logger.String("db_system", "mysql"),
			logger.String("table", "parsed_emails"),
			logger.String("operation", "create"),
			logger.Err(err),
		)
		return nil, fmt.Errorf("failed to create parsed emails: %w", err)
	}

	result := make([]madomain.ParsedEmailRecord, 0, len(records))
	for _, record := range records {
		result = append(result, madomain.ParsedEmailRecord{
			ID:      record.ID,
			EmailID: record.EmailID,
		})
	}

	return result, nil
}
