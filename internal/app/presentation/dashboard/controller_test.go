package dashboard

import (
	dashboardqueryapp "business/internal/dashboardquery/application"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setUserID(c *gin.Context, uid uint) {
	c.Set("userID", uid)
}

func summaryRouter(ctrl *Controller) *gin.Engine {
	r := gin.New()
	r.GET("/dashboard/summary", func(c *gin.Context) { setUserID(c, 1) }, ctrl.Summary)
	return r
}

func TestSummary_200(t *testing.T) {
	t.Parallel()

	uc := new(mockSummaryUseCase)
	uc.
		On("Get", mock.Anything, dashboardqueryapp.SummaryQuery{UserID: 1}).
		Return(dashboardqueryapp.SummaryResult{
			CurrentMonthAnalysisSuccessCount: 1280,
			TotalSavedBillingCount:           842,
			CurrentMonthFallbackBillingCount: 73,
		}, nil).
		Once()

	ctrl := newTestController(uc)
	r := summaryRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/dashboard/summary", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(t, `{
		"current_month_analysis_success_count": 1280,
		"total_saved_billing_count": 842,
		"current_month_fallback_billing_count": 73
	}`, resp.Body.String())
	uc.AssertExpectations(t)
}

func TestSummary_401_NoUser(t *testing.T) {
	t.Parallel()

	ctrl := newTestController(new(mockSummaryUseCase))

	r := gin.New()
	r.GET("/dashboard/summary", ctrl.Summary)

	req := httptest.NewRequest(http.MethodGet, "/dashboard/summary", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.Contains(t, resp.Body.String(), "unauthorized")
}

func TestSummary_500_Internal(t *testing.T) {
	t.Parallel()

	uc := new(mockSummaryUseCase)
	uc.
		On("Get", mock.Anything, dashboardqueryapp.SummaryQuery{UserID: 1}).
		Return(dashboardqueryapp.SummaryResult{}, errors.New("db fail")).
		Once()

	ctrl := newTestController(uc)
	r := summaryRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/dashboard/summary", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Contains(t, resp.Body.String(), "internal_server_error")
	uc.AssertExpectations(t)
}
