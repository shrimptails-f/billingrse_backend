package application

import (
	"business/internal/library/logger"
	"context"
	"errors"
	"testing"
	"time"
)

type stubWorkflowStatusRepository struct {
	createQueued func(ctx context.Context, cmd QueuedWorkflowHistory) (WorkflowHistoryRef, error)
	markRunning  func(ctx context.Context, historyID uint64, currentStage string) error
	saveStage    func(ctx context.Context, progress StageProgress) error
	complete     func(ctx context.Context, historyID uint64, status string, finishedAt time.Time) error
	fail         func(ctx context.Context, historyID uint64, currentStage string, finishedAt time.Time, errorMessage string) error
	list         func(ctx context.Context, query ListQuery) (ListResult, error)
}

func (s *stubWorkflowStatusRepository) CreateQueued(ctx context.Context, cmd QueuedWorkflowHistory) (WorkflowHistoryRef, error) {
	if s.createQueued == nil {
		return WorkflowHistoryRef{HistoryID: 1, WorkflowID: cmd.WorkflowID}, nil
	}
	return s.createQueued(ctx, cmd)
}

func (s *stubWorkflowStatusRepository) MarkRunning(ctx context.Context, historyID uint64, currentStage string) error {
	if s.markRunning == nil {
		return nil
	}
	return s.markRunning(ctx, historyID, currentStage)
}

func (s *stubWorkflowStatusRepository) SaveStageProgress(ctx context.Context, progress StageProgress) error {
	if s.saveStage == nil {
		return nil
	}
	return s.saveStage(ctx, progress)
}

func (s *stubWorkflowStatusRepository) Complete(ctx context.Context, historyID uint64, status string, finishedAt time.Time) error {
	if s.complete == nil {
		return nil
	}
	return s.complete(ctx, historyID, status, finishedAt)
}

func (s *stubWorkflowStatusRepository) Fail(
	ctx context.Context,
	historyID uint64,
	currentStage string,
	finishedAt time.Time,
	errorMessage string,
) error {
	if s.fail == nil {
		return nil
	}
	return s.fail(ctx, historyID, currentStage, finishedAt, errorMessage)
}

func (s *stubWorkflowStatusRepository) List(ctx context.Context, query ListQuery) (ListResult, error) {
	if s.list == nil {
		return ListResult{}, nil
	}
	return s.list(ctx, query)
}

type fixedClock struct {
	now time.Time
}

func (c *fixedClock) Now() time.Time {
	return c.now
}

func (c *fixedClock) After(d time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	ch <- c.now.Add(d)
	return ch
}

type stubWorkflowDispatcher struct {
	dispatch func(ctx context.Context, job DispatchJob) error
}

func (s *stubWorkflowDispatcher) Dispatch(ctx context.Context, job DispatchJob) error {
	return s.dispatch(ctx, job)
}

func TestStartUseCase_Start_DispatchesValidatedCommand(t *testing.T) {
	t.Parallel()

	dispatchCalls := 0
	queuedCalls := 0
	now := time.Date(2026, 3, 25, 10, 0, 0, 0, time.UTC)
	uc := NewStartUseCase(&stubWorkflowDispatcher{
		dispatch: func(ctx context.Context, job DispatchJob) error {
			dispatchCalls++
			if job.HistoryID != 99 {
				t.Fatalf("unexpected history id: %d", job.HistoryID)
			}
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
	}, &stubWorkflowStatusRepository{
		createQueued: func(ctx context.Context, cmd QueuedWorkflowHistory) (WorkflowHistoryRef, error) {
			queuedCalls++
			if cmd.UserID != 7 || cmd.ConnectionID != 12 {
				t.Fatalf("unexpected queued history command: %+v", cmd)
			}
			if cmd.LabelName != "billing" {
				t.Fatalf("unexpected normalized label name: %+v", cmd)
			}
			if !cmd.QueuedAt.Equal(now) {
				t.Fatalf("unexpected queued at: %s", cmd.QueuedAt)
			}
			return WorkflowHistoryRef{HistoryID: 99, WorkflowID: cmd.WorkflowID}, nil
		},
	}, &fixedClock{now: now}, logger.NewNop())

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
	if queuedCalls != 1 {
		t.Fatalf("expected 1 queued history call, got %d", queuedCalls)
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
	}, &stubWorkflowStatusRepository{}, &fixedClock{now: time.Date(2026, 3, 25, 10, 0, 0, 0, time.UTC)}, logger.NewNop())

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
	failCalls := 0
	now := time.Date(2026, 3, 25, 10, 0, 0, 0, time.UTC)
	uc := NewStartUseCase(&stubWorkflowDispatcher{
		dispatch: func(ctx context.Context, job DispatchJob) error {
			return wantErr
		},
	}, &stubWorkflowStatusRepository{
		createQueued: func(ctx context.Context, cmd QueuedWorkflowHistory) (WorkflowHistoryRef, error) {
			return WorkflowHistoryRef{HistoryID: 55, WorkflowID: cmd.WorkflowID}, nil
		},
		fail: func(ctx context.Context, historyID uint64, currentStage string, finishedAt time.Time, errorMessage string) error {
			failCalls++
			if historyID != 55 {
				t.Fatalf("unexpected history id: %d", historyID)
			}
			if currentStage != "" {
				t.Fatalf("unexpected current stage: %q", currentStage)
			}
			if !finishedAt.Equal(now) {
				t.Fatalf("unexpected finished at: %s", finishedAt)
			}
			if errorMessage != "メール取得ワークフローの起動に失敗しました。" {
				t.Fatalf("unexpected error message: %q", errorMessage)
			}
			return nil
		},
	}, &fixedClock{now: now}, logger.NewNop())

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
	if failCalls != 1 {
		t.Fatalf("expected 1 fail call, got %d", failCalls)
	}
}
