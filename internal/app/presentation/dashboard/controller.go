package dashboard

import (
	"business/internal/app/httpresponse"
	dashboardqueryapp "business/internal/dashboardquery/application"
	"business/internal/library/logger"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Controller handles dashboard summary HTTP requests.
type Controller struct {
	usecase dashboardqueryapp.SummaryUseCaseInterface
	log     logger.Interface
}

type summaryResponse struct {
	CurrentMonthAnalysisSuccessCount int `json:"current_month_analysis_success_count"`
	TotalSavedBillingCount           int `json:"total_saved_billing_count"`
	CurrentMonthFallbackBillingCount int `json:"current_month_fallback_billing_count"`
}

// NewController creates a dashboard summary controller.
func NewController(usecase dashboardqueryapp.SummaryUseCaseInterface, log logger.Interface) *Controller {
	if log == nil {
		log = logger.NewNop()
	}

	return &Controller{
		usecase: usecase,
		log:     log.With(logger.Component("dashboard_summary_controller")),
	}
}

// Summary handles GET /api/v1/dashboard/summary.
func (ctrl *Controller) Summary(c *gin.Context) {
	reqLog := ctrl.log
	if withContext, err := ctrl.log.WithContext(c.Request.Context()); err == nil {
		reqLog = withContext
	}

	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	if ctrl.usecase == nil {
		reqLog.Error("dashboard_summary_usecase_not_configured",
			logger.UserID(userID),
		)
		httpresponse.WriteInternalServerError(c)
		return
	}

	result, err := ctrl.usecase.Get(c.Request.Context(), dashboardqueryapp.SummaryQuery{
		UserID: userID,
	})
	if err != nil {
		reqLog.Error("dashboard_summary_failed",
			logger.UserID(userID),
			logger.Err(err),
		)
		httpresponse.WriteInternalServerError(c)
		return
	}

	c.JSON(http.StatusOK, summaryResponse{
		CurrentMonthAnalysisSuccessCount: result.CurrentMonthAnalysisSuccessCount,
		TotalSavedBillingCount:           result.TotalSavedBillingCount,
		CurrentMonthFallbackBillingCount: result.CurrentMonthFallbackBillingCount,
	})
}

func currentUserID(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("userID")
	if !exists {
		httpresponse.WriteError(c, http.StatusUnauthorized, "unauthorized", "認証が必要です。")
		return 0, false
	}

	uid, ok := userID.(uint)
	if !ok {
		httpresponse.WriteInternalServerError(c)
		return 0, false
	}

	return uid, true
}
