package application

import (
	"business/internal/library/logger"
	"context"
	"errors"
	"testing"
	"time"
)

type stubWorkflowDispatcher struct {
	dispatch func(ctx context.Context, job DispatchJob) error
}

func (s *stubWorkflowDispatcher) Dispatch(ctx context.Context, job DispatchJob) error {
	return s.dispatch(ctx, job)
}

func TestStartUseCase_Start_DispatchesValidatedCommand(t *testing.T) {
	t.Parallel()

	dispatchCalls := 0
	uc := NewStartUseCase(&stubWorkflowDispatcher{
		dispatch: func(ctx context.Context, job DispatchJob) error {
			dispatchCalls++
			if job.UserID != 7 || job.ConnectionID != 12 {
				t.Fatalf("unexpected dispatch job: %+v", job)
			}
			if job.Condition.LabelName != "billing" {
				t.Fatalf("expected normalized label name, got %+v", job.Condition)
			}
			if job.WorkflowID == "" {
				t.Fatal("expected workflow id to be generated")
			}
			return nil
		},
	}, logger.NewNop())

	result, err := uc.Start(context.Background(), Command{
		UserID:       7,
		ConnectionID: 12,
		Condition: FetchCondition{
			LabelName: " billing ",
			Since:     time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
			Until:     time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
		},
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if dispatchCalls != 1 {
		t.Fatalf("expected 1 dispatch call, got %d", dispatchCalls)
	}
	if result.WorkflowID == "" {
		t.Fatal("expected non-empty workflow id")
	}
	if result.Status != WorkflowStatusQueued {
		t.Fatalf("unexpected status: %s", result.Status)
	}
}

func TestStartUseCase_Start_InvalidCommand(t *testing.T) {
	t.Parallel()

	uc := NewStartUseCase(&stubWorkflowDispatcher{
		dispatch: func(ctx context.Context, job DispatchJob) error {
			t.Fatal("dispatch should not be called for invalid command")
			return nil
		},
	}, logger.NewNop())

	_, err := uc.Start(context.Background(), Command{
		UserID:       7,
		ConnectionID: 12,
		Condition: FetchCondition{
			LabelName: "billing",
			Since:     time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
			Until:     time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
		},
	})
	if !errors.Is(err, ErrFetchConditionInvalid) {
		t.Fatalf("expected ErrFetchConditionInvalid, got %v", err)
	}
}

func TestStartUseCase_Start_DispatchFailure(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("dispatch failed")
	uc := NewStartUseCase(&stubWorkflowDispatcher{
		dispatch: func(ctx context.Context, job DispatchJob) error {
			return wantErr
		},
	}, logger.NewNop())

	_, err := uc.Start(context.Background(), Command{
		UserID:       7,
		ConnectionID: 12,
		Condition: FetchCondition{
			LabelName: "billing",
			Since:     time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
			Until:     time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
		},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected dispatch error, got %v", err)
	}
}
