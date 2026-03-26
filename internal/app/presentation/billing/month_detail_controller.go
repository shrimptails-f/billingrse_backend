package billing

import (
	"business/internal/app/httpresponse"
	billingqueryapp "business/internal/billingquery/application"
	"business/internal/library/logger"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type monthDetailQueryRequest struct {
	Currency string `form:"currency"`
}

type monthDetailResponse struct {
	YearMonth            string                        `json:"year_month"`
	Currency             string                        `json:"currency"`
	TotalAmount          float64                       `json:"total_amount"`
	BillingCount         int                           `json:"billing_count"`
	FallbackBillingCount int                           `json:"fallback_billing_count"`
	VendorLimit          int                           `json:"vendor_limit"`
	VendorItems          []monthDetailVendorItemRecord `json:"vendor_items"`
}

type monthDetailVendorItemRecord struct {
	VendorName   string  `json:"vendor_name"`
	TotalAmount  float64 `json:"total_amount"`
	BillingCount int     `json:"billing_count"`
	IsOther      bool    `json:"is_other"`
}

// MonthDetail handles GET /api/v1/billings/summary/monthly-detail/:year_month.
func (ctrl *Controller) MonthDetail(c *gin.Context) {
	reqLog := ctrl.log
	if withContext, err := ctrl.log.WithContext(c.Request.Context()); err == nil {
		reqLog = withContext
	}

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	if ctrl.monthDetailUseCase == nil {
		reqLog.Error("billing_month_detail_usecase_not_configured",
			logger.UserID(userID),
		)
		httpresponse.WriteInternalServerError(c)
		return
	}

	var req monthDetailQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		httpresponse.WriteInvalidRequest(c)
		return
	}

	result, err := ctrl.monthDetailUseCase.Get(c.Request.Context(), billingqueryapp.MonthDetailQuery{
		UserID:    userID,
		YearMonth: c.Param("year_month"),
		Currency:  req.Currency,
	})
	if err != nil {
		if errors.Is(err, billingqueryapp.ErrInvalidMonthDetailQuery) {
			httpresponse.WriteInvalidRequest(c)
			return
		}

		reqLog.Error("billing_month_detail_failed",
			logger.UserID(userID),
			logger.String("year_month", c.Param("year_month")),
			logger.String("currency", req.Currency),
			logger.Err(err),
		)
		httpresponse.WriteInternalServerError(c)
		return
	}

	items := make([]monthDetailVendorItemRecord, 0, len(result.VendorItems))
	for _, item := range result.VendorItems {
		items = append(items, monthDetailVendorItemRecord{
			VendorName:   item.VendorName,
			TotalAmount:  item.TotalAmount,
			BillingCount: item.BillingCount,
			IsOther:      item.IsOther,
		})
	}

	c.JSON(http.StatusOK, monthDetailResponse{
		YearMonth:            result.YearMonth,
		Currency:             result.Currency,
		TotalAmount:          result.TotalAmount,
		BillingCount:         result.BillingCount,
		FallbackBillingCount: result.FallbackBillingCount,
		VendorLimit:          result.VendorLimit,
		VendorItems:          items,
	})
}
