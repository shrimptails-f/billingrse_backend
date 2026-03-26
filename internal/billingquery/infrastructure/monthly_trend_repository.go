package infrastructure

import (
	billingqueryapp "business/internal/billingquery/application"
	"business/internal/library/logger"
	"context"
	"fmt"

	"github.com/shopspring/decimal"
)

type billingMonthlyTrendRow struct {
	Year                 int64           `gorm:"column:year"`
	Month                int64           `gorm:"column:month"`
	TotalAmount          decimal.Decimal `gorm:"column:total_amount"`
	BillingCount         int64           `gorm:"column:billing_count"`
	FallbackBillingCount int64           `gorm:"column:fallback_billing_count"`
}

// MonthlyTrend returns the raw monthly totals for the requested fixed window.
func (r *BillingQueryRepository) MonthlyTrend(
	ctx context.Context,
	query billingqueryapp.MonthlyTrendQuery,
) ([]billingqueryapp.MonthlyTrendAggregate, error) {
	if ctx == nil {
		return nil, logger.ErrNilContext
	}
	if r.db == nil {
		return nil, fmt.Errorf("gorm db is not configured")
	}

	query = query.Normalize(r.clock.Now())
	if err := query.Validate(); err != nil {
		return nil, err
	}

	reqLog := r.log
	if withContext, err := r.log.WithContext(ctx); err == nil {
		reqLog = withContext
	}

	const billingSummaryDateExpr = "billings.billing_summary_date"

	var rows []billingMonthlyTrendRow
	if err := r.db.WithContext(ctx).
		Table("billings").
		Select([]string{
			"YEAR(" + billingSummaryDateExpr + ") AS year",
			"MONTH(" + billingSummaryDateExpr + ") AS month",
			"SUM(billings.amount) AS total_amount",
			"COUNT(*) AS billing_count",
			"SUM(CASE WHEN billings.billing_date IS NULL THEN 1 ELSE 0 END) AS fallback_billing_count",
		}).
		Where("billings.user_id = ?", query.UserID).
		Where("billings.currency = ?", query.Currency).
		Where(billingSummaryDateExpr+" >= ?", query.WindowStartAt()).
		Where(billingSummaryDateExpr+" < ?", query.WindowEndAtExclusive()).
		Group("YEAR(" + billingSummaryDateExpr + "), MONTH(" + billingSummaryDateExpr + ")").
		Order("YEAR(" + billingSummaryDateExpr + ") ASC, MONTH(" + billingSummaryDateExpr + ") ASC").
		Scan(&rows).Error; err != nil {
		reqLog.Error("db_query_failed",
			logger.String("db_system", "mysql"),
			logger.String("table", "billings"),
			logger.String("operation", "monthly_trend"),
			logger.Err(err),
		)
		return nil, fmt.Errorf("failed to load billing monthly trend: %w", err)
	}

	items := make([]billingqueryapp.MonthlyTrendAggregate, 0, len(rows))
	for _, row := range rows {
		items = append(items, billingqueryapp.MonthlyTrendAggregate{
			YearMonth:            fmt.Sprintf("%04d-%02d", row.Year, row.Month),
			TotalAmount:          row.TotalAmount,
			BillingCount:         int(row.BillingCount),
			FallbackBillingCount: int(row.FallbackBillingCount),
		})
	}

	return items, nil
}
