package infrastructure

import (
	"business/internal/library/logger"
	manualapp "business/internal/manualmailworkflow/application"
	"context"
	"errors"
	"testing"
	"time"
)

type stubWorkflowRunner struct {
	execute func(ctx context.Context, job manualapp.DispatchJob) (manualapp.Result, error)
}

func (s *stubWorkflowRunner) Execute(ctx context.Context, job manualapp.DispatchJob) (manualapp.Result, error) {
	return s.execute(ctx, job)
}

func TestInProcessWorkflowDispatcher_Dispatch_RunsInBackgroundWithContextFields(t *testing.T) {
	t.Parallel()

	done := make(chan struct{})

	requestCtx, err := logger.ContextWithRequestID(context.Background(), "req-123")
	if err != nil {
		t.Fatalf("ContextWithRequestID returned error: %v", err)
	}

	dispatcher := NewInProcessWorkflowDispatcher(&stubWorkflowRunner{
		execute: func(ctx context.Context, job manualapp.DispatchJob) (manualapp.Result, error) {
			if job.HistoryID != 44 || job.WorkflowID != "wf-123" {
				t.Fatalf("unexpected dispatch job: %+v", job)
			}
			if job.UserID != 9 || job.ConnectionID != 21 {
				t.Fatalf("unexpected command: %+v", job)
			}

			requestID, ok := logger.RequestIDFromContext(ctx)
			if !ok || requestID != "req-123" {
				t.Fatalf("expected request id to be propagated, got %q %v", requestID, ok)
			}

			jobID, ok := logger.JobIDFromContext(ctx)
			if !ok || jobID != "wf-123" {
				t.Fatalf("expected job id to be propagated, got %q %v", jobID, ok)
			}

			userID, ok := logger.UserIDFromContext(ctx)
			if !ok || userID != 9 {
				t.Fatalf("expected user id to be propagated, got %d %v", userID, ok)
			}

			close(done)
			return manualapp.Result{}, nil
		},
	}, logger.NewNop())

	if err := dispatcher.Dispatch(requestCtx, manualapp.DispatchJob{
		HistoryID:    44,
		WorkflowID:   "wf-123",
		UserID:       9,
		ConnectionID: 21,
		Condition: manualapp.FetchCondition{
			LabelName: "billing",
			Since:     time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
			Until:     time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
		},
	}); err != nil {
		t.Fatalf("Dispatch returned error: %v", err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("background runner did not complete")
	}
}

func TestInProcessWorkflowDispatcher_Dispatch_RejectsNilContext(t *testing.T) {
	t.Parallel()

	dispatcher := NewInProcessWorkflowDispatcher(&stubWorkflowRunner{
		execute: func(ctx context.Context, job manualapp.DispatchJob) (manualapp.Result, error) {
			return manualapp.Result{}, nil
		},
	}, logger.NewNop())

	err := dispatcher.Dispatch(nil, manualapp.DispatchJob{})
	if !errors.Is(err, logger.ErrNilContext) {
		t.Fatalf("expected ErrNilContext, got %v", err)
	}
}
