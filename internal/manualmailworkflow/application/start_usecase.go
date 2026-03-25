package application

import (
	"business/internal/library/logger"
	"business/internal/library/timewrapper"
	"context"
	"errors"
	"fmt"

	"github.com/aidarkhanov/nanoid/v2"
)

// StartResult is the accepted response payload for the manual mail workflow.
type StartResult struct {
	WorkflowID string
	Status     string
}

// DispatchJob is the background execution payload for a workflow run.
type DispatchJob struct {
	HistoryID    uint64
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
	repository WorkflowStatusRepository
	clock      timewrapper.ClockInterface
	log        logger.Interface
}

// NewStartUseCase creates a start use case for background workflow acceptance.
func NewStartUseCase(
	dispatcher WorkflowDispatcher,
	repository WorkflowStatusRepository,
	clock timewrapper.ClockInterface,
	log logger.Interface,
) StartUseCase {
	if clock == nil {
		clock = timewrapper.NewClock()
	}
	if log == nil {
		log = logger.NewNop()
	}

	return &startUseCase{
		dispatcher: dispatcher,
		repository: repository,
		clock:      clock,
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
	if uc.repository == nil {
		return StartResult{}, errors.New("workflow_status_repository is not configured")
	}

	reqLog := uc.log
	if withContext, err := uc.log.WithContext(ctx); err == nil {
		reqLog = withContext
	}

	cmd.Condition = cmd.Condition.Normalize()
	workflowID, err := newWorkflowID()
	if err != nil {
		return StartResult{}, fmt.Errorf("failed to generate workflow id: %w", err)
	}
	queuedAt := uc.clock.Now().UTC()

	historyRef, err := uc.repository.CreateQueued(ctx, QueuedWorkflowHistory{
		WorkflowID:   workflowID,
		UserID:       cmd.UserID,
		ConnectionID: cmd.ConnectionID,
		LabelName:    cmd.Condition.LabelName,
		SinceAt:      cmd.Condition.Since,
		UntilAt:      cmd.Condition.Until,
		QueuedAt:     queuedAt,
	})
	if err != nil {
		return StartResult{}, err
	}

	result := StartResult{
		WorkflowID: historyRef.WorkflowID,
		Status:     WorkflowStatusQueued,
	}

	if err := uc.dispatcher.Dispatch(ctx, DispatchJob{
		HistoryID:    historyRef.HistoryID,
		WorkflowID:   result.WorkflowID,
		UserID:       cmd.UserID,
		ConnectionID: cmd.ConnectionID,
		Condition:    cmd.Condition,
	}); err != nil {
		if failErr := uc.repository.Fail(ctx, historyRef.HistoryID, "", uc.clock.Now().UTC()); failErr != nil {
			reqLog.Error("manual_mail_workflow_dispatch_failed_to_mark_history",
				logger.UserID(cmd.UserID),
				logger.Uint("connection_id", cmd.ConnectionID),
				logger.String("workflow_id", result.WorkflowID),
				logger.Err(failErr),
			)
		}
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

func newWorkflowID() (string, error) {
	return nanoid.GenerateString("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ", 26)
}
