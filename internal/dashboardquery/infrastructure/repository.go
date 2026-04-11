package infrastructure

import (
	dashboardqueryapp "business/internal/dashboardquery/application"
	"business/internal/library/logger"
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type parsedEmailSummaryRecord struct {
	ID          uint      `gorm:"column:id;primaryKey;autoIncrement"`
	UserID      uint      `gorm:"column:user_id;not null;index:idx_parsed_emails_user_email,priority:1"`
	EmailID     uint      `gorm:"column:email_id;not null;index:idx_parsed_emails_user_email,priority:2"`
	ExtractedAt time.Time `gorm:"column:extracted_at;not null"`
	CreatedAt   time.Time `gorm:"column:created_at;not null"`
	UpdatedAt   time.Time `gorm:"column:updated_at;not null"`
}

func (parsedEmailSummaryRecord) TableName() string {
	return "parsed_emails"
}

type billingSummaryRecord struct {
	ID                 uint       `gorm:"column:id;primaryKey;autoIncrement"`
	UserID             uint       `gorm:"column:user_id;not null;index:idx_billings_user_summary_date_id,priority:1"`
	BillingDate        *time.Time `gorm:"column:billing_date"`
	BillingSummaryDate time.Time  `gorm:"column:billing_summary_date;not null;index:idx_billings_user_summary_date_id,priority:2"`
	CreatedAt          time.Time  `gorm:"column:created_at;not null"`
	UpdatedAt          time.Time  `gorm:"column:updated_at;not null"`
}

func (billingSummaryRecord) TableName() string {
	return "billings"
}

type billingCountsRow struct {
	TotalSavedBillingCount           int64 `gorm:"column:total_saved_billing_count"`
	CurrentMonthFallbackBillingCount int64 `gorm:"column:current_month_fallback_billing_count"`
}

// DashboardSummaryRepository loads dashboard summary read models from MySQL.
type DashboardSummaryRepository struct {
	db  *gorm.DB
	log logger.Interface
}

// NewDashboardSummaryRepository creates a dashboard summary repository backed by MySQL.
func NewDashboardSummaryRepository(
	db *gorm.DB,
	log logger.Interface,
) *DashboardSummaryRepository {
	if log == nil {
		log = logger.NewNop()
	}

	return &DashboardSummaryRepository{
		db:  db,
		log: log.With(logger.Component("dashboard_summary_repository")),
	}
}

// CountCurrentMonthAnalysisSuccess counts the parsed emails saved in the current UTC month.
func (r *DashboardSummaryRepository) CountCurrentMonthAnalysisSuccess(
	ctx context.Context,
	userID uint,
	monthStartAt,
	nextMonthStartAt time.Time,
) (int, error) {
	if ctx == nil {
		return 0, logger.ErrNilContext
	}
	if r.db == nil {
		return 0, fmt.Errorf("gorm db is not configured")
	}
	if userID == 0 {
		return 0, fmt.Errorf("user_id is required")
	}
	if monthStartAt.IsZero() || nextMonthStartAt.IsZero() || !monthStartAt.Before(nextMonthStartAt) {
		return 0, fmt.Errorf("invalid current month range")
	}

	reqLog := r.log
	if withContext, err := r.log.WithContext(ctx); err == nil {
		reqLog = withContext
	}

	var count int64
	if err := r.db.WithContext(ctx).
		Table("parsed_emails").
		Where("user_id = ?", userID).
		Where("extracted_at >= ?", monthStartAt.UTC()).
		Where("extracted_at < ?", nextMonthStartAt.UTC()).
		Count(&count).Error; err != nil {
		reqLog.Error("db_query_failed",
			logger.String("db_system", "mysql"),
			logger.String("table", "parsed_emails"),
			logger.String("operation", "count_current_month_analysis_success"),
			logger.Err(err),
		)
		return 0, fmt.Errorf("failed to count current month analysis success: %w", err)
	}

	return int(count), nil
}

// GetBillingCounts loads the total saved billing count and current-month fallback billing count.
func (r *DashboardSummaryRepository) GetBillingCounts(
	ctx context.Context,
	userID uint,
	monthStartAt,
	nextMonthStartAt time.Time,
) (dashboardqueryapp.BillingCounts, error) {
	if ctx == nil {
		return dashboardqueryapp.BillingCounts{}, logger.ErrNilContext
	}
	if r.db == nil {
		return dashboardqueryapp.BillingCounts{}, fmt.Errorf("gorm db is not configured")
	}
	if userID == 0 {
		return dashboardqueryapp.BillingCounts{}, fmt.Errorf("user_id is required")
	}
	if monthStartAt.IsZero() || nextMonthStartAt.IsZero() || !monthStartAt.Before(nextMonthStartAt) {
		return dashboardqueryapp.BillingCounts{}, fmt.Errorf("invalid current month range")
	}

	reqLog := r.log
	if withContext, err := r.log.WithContext(ctx); err == nil {
		reqLog = withContext
	}

	var row billingCountsRow
	if err := r.db.WithContext(ctx).
		Table("billings").
		Select(
			"COUNT(*) AS total_saved_billing_count, "+
				"COALESCE(SUM(CASE WHEN billing_date IS NULL AND billing_summary_date >= ? AND billing_summary_date < ? THEN 1 ELSE 0 END), 0) AS current_month_fallback_billing_count",
			monthStartAt.UTC(),
			nextMonthStartAt.UTC(),
		).
		Where("user_id = ?", userID).
		Scan(&row).Error; err != nil {
		reqLog.Error("db_query_failed",
			logger.String("db_system", "mysql"),
			logger.String("table", "billings"),
			logger.String("operation", "count_dashboard_summary_billings"),
			logger.Err(err),
		)
		return dashboardqueryapp.BillingCounts{}, fmt.Errorf("failed to count dashboard summary billings: %w", err)
	}

	return dashboardqueryapp.BillingCounts{
		TotalSavedBillingCount:           int(row.TotalSavedBillingCount),
		CurrentMonthFallbackBillingCount: int(row.CurrentMonthFallbackBillingCount),
	}, nil
}
