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

	ctrl := newTestController(uc)
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
	ctrl := newTestController(uc)
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
	ctrl := newTestController(uc)

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

	ctrl := newTestController(uc)
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
