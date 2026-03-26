package infrastructure

import (
	billingqueryapp "business/internal/billingquery/application"
	"business/internal/library/logger"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type billingMonthDetailTotalsRow struct {
	TotalAmount          decimal.Decimal `gorm:"column:total_amount"`
	BillingCount         int64           `gorm:"column:billing_count"`
	FallbackBillingCount int64           `gorm:"column:fallback_billing_count"`
}

type billingMonthDetailVendorRow struct {
	VendorName   string          `gorm:"column:vendor_name"`
	TotalAmount  decimal.Decimal `gorm:"column:total_amount"`
	BillingCount int64           `gorm:"column:billing_count"`
}

// MonthDetail returns the raw totals and vendor breakdown for one selected month.
func (r *BillingQueryRepository) MonthDetail(ctx context.Context, query billingqueryapp.MonthDetailQuery) (billingqueryapp.MonthDetailReadModel, error) {
	if ctx == nil {
		return billingqueryapp.MonthDetailReadModel{}, logger.ErrNilContext
	}
	if r.db == nil {
		return billingqueryapp.MonthDetailReadModel{}, fmt.Errorf("gorm db is not configured")
	}

	query = query.Normalize()
	if err := query.Validate(); err != nil {
		return billingqueryapp.MonthDetailReadModel{}, err
	}

	monthStart, monthEnd, err := query.MonthRange()
	if err != nil {
		return billingqueryapp.MonthDetailReadModel{}, err
	}

	reqLog := r.log
	if withContext, err := r.log.WithContext(ctx); err == nil {
		reqLog = withContext
	}

	baseQuery := r.buildMonthDetailBaseQuery(ctx, query, monthStart, monthEnd)

	var totals billingMonthDetailTotalsRow
	if err := baseQuery.
		Select([]string{
			"COALESCE(SUM(COALESCE(billing_line_items.amount, 0)), 0) AS total_amount",
			"COUNT(DISTINCT billings.id) AS billing_count",
			"COUNT(DISTINCT CASE WHEN billings.billing_date IS NULL THEN billings.id END) AS fallback_billing_count",
		}).
		Scan(&totals).Error; err != nil {
		reqLog.Error("db_query_failed",
			logger.String("db_system", "mysql"),
			logger.String("table", "billings"),
			logger.String("operation", "month_detail_totals"),
			logger.Err(err),
		)
		return billingqueryapp.MonthDetailReadModel{}, fmt.Errorf("failed to load billing month detail totals: %w", err)
	}

	var rows []billingMonthDetailVendorRow
	if err := r.buildMonthDetailBaseQuery(ctx, query, monthStart, monthEnd).
		Joins("INNER JOIN vendors ON vendors.id = billings.vendor_id").
		Select([]string{
			"vendors.name AS vendor_name",
			"SUM(COALESCE(billing_line_items.amount, 0)) AS total_amount",
			"COUNT(DISTINCT billings.id) AS billing_count",
		}).
		Group("vendors.id, vendors.name").
		Order("SUM(COALESCE(billing_line_items.amount, 0)) DESC").
		Order("vendors.name ASC").
		Scan(&rows).Error; err != nil {
		reqLog.Error("db_query_failed",
			logger.String("db_system", "mysql"),
			logger.String("table", "billings"),
			logger.String("operation", "month_detail_vendor_breakdown"),
			logger.Err(err),
		)
		return billingqueryapp.MonthDetailReadModel{}, fmt.Errorf("failed to load billing month detail vendor breakdown: %w", err)
	}

	vendorItems := make([]billingqueryapp.MonthDetailVendorAggregate, 0, len(rows))
	for _, row := range rows {
		vendorItems = append(vendorItems, billingqueryapp.MonthDetailVendorAggregate{
			VendorName:   strings.TrimSpace(row.VendorName),
			TotalAmount:  row.TotalAmount,
			BillingCount: int(row.BillingCount),
		})
	}

	return billingqueryapp.MonthDetailReadModel{
		TotalAmount:          totals.TotalAmount,
		BillingCount:         int(totals.BillingCount),
		FallbackBillingCount: int(totals.FallbackBillingCount),
		VendorItems:          vendorItems,
	}, nil
}

func (r *BillingQueryRepository) buildMonthDetailBaseQuery(
	ctx context.Context,
	query billingqueryapp.MonthDetailQuery,
	monthStart time.Time,
	monthEnd time.Time,
) *gorm.DB {
	return r.db.WithContext(ctx).
		Table("billings").
		Joins("INNER JOIN billing_line_items ON billing_line_items.billing_id = billings.id AND billing_line_items.user_id = billings.user_id").
		Where("billings.user_id = ?", query.UserID).
		Where("billing_line_items.currency = ?", query.Currency).
		Where("billings.billing_summary_date >= ?", monthStart).
		Where("billings.billing_summary_date < ?", monthEnd)
}
