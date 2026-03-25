package manualmailworkflow

import (
	manualapp "business/internal/manualmailworkflow/application"
	"bytes"
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

func executeRouter(ctrl *Controller) *gin.Engine {
	r := gin.New()
	r.POST("/manual-mail-workflows", func(c *gin.Context) { setUserID(c, 1) }, ctrl.Execute)
	return r
}

func listRouter(ctrl *Controller) *gin.Engine {
	r := gin.New()
	r.GET("/manual-mail-workflows", func(c *gin.Context) { setUserID(c, 1) }, ctrl.List)
	return r
}

func TestExecute_202(t *testing.T) {
	t.Parallel()

	uc := new(mockUseCase)
	uc.On("Start", mock.Anything, manualapp.Command{
		UserID:       1,
		ConnectionID: 12,
		Condition: manualapp.FetchCondition{
			LabelName: "billing",
			Since:     time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
			Until:     time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
		},
	}).Return(manualapp.StartResult{
		WorkflowID: "wf-123",
		Status:     manualapp.WorkflowStatusQueued,
	}, nil).Once()

	ctrl := newTestController(uc, nil)
	r := executeRouter(ctrl)

	body := []byte(`{"connection_id":12,"label_name":"billing","since":"2026-03-24T00:00:00Z","until":"2026-03-25T00:00:00Z"}`)
	req := httptest.NewRequest(http.MethodPost, "/manual-mail-workflows", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusAccepted, resp.Code)
	assert.JSONEq(t, `{
		"message": "メール取得ワークフローを受け付けました。",
		"workflow_id": "wf-123",
		"status": "queued"
	}`, resp.Body.String())
	uc.AssertExpectations(t)
}

func TestExecute_400_InvalidRequest(t *testing.T) {
	t.Parallel()

	uc := new(mockUseCase)
	ctrl := newTestController(uc, nil)
	r := executeRouter(ctrl)

	body := []byte(`{"connection_id":12,"label_name":"","since":"2026-03-24T00:00:00Z"}`)
	req := httptest.NewRequest(http.MethodPost, "/manual-mail-workflows", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "invalid_request")
}

func TestExecute_401_NoUser(t *testing.T) {
	t.Parallel()

	uc := new(mockUseCase)
	ctrl := newTestController(uc, nil)

	r := gin.New()
	r.POST("/manual-mail-workflows", ctrl.Execute)

	body := []byte(`{"connection_id":12,"label_name":"billing","since":"2026-03-24T00:00:00Z","until":"2026-03-25T00:00:00Z"}`)
	req := httptest.NewRequest(http.MethodPost, "/manual-mail-workflows", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.Contains(t, resp.Body.String(), "unauthorized")
}

func TestExecute_500_StartFailure(t *testing.T) {
	t.Parallel()

	uc := new(mockUseCase)
	uc.On("Start", mock.Anything, mock.Anything).Return(manualapp.StartResult{}, errors.New("dispatch failed")).Once()

	ctrl := newTestController(uc, nil)
	r := executeRouter(ctrl)

	body := []byte(`{"connection_id":12,"label_name":"billing","since":"2026-03-24T00:00:00Z","until":"2026-03-25T00:00:00Z"}`)
	req := httptest.NewRequest(http.MethodPost, "/manual-mail-workflows", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Contains(t, resp.Body.String(), "internal_server_error")
	uc.AssertExpectations(t)
}

func TestList_200(t *testing.T) {
	t.Parallel()

	listUC := new(mockListUseCase)
	listUC.On("List", mock.Anything, manualapp.ListQuery{
		UserID: 1,
	}).Return(manualapp.ListResult{
		Items: []manualapp.WorkflowHistoryListItem{
			{
				WorkflowID:        "wf-123",
				Provider:          "gmail",
				AccountIdentifier: "billing@example.com",
				LabelName:         "billing",
				Since:             time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
				Until:             time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
				Status:            manualapp.WorkflowStatusPartialSuccess,
				QueuedAt:          time.Date(2026, 3, 25, 17, 0, 0, 0, time.UTC),
				FinishedAt:        timePtr(time.Date(2026, 3, 25, 17, 0, 12, 0, time.UTC)),
				Fetch: manualapp.StageSummaryView{
					SuccessCount:          14,
					BusinessFailureCount:  0,
					TechnicalFailureCount: 1,
					Failures: []manualapp.StageFailureView{
						{
							ExternalMessageID: stringPtr("msg-1"),
							ReasonCode:        "fetch_detail_failed",
							Message:           "メールの取得に失敗しました。",
							CreatedAt:         time.Date(2026, 3, 25, 17, 0, 2, 0, time.UTC),
						},
					},
				},
				Analysis:           manualapp.StageSummaryView{Failures: []manualapp.StageFailureView{}},
				VendorResolution:   manualapp.StageSummaryView{Failures: []manualapp.StageFailureView{}},
				BillingEligibility: manualapp.StageSummaryView{Failures: []manualapp.StageFailureView{}},
				Billing:            manualapp.StageSummaryView{Failures: []manualapp.StageFailureView{}},
			},
		},
		TotalCount: 57,
	}, nil).Once()

	ctrl := newTestController(nil, listUC)
	r := listRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/manual-mail-workflows", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.JSONEq(t, `{
		"items": [
			{
				"workflow_id": "wf-123",
				"provider": "gmail",
				"account_identifier": "billing@example.com",
				"label_name": "billing",
				"since": "2026-03-24T00:00:00Z",
				"until": "2026-03-25T00:00:00Z",
				"status": "partial_success",
				"current_stage": null,
				"queued_at": "2026-03-25T17:00:00Z",
				"finished_at": "2026-03-25T17:00:12Z",
				"fetch": {
					"success_count": 14,
					"business_failure_count": 0,
					"technical_failure_count": 1,
					"failures": [
						{
							"external_message_id": "msg-1",
							"reason_code": "fetch_detail_failed",
							"message": "メールの取得に失敗しました。",
							"created_at": "2026-03-25T17:00:02Z"
						}
					]
				},
				"analysis": {
					"success_count": 0,
					"business_failure_count": 0,
					"technical_failure_count": 0,
					"failures": []
				},
				"vendor_resolution": {
					"success_count": 0,
					"business_failure_count": 0,
					"technical_failure_count": 0,
					"failures": []
				},
				"billing_eligibility": {
					"success_count": 0,
					"business_failure_count": 0,
					"technical_failure_count": 0,
					"failures": []
				},
				"billing": {
					"success_count": 0,
					"business_failure_count": 0,
					"technical_failure_count": 0,
					"failures": []
				}
			}
		],
		"total_count": 57
	}`, resp.Body.String())
	listUC.AssertExpectations(t)
}

func TestList_200_WithQuery(t *testing.T) {
	t.Parallel()

	listUC := new(mockListUseCase)
	status := manualapp.WorkflowStatusPartialSuccess
	listUC.On("List", mock.Anything, manualapp.ListQuery{
		UserID:    1,
		Limit:     50,
		Offset:    10,
		Status:    &status,
		HasLimit:  true,
		HasOffset: true,
	}).Return(manualapp.ListResult{
		Items:      []manualapp.WorkflowHistoryListItem{},
		TotalCount: 0,
	}, nil).Once()

	ctrl := newTestController(nil, listUC)
	r := listRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/manual-mail-workflows?limit=50&offset=10&status=partial_success", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	listUC.AssertExpectations(t)
}

func TestList_400_InvalidRequest(t *testing.T) {
	t.Parallel()

	listUC := new(mockListUseCase)
	ctrl := newTestController(nil, listUC)
	r := listRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/manual-mail-workflows?limit=abc", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "invalid_request")
}

func TestList_400_InvalidQueryFromUseCase(t *testing.T) {
	t.Parallel()

	listUC := new(mockListUseCase)
	listUC.On("List", mock.Anything, manualapp.ListQuery{
		UserID:    1,
		Limit:     0,
		Offset:    0,
		HasLimit:  true,
		HasOffset: true,
	}).Return(manualapp.ListResult{}, manualapp.ErrInvalidListQuery).Once()

	ctrl := newTestController(nil, listUC)
	r := listRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/manual-mail-workflows?limit=0&offset=0", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "invalid_request")
	listUC.AssertExpectations(t)
}

func TestList_401_NoUser(t *testing.T) {
	t.Parallel()

	listUC := new(mockListUseCase)
	ctrl := newTestController(nil, listUC)

	r := gin.New()
	r.GET("/manual-mail-workflows", ctrl.List)

	req := httptest.NewRequest(http.MethodGet, "/manual-mail-workflows", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.Contains(t, resp.Body.String(), "unauthorized")
}

func TestList_500_Internal(t *testing.T) {
	t.Parallel()

	listUC := new(mockListUseCase)
	listUC.On("List", mock.Anything, manualapp.ListQuery{
		UserID: 1,
	}).Return(manualapp.ListResult{}, errors.New("db failed")).Once()

	ctrl := newTestController(nil, listUC)
	r := listRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/manual-mail-workflows", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Contains(t, resp.Body.String(), "internal_server_error")
	listUC.AssertExpectations(t)
}

func timePtr(value time.Time) *time.Time {
	return &value
}

func stringPtr(value string) *string {
	return &value
}
