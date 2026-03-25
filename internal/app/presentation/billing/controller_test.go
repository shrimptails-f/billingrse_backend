package billing

import (
	"business/internal/billing/application"
	billingdomain "business/internal/billing/domain"
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

func TestList_200(t *testing.T) {
	t.Parallel()

	uc := new(mockUseCase)
	uc.
		On("List", mock.Anything, mock.MatchedBy(func(query application.ListQuery) bool {
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
		Return(application.ListResult{
			Items: []application.ListItem{
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

	ctrl := newTestController(uc)
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
	ctrl := newTestController(uc)
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
	uc.On("List", mock.Anything, mock.Anything).Return(application.ListResult{}, billingdomain.ErrInvalidListQuery).Once()

	ctrl := newTestController(uc)
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
	ctrl := newTestController(uc)

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
	uc.On("List", mock.Anything, mock.Anything).Return(application.ListResult{}, errors.New("db fail")).Once()

	ctrl := newTestController(uc)
	r := listRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/billings", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Contains(t, resp.Body.String(), "internal_server_error")
	uc.AssertExpectations(t)
}

func stringPtr(value string) *string {
	return &value
}

func timePtr(value time.Time) *time.Time {
	return &value
}
