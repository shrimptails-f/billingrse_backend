package infrastructure

import (
	cd "business/internal/common/domain"
	gmaillib "business/internal/library/gmail"
	mfdomain "business/internal/mailfetch/domain"
	"context"
	"errors"
	"testing"
	"time"
)

type stubGmailClientBuilder struct {
	build func(ctx context.Context, connectionID, userID uint) (gmailMessageClient, error)
}

func (s *stubGmailClientBuilder) Build(ctx context.Context, connectionID, userID uint) (gmailMessageClient, error) {
	return s.build(ctx, connectionID, userID)
}

type stubGmailMessageClient struct {
	list   func(ctx context.Context, labelName string, startDate time.Time) ([]string, error)
	detail func(ctx context.Context, id string) (cd.FetchedEmailDTO, error)
}

func (s *stubGmailMessageClient) GetMessagesByLabelName(ctx context.Context, labelName string, startDate time.Time) ([]string, error) {
	return s.list(ctx, labelName, startDate)
}

func (s *stubGmailMessageClient) GetGmailDetail(ctx context.Context, id string) (cd.FetchedEmailDTO, error) {
	return s.detail(ctx, id)
}

func TestGmailMailFetcherAdapter_Fetch_FiltersUntilAndNormalizeFailures(t *testing.T) {
	t.Parallel()

	since := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	until := time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC)

	adapter := NewGmailMailFetcherAdapter(
		mfdomain.ConnectionRef{ConnectionID: 1, UserID: 2, Provider: "gmail", AccountIdentifier: "user@gmail.com"},
		&stubGmailClientBuilder{
			build: func(ctx context.Context, connectionID, userID uint) (gmailMessageClient, error) {
				return &stubGmailMessageClient{
					list: func(ctx context.Context, labelName string, startDate time.Time) ([]string, error) {
						return []string{"msg-1", "msg-2", "msg-3", "msg-4"}, nil
					},
					detail: func(ctx context.Context, id string) (cd.FetchedEmailDTO, error) {
						switch id {
						case "msg-1":
							return cd.FetchedEmailDTO{ID: "msg-1", Date: since.Add(12 * time.Hour)}, nil
						case "msg-2":
							return cd.FetchedEmailDTO{ID: "msg-2", Date: since.Add(-time.Minute)}, nil
						case "msg-3":
							return cd.FetchedEmailDTO{ID: "msg-3", Date: until}, nil
						case "msg-4":
							return cd.FetchedEmailDTO{ID: "msg-4"}, nil
						default:
							return cd.FetchedEmailDTO{}, nil
						}
					},
				}, nil
			},
		},
		nil,
	)

	fetched, failures, err := adapter.Fetch(context.Background(), mfdomain.FetchCondition{
		LabelName: "billing",
		Since:     since,
		Until:     until,
	})
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}

	if len(fetched) != 1 || fetched[0].ID != "msg-1" {
		t.Fatalf("unexpected fetched messages: %+v", fetched)
	}
	if len(failures) != 1 || failures[0].ExternalMessageID != "msg-4" || failures[0].Stage != mfdomain.FailureStageNormalize {
		t.Fatalf("unexpected failures: %+v", failures)
	}
	if failures[0].Message != "取得メール(msg-4)の受信日時が不正でした。" {
		t.Fatalf("unexpected normalize failure message: %+v", failures[0])
	}
}

func TestGmailMailFetcherAdapter_Fetch_DetailFailureContinues(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	adapter := NewGmailMailFetcherAdapter(
		mfdomain.ConnectionRef{ConnectionID: 1, UserID: 2, Provider: "gmail", AccountIdentifier: "user@gmail.com"},
		&stubGmailClientBuilder{
			build: func(ctx context.Context, connectionID, userID uint) (gmailMessageClient, error) {
				return &stubGmailMessageClient{
					list: func(ctx context.Context, labelName string, startDate time.Time) ([]string, error) {
						return []string{"msg-1", "msg-2"}, nil
					},
					detail: func(ctx context.Context, id string) (cd.FetchedEmailDTO, error) {
						if id == "msg-1" {
							return cd.FetchedEmailDTO{}, errors.New("boom")
						}
						return cd.FetchedEmailDTO{ID: "msg-2", Date: now}, nil
					},
				}, nil
			},
		},
		nil,
	)

	fetched, failures, err := adapter.Fetch(context.Background(), mfdomain.FetchCondition{
		LabelName: "billing",
		Since:     now.Add(-time.Hour),
		Until:     now.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}

	if len(fetched) != 1 || fetched[0].ID != "msg-2" {
		t.Fatalf("unexpected fetched messages: %+v", fetched)
	}
	if len(failures) != 1 || failures[0].ExternalMessageID != "msg-1" || failures[0].Stage != mfdomain.FailureStageFetchDetail {
		t.Fatalf("unexpected failures: %+v", failures)
	}
	if failures[0].Message != "Gmail本文の取得に失敗しました。メールID=msg-1" {
		t.Fatalf("unexpected fetch detail failure message: %+v", failures[0])
	}
}

func TestGmailMailFetcherAdapter_Fetch_SessionBuildFailure(t *testing.T) {
	t.Parallel()

	adapter := NewGmailMailFetcherAdapter(
		mfdomain.ConnectionRef{ConnectionID: 1, UserID: 2, Provider: "gmail", AccountIdentifier: "user@gmail.com"},
		&stubGmailClientBuilder{
			build: func(ctx context.Context, connectionID, userID uint) (gmailMessageClient, error) {
				return nil, errors.New("session failed")
			},
		},
		nil,
	)

	_, _, err := adapter.Fetch(context.Background(), mfdomain.FetchCondition{
		LabelName: "billing",
		Since:     time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC),
		Until:     time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC),
	})
	if !errors.Is(err, mfdomain.ErrProviderSessionBuildFailed) {
		t.Fatalf("expected ErrProviderSessionBuildFailed, got %v", err)
	}
}

func TestGmailMailFetcherAdapter_Fetch_LabelNotFound(t *testing.T) {
	t.Parallel()

	adapter := NewGmailMailFetcherAdapter(
		mfdomain.ConnectionRef{ConnectionID: 1, UserID: 2, Provider: "gmail", AccountIdentifier: "user@gmail.com"},
		&stubGmailClientBuilder{
			build: func(ctx context.Context, connectionID, userID uint) (gmailMessageClient, error) {
				return &stubGmailMessageClient{
					list: func(ctx context.Context, labelName string, startDate time.Time) ([]string, error) {
						return nil, gmaillib.ErrLabelNotFound
					},
					detail: func(ctx context.Context, id string) (cd.FetchedEmailDTO, error) {
						return cd.FetchedEmailDTO{}, nil
					},
				}, nil
			},
		},
		nil,
	)

	_, _, err := adapter.Fetch(context.Background(), mfdomain.FetchCondition{
		LabelName: "missing",
		Since:     time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC),
		Until:     time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC),
	})
	if !errors.Is(err, mfdomain.ErrProviderLabelNotFound) {
		t.Fatalf("expected ErrProviderLabelNotFound, got %v", err)
	}
}
