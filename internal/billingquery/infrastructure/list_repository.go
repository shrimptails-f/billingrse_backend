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

type billingListRow struct {
	EmailID            uint             `gorm:"column:email_id"`
	ExternalMessageID  string           `gorm:"column:external_message_id"`
	VendorName         string           `gorm:"column:vendor_name"`
	ReceivedAt         time.Time        `gorm:"column:received_at"`
	BillingDate        *time.Time       `gorm:"column:billing_date"`
	ProductNameDisplay *string          `gorm:"column:product_name_display"`
	Amount             *decimal.Decimal `gorm:"column:amount"`
	Currency           *string          `gorm:"column:currency"`
}

type billingListEmailRecord struct {
	ID                uint      `gorm:"column:id;primaryKey;autoIncrement"`
	UserID            uint      `gorm:"column:user_id;not null"`
	Provider          string    `gorm:"column:provider;size:50;not null"`
	AccountIdentifier string    `gorm:"column:account_identifier;size:255;not null"`
	ExternalMessageID string    `gorm:"column:external_message_id;size:255;not null"`
	Subject           string    `gorm:"column:subject;type:text;not null"`
	FromRaw           string    `gorm:"column:from_raw;type:text;not null"`
	ToJSON            string    `gorm:"column:to_json;type:json;not null"`
	BodyDigest        string    `gorm:"column:body_digest;size:64;not null"`
	ReceivedAt        time.Time `gorm:"column:received_at;not null"`
	CreatedAt         time.Time `gorm:"column:created_at;not null"`
	UpdatedAt         time.Time `gorm:"column:updated_at;not null"`
}

func (billingListEmailRecord) TableName() string {
	return "emails"
}

type billingListVendorRecord struct {
	ID             uint      `gorm:"column:id;primaryKey;autoIncrement"`
	UserID         uint      `gorm:"column:user_id;not null;uniqueIndex:uni_vendors_user_normalized_name,priority:1"`
	Name           string    `gorm:"column:name;size:255;not null"`
	NormalizedName string    `gorm:"column:normalized_name;size:255;not null;uniqueIndex:uni_vendors_user_normalized_name,priority:2"`
	CreatedAt      time.Time `gorm:"column:created_at;not null"`
	UpdatedAt      time.Time `gorm:"column:updated_at;not null"`
}

func (billingListVendorRecord) TableName() string {
	return "vendors"
}

// List loads billing list read models joined with vendor and email metadata.
func (r *BillingQueryRepository) List(ctx context.Context, query billingqueryapp.ListQuery) (billingqueryapp.ListResult, error) {
	if ctx == nil {
		return billingqueryapp.ListResult{}, logger.ErrNilContext
	}
	if r.db == nil {
		return billingqueryapp.ListResult{}, fmt.Errorf("gorm db is not configured")
	}

	query = query.Normalize()
	if err := query.Validate(); err != nil {
		return billingqueryapp.ListResult{}, err
	}

	reqLog := r.log
	if withContext, err := r.log.WithContext(ctx); err == nil {
		reqLog = withContext
	}

	var totalCount int64
	countQuery := r.buildListBaseQuery(ctx, query)
	if err := countQuery.Distinct("billings.id").Count(&totalCount).Error; err != nil {
		reqLog.Error("db_query_failed",
			logger.String("db_system", "mysql"),
			logger.String("table", "billings"),
			logger.String("operation", "list_count"),
			logger.Err(err),
		)
		return billingqueryapp.ListResult{}, fmt.Errorf("failed to count billings: %w", err)
	}

	var rows []billingListRow
	itemsQuery := r.buildListBaseQuery(ctx, query).
		Select([]string{
			"billings.email_id AS email_id",
			"emails.external_message_id AS external_message_id",
			"vendors.name AS vendor_name",
			"emails.received_at AS received_at",
			"billings.billing_date AS billing_date",
			"billings.product_name_display AS product_name_display",
			"line_item_summary.amount AS amount",
			"line_item_summary.currency AS currency",
		}).
		Limit(*query.Limit).
		Offset(*query.Offset)
	itemsQuery = applyBillingListOrder(itemsQuery, query)

	if err := itemsQuery.Scan(&rows).Error; err != nil {
		reqLog.Error("db_query_failed",
			logger.String("db_system", "mysql"),
			logger.String("table", "billings"),
			logger.String("operation", "list_items"),
			logger.Err(err),
		)
		return billingqueryapp.ListResult{}, fmt.Errorf("failed to list billings: %w", err)
	}

	items := make([]billingqueryapp.ListItem, 0, len(rows))
	for _, row := range rows {
		amount := 0.0
		if row.Amount != nil {
			amount = row.Amount.InexactFloat64()
		}
		currency := ""
		if row.Currency != nil {
			currency = strings.TrimSpace(*row.Currency)
		}

		items = append(items, billingqueryapp.ListItem{
			EmailID:            row.EmailID,
			ExternalMessageID:  strings.TrimSpace(row.ExternalMessageID),
			VendorName:         strings.TrimSpace(row.VendorName),
			ReceivedAt:         row.ReceivedAt.UTC(),
			BillingDate:        cloneBillingDate(row.BillingDate),
			ProductNameDisplay: cloneOptionalString(row.ProductNameDisplay),
			Amount:             amount,
			Currency:           currency,
		})
	}

	return billingqueryapp.ListResult{
		Items:      items,
		Limit:      *query.Limit,
		Offset:     *query.Offset,
		TotalCount: totalCount,
	}, nil
}

func (r *BillingQueryRepository) buildListBaseQuery(ctx context.Context, query billingqueryapp.ListQuery) *gorm.DB {
	tx := r.db.WithContext(ctx).
		Table("billings").
		Joins("INNER JOIN vendors ON vendors.id = billings.vendor_id AND vendors.user_id = billings.user_id").
		Joins("INNER JOIN emails ON emails.id = billings.email_id AND emails.user_id = billings.user_id").
		Joins("LEFT JOIN "+billingListLineItemSummarySubQuery()+" ON line_item_summary.billing_id = billings.id").
		Where("billings.user_id = ?", query.UserID)

	if query.EmailID != nil {
		tx = tx.Where("billings.email_id = ?", *query.EmailID)
	}
	if query.ExternalMessageID != "" {
		tx = tx.Where("emails.user_id = ? AND emails.external_message_id = ?", query.UserID, query.ExternalMessageID)
	}
	if query.Q != "" {
		pattern := "%" + strings.ToLower(query.Q) + "%"
		tx = tx.Where(`(
			LOWER(vendors.name) LIKE ?
			OR LOWER(COALESCE(billings.product_name_display, '')) LIKE ?
			OR LOWER(billings.billing_number) LIKE ?
			OR LOWER(emails.external_message_id) LIKE ?
		)`, pattern, pattern, pattern, pattern)
	}

	dateColumn := "billings.billing_date"
	if query.UseReceivedAtFallback != nil && *query.UseReceivedAtFallback {
		dateColumn = "billings.billing_summary_date"
	}
	if query.DateFrom != nil {
		tx = tx.Where(dateColumn+" >= ?", *query.DateFrom)
	}
	if query.DateTo != nil {
		tx = tx.Where(dateColumn+" <= ?", *query.DateTo)
	}

	return tx
}

func billingListLineItemSummarySubQuery() string {
	return `(
SELECT
	billing_id,
	CASE
		WHEN COUNT(DISTINCT CASE WHEN currency IS NOT NULL AND TRIM(currency) <> '' THEN UPPER(TRIM(currency)) END) = 1
			THEN SUM(CASE WHEN amount IS NOT NULL THEN amount ELSE 0 END)
		ELSE NULL
	END AS amount,
	CASE
		WHEN COUNT(DISTINCT CASE WHEN currency IS NOT NULL AND TRIM(currency) <> '' THEN UPPER(TRIM(currency)) END) = 1
			THEN MAX(CASE WHEN currency IS NOT NULL AND TRIM(currency) <> '' THEN UPPER(TRIM(currency)) END)
		ELSE NULL
	END AS currency
FROM billing_line_items
GROUP BY billing_id
) AS line_item_summary`
}

func applyBillingListOrder(tx *gorm.DB, query billingqueryapp.ListQuery) *gorm.DB {
	if query.UseReceivedAtFallback != nil && *query.UseReceivedAtFallback {
		return tx.
			Order("billings.billing_summary_date DESC").
			Order("billings.id DESC")
	}

	return tx.
		Order("billings.billing_date DESC").
		Order("emails.received_at DESC").
		Order("billings.id DESC")
}
