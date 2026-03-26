package billing

import (
	billingqueryapp "business/internal/billingquery/application"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

func listRouter(ctrl *Controller) *gin.Engine {
	r := gin.New()
	r.GET("/billings", func(c *gin.Context) { setUserID(c, 1) }, ctrl.List)
	return r
}

func monthDetailRouter(ctrl *Controller) *gin.Engine {
	r := gin.New()
	r.GET("/billings/summary/monthly-detail/:year_month", func(c *gin.Context) { setUserID(c, 1) }, ctrl.MonthDetail)
	return r
}

func monthlyTrendRouter(ctrl *Controller) *gin.Engine {
	r := gin.New()
	r.GET("/billings/summary/monthly-trend", func(c *gin.Context) { setUserID(c, 1) }, ctrl.MonthlyTrend)
	return r
}

func TestList_200(t *testing.T) {
	t.Parallel()

	uc := new(mockUseCase)
	uc.
		On("List", mock.Anything, mock.MatchedBy(func(query billingqueryapp.ListQuery) bool {
			if query.UserID != 1 {
				return false
			}
			if query.Q != " aws " {
				return false
			}
			if query.EmailID == nil || *query.EmailID != 101 {
				return false
			}
			if query.ExternalMessageID != "msg-101" {
				return false
			}
			if query.DateFrom == nil || !query.DateFrom.Equal(time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC)) {
				return false
			}
			if query.DateTo == nil || !query.DateTo.Equal(time.Date(2026, 3, 25, 23, 59, 59, 0, time.UTC)) {
				return false
			}
			if query.UseReceivedAtFallback == nil || *query.UseReceivedAtFallback {
				return false
			}
			if query.Limit == nil || *query.Limit != 10 {
				return false
			}
			if query.Offset == nil || *query.Offset != 20 {
				return false
			}
			return true
		})).
		Return(billingqueryapp.ListResult{
			Items: []billingqueryapp.ListItem{
				{
					EmailID:            101,
					ExternalMessageID:  "msg-101",
					VendorName:         "AWS",
					ReceivedAt:         time.Date(2026, 3, 24, 10, 0, 0, 0, time.UTC),
					BillingDate:        timePtr(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)),
					ProductNameDisplay: stringPtr("AWS Support Enterprise"),
					Amount:             12345.678,
					Currency:           "JPY",
				},
			},
			Limit:      10,
			Offset:     20,
			TotalCount: 132,
		}, nil).Once()

	ctrl := newTestController(uc, nil, nil)
	r := listRouter(ctrl)

	req := httptest.NewRequest(
		http.MethodGet,
		"/billings?q=%20aws%20&email_id=101&external_message_id=msg-101&date_from=2026-03-24T00:00:00Z&date_to=2026-03-25T23:59:59Z&use_received_at_fallback=false&limit=10&offset=20",
		nil,
	)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(t, `{
		"items": [
			{
				"email_id": 101,
				"external_message_id": "msg-101",
				"vendor_name": "AWS",
				"received_at": "2026-03-24T10:00:00Z",
				"billing_date": "2026-03-01T00:00:00Z",
				"product_name_display": "AWS Support Enterprise",
				"amount": 12345.678,
				"currency": "JPY"
			}
		],
		"limit": 10,
		"offset": 20,
		"total_count": 132
	}`, resp.Body.String())
	uc.AssertExpectations(t)
}

func TestList_400_InvalidQuerySyntax(t *testing.T) {
	t.Parallel()

	uc := new(mockUseCase)
	ctrl := newTestController(uc, nil, nil)
	r := listRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/billings?date_from=not-a-date", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "invalid_request")
}

func TestList_400_InvalidQuerySemantics(t *testing.T) {
	t.Parallel()

	uc := new(mockUseCase)
	uc.On("List", mock.Anything, mock.Anything).Return(billingqueryapp.ListResult{}, billingqueryapp.ErrInvalidListQuery).Once()

	ctrl := newTestController(uc, nil, nil)
	r := listRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/billings?date_from=2026-03-25T00:00:00Z&date_to=2026-03-24T00:00:00Z", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "invalid_request")
	uc.AssertExpectations(t)
}

func TestList_401_NoUser(t *testing.T) {
	t.Parallel()

	uc := new(mockUseCase)
	ctrl := newTestController(uc, nil, nil)

	r := gin.New()
	r.GET("/billings", ctrl.List)

	req := httptest.NewRequest(http.MethodGet, "/billings", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.Contains(t, resp.Body.String(), "unauthorized")
}

func TestList_500_Internal(t *testing.T) {
	t.Parallel()

	uc := new(mockUseCase)
	uc.On("List", mock.Anything, mock.Anything).Return(billingqueryapp.ListResult{}, errors.New("db fail")).Once()

	ctrl := newTestController(uc, nil, nil)
	r := listRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/billings", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Contains(t, resp.Body.String(), "internal_server_error")
	uc.AssertExpectations(t)
}

func TestMonthlyTrend_200(t *testing.T) {
	t.Parallel()

	monthlyTrendUC := new(mockMonthlyTrendUseCase)
	monthlyTrendUC.
		On("Get", mock.Anything, billingqueryapp.MonthlyTrendQuery{
			UserID:         1,
			Currency:       "USD",
			WindowEndMonth: timePtr(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)),
		}).
		Return(billingqueryapp.MonthlyTrendResult{
			Currency:             "USD",
			WindowStartMonth:     "2025-04",
			WindowEndMonth:       "2026-03",
			DefaultSelectedMonth: "2026-03",
			Items: []billingqueryapp.MonthlyTrendItem{
				{YearMonth: "2025-04", TotalAmount: 0, BillingCount: 0, FallbackBillingCount: 0},
				{YearMonth: "2026-03", TotalAmount: 299.97, BillingCount: 3, FallbackBillingCount: 1},
			},
		}, nil).
		Once()

	ctrl := newTestController(nil, monthlyTrendUC, nil)
	r := monthlyTrendRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/billings/summary/monthly-trend?currency=USD&window_end_month=2026-03", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(t, `{
		"currency": "USD",
		"window_start_month": "2025-04",
		"window_end_month": "2026-03",
		"default_selected_month": "2026-03",
		"items": [
			{
				"year_month": "2025-04",
				"total_amount": 0,
				"billing_count": 0,
				"fallback_billing_count": 0
			},
			{
				"year_month": "2026-03",
				"total_amount": 299.97,
				"billing_count": 3,
				"fallback_billing_count": 1
			}
		]
	}`, resp.Body.String())
	monthlyTrendUC.AssertExpectations(t)
}

func TestMonthlyTrend_400_InvalidQuerySyntax(t *testing.T) {
	t.Parallel()

	ctrl := newTestController(nil, new(mockMonthlyTrendUseCase), nil)
	r := monthlyTrendRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/billings/summary/monthly-trend?window_end_month=2026/03", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "invalid_request")
}

func TestMonthlyTrend_400_InvalidQuerySemantics(t *testing.T) {
	t.Parallel()

	monthlyTrendUC := new(mockMonthlyTrendUseCase)
	monthlyTrendUC.
		On("Get", mock.Anything, billingqueryapp.MonthlyTrendQuery{
			UserID:         1,
			Currency:       "EUR",
			WindowEndMonth: nil,
		}).
		Return(billingqueryapp.MonthlyTrendResult{}, billingqueryapp.ErrInvalidMonthlyTrendQuery).
		Once()

	ctrl := newTestController(nil, monthlyTrendUC, nil)
	r := monthlyTrendRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/billings/summary/monthly-trend?currency=EUR", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "invalid_request")
	monthlyTrendUC.AssertExpectations(t)
}

func TestMonthlyTrend_401_NoUser(t *testing.T) {
	t.Parallel()

	ctrl := newTestController(nil, new(mockMonthlyTrendUseCase), nil)

	r := gin.New()
	r.GET("/billings/summary/monthly-trend", ctrl.MonthlyTrend)

	req := httptest.NewRequest(http.MethodGet, "/billings/summary/monthly-trend", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.Contains(t, resp.Body.String(), "unauthorized")
}

func TestMonthlyTrend_500_Internal(t *testing.T) {
	t.Parallel()

	monthlyTrendUC := new(mockMonthlyTrendUseCase)
	monthlyTrendUC.
		On("Get", mock.Anything, billingqueryapp.MonthlyTrendQuery{
			UserID:         1,
			Currency:       "JPY",
			WindowEndMonth: nil,
		}).
		Return(billingqueryapp.MonthlyTrendResult{}, errors.New("db fail")).
		Once()

	ctrl := newTestController(nil, monthlyTrendUC, nil)
	r := monthlyTrendRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/billings/summary/monthly-trend?currency=JPY", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Contains(t, resp.Body.String(), "internal_server_error")
	monthlyTrendUC.AssertExpectations(t)
}

func TestMonthDetail_200(t *testing.T) {
	t.Parallel()

	monthDetailUC := new(mockMonthDetailUseCase)
	monthDetailUC.
		On("Get", mock.Anything, billingqueryapp.MonthDetailQuery{
			UserID:    1,
			YearMonth: "2026-03",
			Currency:  "",
		}).
		Return(billingqueryapp.MonthDetailResult{
			YearMonth:            "2026-03",
			Currency:             "JPY",
			TotalAmount:          182400,
			BillingCount:         12,
			FallbackBillingCount: 3,
			VendorLimit:          5,
			VendorItems: []billingqueryapp.MonthDetailVendorItem{
				{VendorName: "AWS", TotalAmount: 82000, BillingCount: 4, IsOther: false},
				{VendorName: "その他", TotalAmount: 14200, BillingCount: 2, IsOther: true},
			},
		}, nil).
		Once()

	ctrl := newTestController(nil, nil, monthDetailUC)
	r := monthDetailRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/billings/summary/monthly-detail/2026-03", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(t, `{
		"year_month": "2026-03",
		"currency": "JPY",
		"total_amount": 182400,
		"billing_count": 12,
		"fallback_billing_count": 3,
		"vendor_limit": 5,
		"vendor_items": [
			{
				"vendor_name": "AWS",
				"total_amount": 82000,
				"billing_count": 4,
				"is_other": false
			},
			{
				"vendor_name": "その他",
				"total_amount": 14200,
				"billing_count": 2,
				"is_other": true
			}
		]
	}`, resp.Body.String())
	monthDetailUC.AssertExpectations(t)
}

func TestMonthDetail_400_InvalidQuery(t *testing.T) {
	t.Parallel()

	monthDetailUC := new(mockMonthDetailUseCase)
	monthDetailUC.
		On("Get", mock.Anything, billingqueryapp.MonthDetailQuery{
			UserID:    1,
			YearMonth: "2026-13",
			Currency:  "EUR",
		}).
		Return(billingqueryapp.MonthDetailResult{}, billingqueryapp.ErrInvalidMonthDetailQuery).
		Once()

	ctrl := newTestController(nil, nil, monthDetailUC)
	r := monthDetailRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/billings/summary/monthly-detail/2026-13?currency=EUR", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "invalid_request")
	monthDetailUC.AssertExpectations(t)
}

func TestMonthDetail_401_NoUser(t *testing.T) {
	t.Parallel()

	ctrl := newTestController(nil, nil, new(mockMonthDetailUseCase))

	r := gin.New()
	r.GET("/billings/summary/monthly-detail/:year_month", ctrl.MonthDetail)

	req := httptest.NewRequest(http.MethodGet, "/billings/summary/monthly-detail/2026-03", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.Contains(t, resp.Body.String(), "unauthorized")
}

func TestMonthDetail_500_Internal(t *testing.T) {
	t.Parallel()

	monthDetailUC := new(mockMonthDetailUseCase)
	monthDetailUC.
		On("Get", mock.Anything, billingqueryapp.MonthDetailQuery{
			UserID:    1,
			YearMonth: "2026-03",
			Currency:  "JPY",
		}).
		Return(billingqueryapp.MonthDetailResult{}, errors.New("db fail")).
		Once()

	ctrl := newTestController(nil, nil, monthDetailUC)
	r := monthDetailRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/billings/summary/monthly-detail/2026-03?currency=JPY", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Contains(t, resp.Body.String(), "internal_server_error")
	monthDetailUC.AssertExpectations(t)
}

func stringPtr(value string) *string {
	return &value
}

func timePtr(value time.Time) *time.Time {
	return &value
}
