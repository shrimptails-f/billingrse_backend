package manualmailworkflow

import (
	"business/internal/app/httpresponse"
	"business/internal/library/logger"
	manualapp "business/internal/manualmailworkflow/application"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Controller handles manual mail workflow HTTP requests.
type Controller struct {
	usecase manualapp.StartUseCase
	log     logger.Interface
}

// NewController creates a new Controller.
func NewController(usecase manualapp.StartUseCase, log logger.Interface) *Controller {
	if log == nil {
		log = logger.NewNop()
	}

	return &Controller{
		usecase: usecase,
		log:     log.With(logger.Component("manual_mail_workflow_controller")),
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

	result, err := ctrl.usecase.Start(c.Request.Context(), manualapp.Command{
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
