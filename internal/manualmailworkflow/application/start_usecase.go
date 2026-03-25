package application

import (
	"business/internal/library/logger"
	"context"
	"errors"

	"github.com/google/uuid"
)

const (
	// WorkflowStatusQueued indicates the workflow has been accepted for background execution.
	WorkflowStatusQueued = "queued"
)

// StartResult is the accepted response payload for the manual mail workflow.
type StartResult struct {
	WorkflowID string
	Status     string
}

// DispatchJob is the background execution payload for a workflow run.
type DispatchJob struct {
	WorkflowID   string
	UserID       uint
	ConnectionID uint
	Condition    FetchCondition
}

// WorkflowDispatcher dispatches the workflow for background execution.
type WorkflowDispatcher interface {
	Dispatch(ctx context.Context, job DispatchJob) error
}

// StartUseCase validates and accepts a manual mail workflow request.
type StartUseCase interface {
	Start(ctx context.Context, cmd Command) (StartResult, error)
}

type startUseCase struct {
	dispatcher WorkflowDispatcher
	log        logger.Interface
}

// NewStartUseCase creates a start use case for background workflow acceptance.
func NewStartUseCase(dispatcher WorkflowDispatcher, log logger.Interface) StartUseCase {
	if log == nil {
		log = logger.NewNop()
	}

	return &startUseCase{
		dispatcher: dispatcher,
		log:        log.With(logger.Component("manual_mail_workflow_start_usecase")),
	}
}

// Start validates the request and dispatches the workflow for background execution.
func (uc *startUseCase) Start(ctx context.Context, cmd Command) (StartResult, error) {
	if ctx == nil {
		return StartResult{}, logger.ErrNilContext
	}
	if err := validateCommand(cmd); err != nil {
		return StartResult{}, err
	}
	if uc.dispatcher == nil {
		return StartResult{}, errors.New("workflow_dispatcher is not configured")
	}

	reqLog := uc.log
	if withContext, err := uc.log.WithContext(ctx); err == nil {
		reqLog = withContext
	}

	cmd.Condition = cmd.Condition.Normalize()
	result := StartResult{
		WorkflowID: uuid.NewString(),
		Status:     WorkflowStatusQueued,
	}

	if err := uc.dispatcher.Dispatch(ctx, DispatchJob{
		WorkflowID:   result.WorkflowID,
		UserID:       cmd.UserID,
		ConnectionID: cmd.ConnectionID,
		Condition:    cmd.Condition,
	}); err != nil {
		return StartResult{}, err
	}

	reqLog.Info("manual_mail_workflow_accepted",
		logger.UserID(cmd.UserID),
		logger.Uint("connection_id", cmd.ConnectionID),
		logger.String("workflow_id", result.WorkflowID),
		logger.String("status", result.Status),
	)

	return result, nil
}
