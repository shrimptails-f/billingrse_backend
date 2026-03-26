package billing

import (
	"business/internal/app/httpresponse"
	billingqueryapp "business/internal/billingquery/application"
	"business/internal/library/logger"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const billingYearMonthLayout = "2006-01"

type monthlyTrendQueryRequest struct {
	Currency       string `form:"currency"`
	WindowEndMonth string `form:"window_end_month"`
}

type monthlyTrendResponse struct {
	Currency             string                     `json:"currency"`
	WindowStartMonth     string                     `json:"window_start_month"`
	WindowEndMonth       string                     `json:"window_end_month"`
	DefaultSelectedMonth string                     `json:"default_selected_month"`
	Items                []monthlyTrendItemResponse `json:"items"`
}

type monthlyTrendItemResponse struct {
	YearMonth            string  `json:"year_month"`
	TotalAmount          float64 `json:"total_amount"`
	BillingCount         int     `json:"billing_count"`
	FallbackBillingCount int     `json:"fallback_billing_count"`
}

// MonthlyTrend handles GET /api/v1/billings/summary/monthly-trend.
func (ctrl *Controller) MonthlyTrend(c *gin.Context) {
	reqLog := ctrl.log
	if withContext, err := ctrl.log.WithContext(c.Request.Context()); err == nil {
		reqLog = withContext
	}

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	if ctrl.monthlyTrendUseCase == nil {
		reqLog.Error("billing_monthly_trend_usecase_not_configured",
			logger.UserID(userID),
		)
		httpresponse.WriteInternalServerError(c)
		return
	}

	var req monthlyTrendQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		httpresponse.WriteInvalidRequest(c)
		return
	}

	query, err := req.toApplicationQuery(userID)
	if err != nil {
		httpresponse.WriteInvalidRequest(c)
		return
	}

	result, err := ctrl.monthlyTrendUseCase.Get(c.Request.Context(), query)
	if err != nil {
		if errors.Is(err, billingqueryapp.ErrInvalidMonthlyTrendQuery) {
			httpresponse.WriteInvalidRequest(c)
			return
		}

		reqLog.Error("billing_monthly_trend_failed",
			logger.UserID(userID),
			logger.String("currency", req.Currency),
			logger.String("window_end_month", req.WindowEndMonth),
			logger.Err(err),
		)
		httpresponse.WriteInternalServerError(c)
		return
	}

	items := make([]monthlyTrendItemResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, monthlyTrendItemResponse{
			YearMonth:            item.YearMonth,
			TotalAmount:          item.TotalAmount,
			BillingCount:         item.BillingCount,
			FallbackBillingCount: item.FallbackBillingCount,
		})
	}

	c.JSON(http.StatusOK, monthlyTrendResponse{
		Currency:             result.Currency,
		WindowStartMonth:     result.WindowStartMonth,
		WindowEndMonth:       result.WindowEndMonth,
		DefaultSelectedMonth: result.DefaultSelectedMonth,
		Items:                items,
	})
}

func (r monthlyTrendQueryRequest) toApplicationQuery(userID uint) (billingqueryapp.MonthlyTrendQuery, error) {
	windowEndMonth, err := parseOptionalYearMonth(r.WindowEndMonth)
	if err != nil {
		return billingqueryapp.MonthlyTrendQuery{}, err
	}

	return billingqueryapp.MonthlyTrendQuery{
		UserID:         userID,
		Currency:       r.Currency,
		WindowEndMonth: windowEndMonth,
	}, nil
}

func parseOptionalYearMonth(raw string) (*time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	parsed, err := time.Parse(billingYearMonthLayout, trimmed)
	if err != nil {
		return nil, err
	}

	value := time.Date(parsed.Year(), parsed.Month(), 1, 0, 0, 0, 0, time.UTC)
	return &value, nil
}
