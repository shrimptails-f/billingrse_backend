package infrastructure

import (
	"business/internal/library/logger"
	manualapp "business/internal/manualmailworkflow/application"
	"context"
	"errors"
)

// InProcessWorkflowDispatcher runs the manual mail workflow in a background goroutine.
type InProcessWorkflowDispatcher struct {
	runner manualapp.UseCase
	log    logger.Interface
}

// NewInProcessWorkflowDispatcher creates an in-process dispatcher for the workflow runner.
func NewInProcessWorkflowDispatcher(runner manualapp.UseCase, log logger.Interface) *InProcessWorkflowDispatcher {
	if log == nil {
		log = logger.NewNop()
	}

	return &InProcessWorkflowDispatcher{
		runner: runner,
		log:    log.With(logger.Component("manual_mail_workflow_dispatcher")),
	}
}

// Dispatch starts background execution and returns immediately after scheduling it.
func (d *InProcessWorkflowDispatcher) Dispatch(ctx context.Context, job manualapp.DispatchJob) error {
	if ctx == nil {
		return logger.ErrNilContext
	}
	if d.runner == nil {
		return errors.New("manual mail workflow runner is not configured")
	}

	go d.run(ctx, job)
	return nil
}

func (d *InProcessWorkflowDispatcher) run(requestCtx context.Context, job manualapp.DispatchJob) {
	bgCtx := context.Background()

	if requestID, ok := logger.RequestIDFromContext(requestCtx); ok {
		if next, err := logger.ContextWithRequestID(bgCtx, requestID); err == nil {
			bgCtx = next
		}
	}
	if next, err := logger.ContextWithJobID(bgCtx, job.WorkflowID); err == nil {
		bgCtx = next
	}
	if next, err := logger.ContextWithUserID(bgCtx, job.UserID); err == nil {
		bgCtx = next
	}

	reqLog := d.log
	if withContext, err := d.log.WithContext(bgCtx); err == nil {
		reqLog = withContext
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			reqLog.Error("manual_mail_workflow_panicked",
				logger.Recovered(recovered),
				logger.StackTrace(),
				logger.Uint("connection_id", job.ConnectionID),
			)
		}
	}()

	reqLog.Info("manual_mail_workflow_started",
		logger.String("workflow_id", job.WorkflowID),
		logger.Uint("connection_id", job.ConnectionID),
	)

	if _, err := d.runner.Execute(bgCtx, job); err != nil {
		reqLog.Error("manual_mail_workflow_failed",
			logger.String("workflow_id", job.WorkflowID),
			logger.Uint("connection_id", job.ConnectionID),
			logger.Err(err),
		)
	}
}
