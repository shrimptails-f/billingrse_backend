package application

import (
	"business/internal/library/logger"
	"context"
	"errors"
	"testing"
	"time"
)

type stubBillingListRepository struct {
	list func(ctx context.Context, query ListQuery) (ListResult, error)
}

func (s *stubBillingListRepository) List(ctx context.Context, query ListQuery) (ListResult, error) {
	return s.list(ctx, query)
}

func TestListUseCase_List_NormalizesDefaultsAndReturnsResult(t *testing.T) {
	t.Parallel()

	originalDateFrom := time.Date(2026, 3, 24, 9, 0, 0, 0, time.FixedZone("JST", 9*60*60))
	emailID := uint(42)

	var capturedQuery ListQuery
	uc := NewListUseCase(&stubBillingListRepository{
		list: func(ctx context.Context, query ListQuery) (ListResult, error) {
			capturedQuery = query
			return ListResult{
				Items: []ListItem{
					{
						EmailID:           101,
						ExternalMessageID: "msg-101",
						VendorName:        "AWS",
						ReceivedAt:        time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
						Amount:            1200.5,
						Currency:          "JPY",
					},
				},
				TotalCount: 1,
			}, nil
		},
	}, logger.NewNop())

	result, err := uc.List(context.Background(), ListQuery{
		UserID:            7,
		Q:                 "  AWS  ",
		EmailID:           &emailID,
		ExternalMessageID: " msg-101 ",
		DateFrom:          &originalDateFrom,
	})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if capturedQuery.Q != "AWS" {
		t.Fatalf("expected normalized q, got %#v", capturedQuery.Q)
	}
	if capturedQuery.ExternalMessageID != "msg-101" {
		t.Fatalf("expected normalized external_message_id, got %#v", capturedQuery.ExternalMessageID)
	}
	if capturedQuery.DateFrom == nil || !capturedQuery.DateFrom.Equal(originalDateFrom.UTC()) {
		t.Fatalf("expected UTC-normalized date_from, got %#v", capturedQuery.DateFrom)
	}
	if capturedQuery.UseReceivedAtFallback == nil || !*capturedQuery.UseReceivedAtFallback {
		t.Fatalf("expected default fallback=true, got %#v", capturedQuery.UseReceivedAtFallback)
	}
	if capturedQuery.Limit == nil || *capturedQuery.Limit != defaultBillingListLimit {
		t.Fatalf("expected default limit, got %#v", capturedQuery.Limit)
	}
	if capturedQuery.Offset == nil || *capturedQuery.Offset != defaultBillingListOffset {
		t.Fatalf("expected default offset, got %#v", capturedQuery.Offset)
	}
	if capturedQuery.EmailID == nil || *capturedQuery.EmailID != emailID {
		t.Fatalf("expected email_id to be preserved, got %#v", capturedQuery.EmailID)
	}

	if result.Limit != defaultBillingListLimit {
		t.Fatalf("expected default limit in result, got %d", result.Limit)
	}
	if result.Offset != defaultBillingListOffset {
		t.Fatalf("expected default offset in result, got %d", result.Offset)
	}
	if result.TotalCount != 1 || len(result.Items) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestListUseCase_List_RejectsInvalidQuery(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)
	later := now.Add(time.Hour)
	zeroEmailID := uint(0)
	zeroLimit := 0
	tooLargeLimit := 101
	negativeOffset := -1

	testCases := []struct {
		name  string
		query ListQuery
	}{
		{
			name: "missing user id",
			query: ListQuery{
				Limit:  intPtr(10),
				Offset: intPtr(0),
			},
		},
		{
			name: "date_from after date_to",
			query: ListQuery{
				UserID:   1,
				DateFrom: &later,
				DateTo:   &now,
			},
		},
		{
			name: "zero email id",
			query: ListQuery{
				UserID:  1,
				EmailID: &zeroEmailID,
			},
		},
		{
			name: "limit too small",
			query: ListQuery{
				UserID: 1,
				Limit:  &zeroLimit,
			},
		},
		{
			name: "limit too large",
			query: ListQuery{
				UserID: 1,
				Limit:  &tooLargeLimit,
			},
		},
		{
			name: "negative offset",
			query: ListQuery{
				UserID: 1,
				Offset: &negativeOffset,
			},
		},
	}

	uc := NewListUseCase(&stubBillingListRepository{
		list: func(ctx context.Context, query ListQuery) (ListResult, error) {
			t.Fatal("repository should not be called for invalid query")
			return ListResult{}, nil
		},
	}, logger.NewNop())

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := uc.List(context.Background(), tc.query)
			if !errors.Is(err, ErrInvalidListQuery) {
				t.Fatalf("expected ErrInvalidListQuery, got %v", err)
			}
		})
	}
}

func TestListUseCase_List_InitializesEmptyItems(t *testing.T) {
	t.Parallel()

	uc := NewListUseCase(&stubBillingListRepository{
		list: func(ctx context.Context, query ListQuery) (ListResult, error) {
			return ListResult{}, nil
		},
	}, logger.NewNop())

	result, err := uc.List(context.Background(), ListQuery{UserID: 1})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if result.Items == nil {
		t.Fatal("expected empty items slice, got nil")
	}
}

func intPtr(value int) *int {
	return &value
}
