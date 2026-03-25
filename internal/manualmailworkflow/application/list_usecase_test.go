package application

import (
	"business/internal/library/logger"
	"context"
	"errors"
	"testing"
	"time"
)

func TestListUseCase_List_NormalizesAndDefaultsQuery(t *testing.T) {
	t.Parallel()

	called := 0
	uc := NewListUseCase(&stubWorkflowStatusRepository{
		list: func(ctx context.Context, query ListQuery) (ListResult, error) {
			called++
			if query.UserID != 7 {
				t.Fatalf("unexpected user id: %d", query.UserID)
			}
			if query.Limit != defaultWorkflowHistoryListLimit {
				t.Fatalf("unexpected default limit: %d", query.Limit)
			}
			if query.Offset != defaultWorkflowHistoryListOffset {
				t.Fatalf("unexpected default offset: %d", query.Offset)
			}
			if query.Status == nil || *query.Status != WorkflowStatusPartialSuccess {
				t.Fatalf("unexpected normalized status: %+v", query.Status)
			}

			return ListResult{
				Items: []WorkflowHistoryListItem{
					{
						WorkflowID:        "wf-1",
						Provider:          "gmail",
						AccountIdentifier: "billing@example.com",
						LabelName:         "billing",
						Since:             time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
						Until:             time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
						Status:            WorkflowStatusPartialSuccess,
						QueuedAt:          time.Date(2026, 3, 25, 9, 0, 0, 0, time.UTC),
						Fetch: StageSummaryView{
							Failures: []StageFailureView{},
						},
						Analysis: StageSummaryView{
							Failures: []StageFailureView{},
						},
						VendorResolution: StageSummaryView{
							Failures: []StageFailureView{},
						},
						BillingEligibility: StageSummaryView{
							Failures: []StageFailureView{},
						},
						Billing: StageSummaryView{
							Failures: []StageFailureView{},
						},
					},
				},
				TotalCount: 1,
			}, nil
		},
	}, logger.NewNop())

	result, err := uc.List(context.Background(), ListQuery{
		UserID: 7,
		Status: stringPtr(" partial_success "),
	})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if called != 1 {
		t.Fatalf("expected repository to be called once, got %d", called)
	}
	if result.TotalCount != 1 {
		t.Fatalf("unexpected total count: %d", result.TotalCount)
	}
	if len(result.Items) != 1 || result.Items[0].WorkflowID != "wf-1" {
		t.Fatalf("unexpected items: %+v", result.Items)
	}
}

func TestListUseCase_List_RejectsInvalidQuery(t *testing.T) {
	t.Parallel()

	uc := NewListUseCase(&stubWorkflowStatusRepository{}, logger.NewNop())

	_, err := uc.List(context.Background(), ListQuery{
		UserID:   7,
		Limit:    0,
		HasLimit: true,
	})
	if !errors.Is(err, ErrInvalidListQuery) {
		t.Fatalf("expected ErrInvalidListQuery, got %v", err)
	}
}

func TestListUseCase_List_RepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("list failed")
	uc := NewListUseCase(&stubWorkflowStatusRepository{
		list: func(ctx context.Context, query ListQuery) (ListResult, error) {
			return ListResult{}, wantErr
		},
	}, logger.NewNop())

	_, err := uc.List(context.Background(), ListQuery{
		UserID:    7,
		Limit:     10,
		Offset:    5,
		HasLimit:  true,
		HasOffset: true,
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected repository error, got %v", err)
	}
}
