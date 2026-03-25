package manualmailworkflow

import (
	"business/internal/app/httpresponse"
	"business/internal/library/logger"
	manualapp "business/internal/manualmailworkflow/application"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Controller handles manual mail workflow HTTP requests.
type Controller struct {
	startUseCase manualapp.StartUseCase
	listUseCase  manualapp.ListUseCase
	log          logger.Interface
}

// NewController creates a new Controller.
func NewController(
	startUseCase manualapp.StartUseCase,
	listUseCase manualapp.ListUseCase,
	log logger.Interface,
) *Controller {
	if log == nil {
		log = logger.NewNop()
	}

	return &Controller{
		startUseCase: startUseCase,
		listUseCase:  listUseCase,
		log:          log.With(logger.Component("manual_mail_workflow_controller")),
	}
}

type executeRequest struct {
	ConnectionID uint      `json:"connection_id" binding:"required"`
	LabelName    string    `json:"label_name" binding:"required"`
	Since        time.Time `json:"since" binding:"required"`
	Until        time.Time `json:"until" binding:"required"`
}

type executeAcceptedResponse struct {
	Message    string `json:"message"`
	WorkflowID string `json:"workflow_id"`
	Status     string `json:"status"`
}

type listResponse struct {
	Items      []workflowHistoryItemResponse `json:"items"`
	TotalCount int64                         `json:"total_count"`
}

type workflowHistoryItemResponse struct {
	WorkflowID         string               `json:"workflow_id"`
	Provider           string               `json:"provider"`
	AccountIdentifier  string               `json:"account_identifier"`
	LabelName          string               `json:"label_name"`
	Since              time.Time            `json:"since"`
	Until              time.Time            `json:"until"`
	Status             string               `json:"status"`
	CurrentStage       *string              `json:"current_stage"`
	QueuedAt           time.Time            `json:"queued_at"`
	FinishedAt         *time.Time           `json:"finished_at"`
	Fetch              stageSummaryResponse `json:"fetch"`
	Analysis           stageSummaryResponse `json:"analysis"`
	VendorResolution   stageSummaryResponse `json:"vendor_resolution"`
	BillingEligibility stageSummaryResponse `json:"billing_eligibility"`
	Billing            stageSummaryResponse `json:"billing"`
}

type stageSummaryResponse struct {
	SuccessCount          int                    `json:"success_count"`
	BusinessFailureCount  int                    `json:"business_failure_count"`
	TechnicalFailureCount int                    `json:"technical_failure_count"`
	Failures              []stageFailureResponse `json:"failures"`
}

type stageFailureResponse struct {
	ExternalMessageID *string   `json:"external_message_id"`
	ReasonCode        string    `json:"reason_code"`
	Message           string    `json:"message"`
	CreatedAt         time.Time `json:"created_at"`
}

// Execute handles POST /api/v1/manual-mail-workflows.
func (ctrl *Controller) Execute(c *gin.Context) {
	reqLog := ctrl.log
	if l, err := ctrl.log.WithContext(c.Request.Context()); err == nil {
		reqLog = l
	}

	uid, ok := currentUserID(c)
	if !ok {
		return
	}

	var req executeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresponse.WriteInvalidRequest(c)
		return
	}

	if ctrl.startUseCase == nil {
		reqLog.Error("manual_mail_workflow_start_usecase_not_configured")
		httpresponse.WriteInternalServerError(c)
		return
	}

	result, err := ctrl.startUseCase.Start(c.Request.Context(), manualapp.Command{
		UserID:       uid,
		ConnectionID: req.ConnectionID,
		Condition: manualapp.FetchCondition{
			LabelName: req.LabelName,
			Since:     req.Since,
			Until:     req.Until,
		},
	})
	if err != nil {
		ctrl.writeStartError(c, reqLog, uid, req.ConnectionID, err)
		return
	}

	c.JSON(http.StatusAccepted, executeAcceptedResponse{
		Message:    "メール取得ワークフローを受け付けました。",
		WorkflowID: result.WorkflowID,
		Status:     result.Status,
	})
}

// List handles GET /api/v1/manual-mail-workflows.
func (ctrl *Controller) List(c *gin.Context) {
	reqLog := ctrl.log
	if l, err := ctrl.log.WithContext(c.Request.Context()); err == nil {
		reqLog = l
	}

	uid, ok := currentUserID(c)
	if !ok {
		return
	}
	if ctrl.listUseCase == nil {
		reqLog.Error("manual_mail_workflow_list_usecase_not_configured")
		httpresponse.WriteInternalServerError(c)
		return
	}

	query, err := buildListQuery(c, uid)
	if err != nil {
		httpresponse.WriteInvalidRequest(c)
		return
	}

	result, err := ctrl.listUseCase.List(c.Request.Context(), query)
	if err != nil {
		switch {
		case errors.Is(err, manualapp.ErrInvalidListQuery):
			httpresponse.WriteInvalidRequest(c)
		default:
			reqLog.Error("manual_mail_workflow_list_failed",
				logger.UserID(uid),
				logger.Err(err),
			)
			httpresponse.WriteInternalServerError(c)
		}
		return
	}

	items := make([]workflowHistoryItemResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toWorkflowHistoryItemResponse(item))
	}

	c.JSON(http.StatusOK, listResponse{
		Items:      items,
		TotalCount: result.TotalCount,
	})
}

func (ctrl *Controller) writeStartError(c *gin.Context, reqLog logger.Interface, userID, connectionID uint, err error) {
	switch {
	case errors.Is(err, manualapp.ErrInvalidCommand), errors.Is(err, manualapp.ErrFetchConditionInvalid):
		httpresponse.WriteInvalidRequest(c)
	default:
		reqLog.Error("manual_mail_workflow_start_failed",
			logger.UserID(userID),
			logger.Uint("connection_id", connectionID),
			logger.Err(err),
		)
		httpresponse.WriteInternalServerError(c)
	}
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

func buildListQuery(c *gin.Context, userID uint) (manualapp.ListQuery, error) {
	query := manualapp.ListQuery{
		UserID: userID,
	}

	if rawLimit, exists := c.GetQuery("limit"); exists {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil {
			return manualapp.ListQuery{}, err
		}
		query.Limit = limit
		query.HasLimit = true
	}

	if rawOffset, exists := c.GetQuery("offset"); exists {
		offset, err := strconv.Atoi(rawOffset)
		if err != nil {
			return manualapp.ListQuery{}, err
		}
		query.Offset = offset
		query.HasOffset = true
	}

	if rawStatus, exists := c.GetQuery("status"); exists {
		query.Status = &rawStatus
	}

	return query, nil
}

func toWorkflowHistoryItemResponse(item manualapp.WorkflowHistoryListItem) workflowHistoryItemResponse {
	return workflowHistoryItemResponse{
		WorkflowID:         item.WorkflowID,
		Provider:           item.Provider,
		AccountIdentifier:  item.AccountIdentifier,
		LabelName:          item.LabelName,
		Since:              item.Since,
		Until:              item.Until,
		Status:             item.Status,
		CurrentStage:       cloneOptionalString(item.CurrentStage),
		QueuedAt:           item.QueuedAt,
		FinishedAt:         cloneOptionalTime(item.FinishedAt),
		Fetch:              toStageSummaryResponse(item.Fetch),
		Analysis:           toStageSummaryResponse(item.Analysis),
		VendorResolution:   toStageSummaryResponse(item.VendorResolution),
		BillingEligibility: toStageSummaryResponse(item.BillingEligibility),
		Billing:            toStageSummaryResponse(item.Billing),
	}
}

func toStageSummaryResponse(summary manualapp.StageSummaryView) stageSummaryResponse {
	failures := make([]stageFailureResponse, 0, len(summary.Failures))
	for _, failure := range summary.Failures {
		failures = append(failures, stageFailureResponse{
			ExternalMessageID: cloneOptionalString(failure.ExternalMessageID),
			ReasonCode:        failure.ReasonCode,
			Message:           failure.Message,
			CreatedAt:         failure.CreatedAt,
		})
	}

	return stageSummaryResponse{
		SuccessCount:          summary.SuccessCount,
		BusinessFailureCount:  summary.BusinessFailureCount,
		TechnicalFailureCount: summary.TechnicalFailureCount,
		Failures:              failures,
	}
}

func cloneOptionalString(value *string) *string {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}

func cloneOptionalTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	cloned := value.UTC()
	return &cloned
}
