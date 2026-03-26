package application

import (
	cd "business/internal/common/domain"
	"business/internal/library/logger"
	mfdomain "business/internal/mailfetch/domain"
	"context"
	"errors"
	"testing"
	"time"
)

type recordedLogEntry struct {
	message string
	fields  []logger.Field
}

type recordingLogger struct {
	infoEntries  []recordedLogEntry
	errorEntries []recordedLogEntry
}

func (l *recordingLogger) Debug(message string, fields ...logger.Field) {}

func (l *recordingLogger) Info(message string, fields ...logger.Field) {
	l.infoEntries = append(l.infoEntries, recordedLogEntry{message: message, fields: append([]logger.Field(nil), fields...)})
}

func (l *recordingLogger) Warn(message string, fields ...logger.Field) {}

func (l *recordingLogger) Error(message string, fields ...logger.Field) {
	l.errorEntries = append(l.errorEntries, recordedLogEntry{message: message, fields: append([]logger.Field(nil), fields...)})
}

func (l *recordingLogger) Fatal(message string, fields ...logger.Field) {}

func (l *recordingLogger) With(fields ...logger.Field) logger.Interface { return l }

func (l *recordingLogger) WithContext(ctx context.Context) (logger.Interface, error) { return l, nil }

func (l *recordingLogger) Sync() error { return nil }

func hasField(entries []recordedLogEntry, key string) bool {
	for _, entry := range entries {
		for _, field := range entry.fields {
			if field.Key == key {
				return true
			}
		}
	}
	return false
}

type mockConnectionRepository struct {
	findUsableConnection func(ctx context.Context, userID, connectionID uint) (mfdomain.ConnectionRef, error)
}

func (m *mockConnectionRepository) FindUsableConnection(ctx context.Context, userID, connectionID uint) (mfdomain.ConnectionRef, error) {
	return m.findUsableConnection(ctx, userID, connectionID)
}

type mockMailFetcherFactory struct {
	create func(ctx context.Context, conn mfdomain.ConnectionRef) (MailFetcher, error)
}

func (m *mockMailFetcherFactory) Create(ctx context.Context, conn mfdomain.ConnectionRef) (MailFetcher, error) {
	return m.create(ctx, conn)
}

type mockMailFetcher struct {
	fetch func(ctx context.Context, cond mfdomain.FetchCondition) ([]cd.FetchedEmailDTO, []mfdomain.MessageFailure, error)
}

func (m *mockMailFetcher) Fetch(ctx context.Context, cond mfdomain.FetchCondition) ([]cd.FetchedEmailDTO, []mfdomain.MessageFailure, error) {
	return m.fetch(ctx, cond)
}

type mockEmailRepository struct {
	saveAllIfAbsent func(ctx context.Context, userID uint, source mfdomain.EmailSource, dtos []cd.FetchedEmailDTO) ([]mfdomain.SaveResult, []mfdomain.MessageFailure, error)
}

func (m *mockEmailRepository) SaveAllIfAbsent(ctx context.Context, userID uint, source mfdomain.EmailSource, dtos []cd.FetchedEmailDTO) ([]mfdomain.SaveResult, []mfdomain.MessageFailure, error) {
	return m.saveAllIfAbsent(ctx, userID, source, dtos)
}

func TestUseCaseExecute_CreatesAndReturnsExistingIDs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC)
	conn := mfdomain.ConnectionRef{
		ConnectionID:      10,
		UserID:            5,
		Provider:          "gmail",
		AccountIdentifier: "user@gmail.com",
	}

	saveCalls := 0
	uc := NewUseCase(
		&mockConnectionRepository{
			findUsableConnection: func(ctx context.Context, userID, connectionID uint) (mfdomain.ConnectionRef, error) {
				return conn, nil
			},
		},
		&mockMailFetcherFactory{
			create: func(ctx context.Context, gotConn mfdomain.ConnectionRef) (MailFetcher, error) {
				if gotConn != conn {
					t.Fatalf("unexpected connection: %+v", gotConn)
				}
				return &mockMailFetcher{
					fetch: func(ctx context.Context, cond mfdomain.FetchCondition) ([]cd.FetchedEmailDTO, []mfdomain.MessageFailure, error) {
						return []cd.FetchedEmailDTO{
							{ID: "msg-1", Subject: "a", From: "from1", To: []string{"to1"}, Date: now, Body: "body-1"},
							{ID: "msg-2", Subject: "b", From: "from2", To: []string{"to2"}, Date: now, Body: "body-2"},
							{ID: "msg-2", Subject: "dup", From: "from2", To: []string{"to2"}, Date: now, Body: "dup-body"},
						}, nil, nil
					},
				}, nil
			},
		},
		&mockEmailRepository{
			saveAllIfAbsent: func(ctx context.Context, userID uint, source mfdomain.EmailSource, dtos []cd.FetchedEmailDTO) ([]mfdomain.SaveResult, []mfdomain.MessageFailure, error) {
				saveCalls++
				if userID != 5 {
					t.Fatalf("unexpected userID: %d", userID)
				}
				if source.Provider != "gmail" || source.AccountIdentifier != "user@gmail.com" {
					t.Fatalf("unexpected source: %+v", source)
				}
				if len(dtos) != 2 {
					t.Fatalf("expected 2 deduped save targets, got %+v", dtos)
				}
				if dtos[0].BodyDigest != computeBodyDigest("body-1") {
					t.Fatalf("unexpected body digest for msg-1: %+v", dtos[0])
				}
				if dtos[1].BodyDigest != computeBodyDigest("body-2") {
					t.Fatalf("unexpected body digest for msg-2: %+v", dtos[1])
				}
				return []mfdomain.SaveResult{
					{EmailID: 101, ExternalMessageID: "msg-1", Status: mfdomain.SaveStatusCreated},
					{EmailID: 202, ExternalMessageID: "msg-2", Status: mfdomain.SaveStatusExisting},
				}, nil, nil
			},
		},
		logger.NewNop(),
	)

	result, err := uc.Execute(ctx, Command{
		UserID:       5,
		ConnectionID: 10,
		Condition: mfdomain.FetchCondition{
			LabelName: "billing",
			Since:     now.Add(-time.Hour),
			Until:     now.Add(time.Hour),
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if saveCalls != 1 {
		t.Fatalf("expected 1 save call, got %d", saveCalls)
	}
	if result.Provider != "gmail" || result.AccountIdentifier != "user@gmail.com" {
		t.Fatalf("unexpected result source: %+v", result)
	}
	if result.MatchedMessageCount != 3 {
		t.Fatalf("expected matched count 3, got %d", result.MatchedMessageCount)
	}
	if len(result.CreatedEmails) != 1 {
		t.Fatalf("unexpected created emails: %+v", result.CreatedEmails)
	}
	if result.CreatedEmails[0].EmailID != 101 || result.CreatedEmails[0].ExternalMessageID != "msg-1" || result.CreatedEmails[0].Body != "body-1" {
		t.Fatalf("unexpected created email payload: %+v", result.CreatedEmails[0])
	}
	if result.CreatedEmails[0].BodyDigest != computeBodyDigest("body-1") {
		t.Fatalf("unexpected created email body digest: %+v", result.CreatedEmails[0])
	}
	if len(result.ExistingEmailIDs) != 1 || result.ExistingEmailIDs[0] != 202 {
		t.Fatalf("unexpected existing ids: %+v", result.ExistingEmailIDs)
	}
	if len(result.Failures) != 1 {
		t.Fatalf("unexpected failures: %+v", result.Failures)
	}
	if result.Failures[0].ExternalMessageID != "msg-2" || result.Failures[0].Stage != mfdomain.FailureStageNormalize || result.Failures[0].Code != mfdomain.FailureCodeDuplicateExternalMessageID {
		t.Fatalf("unexpected duplicate failure: %+v", result.Failures[0])
	}
	if result.Failures[0].Message != "取得結果に重複したメールID(msg-2)が含まれていました。" {
		t.Fatalf("unexpected duplicate failure message: %+v", result.Failures[0])
	}
}

func TestUseCaseExecute_InvalidCondition(t *testing.T) {
	t.Parallel()

	uc := NewUseCase(
		&mockConnectionRepository{
			findUsableConnection: func(ctx context.Context, userID, connectionID uint) (mfdomain.ConnectionRef, error) {
				t.Fatal("connection lookup should not be called")
				return mfdomain.ConnectionRef{}, nil
			},
		},
		&mockMailFetcherFactory{
			create: func(ctx context.Context, conn mfdomain.ConnectionRef) (MailFetcher, error) {
				t.Fatal("factory should not be called")
				return nil, nil
			},
		},
		&mockEmailRepository{
			saveAllIfAbsent: func(ctx context.Context, userID uint, source mfdomain.EmailSource, dtos []cd.FetchedEmailDTO) ([]mfdomain.SaveResult, []mfdomain.MessageFailure, error) {
				t.Fatal("save should not be called")
				return nil, nil, nil
			},
		},
		logger.NewNop(),
	)

	_, err := uc.Execute(context.Background(), Command{
		UserID:       1,
		ConnectionID: 2,
		Condition: mfdomain.FetchCondition{
			LabelName: "billing",
			Since:     time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
			Until:     time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC),
		},
	})
	if !errors.Is(err, mfdomain.ErrFetchConditionInvalid) {
		t.Fatalf("expected ErrFetchConditionInvalid, got %v", err)
	}
}

func TestUseCaseExecute_PartialFailuresContinue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Date(2026, 3, 23, 11, 0, 0, 0, time.UTC)

	uc := NewUseCase(
		&mockConnectionRepository{
			findUsableConnection: func(ctx context.Context, userID, connectionID uint) (mfdomain.ConnectionRef, error) {
				return mfdomain.ConnectionRef{
					ConnectionID:      9,
					UserID:            7,
					Provider:          "gmail",
					AccountIdentifier: "partial@gmail.com",
				}, nil
			},
		},
		&mockMailFetcherFactory{
			create: func(ctx context.Context, conn mfdomain.ConnectionRef) (MailFetcher, error) {
				return &mockMailFetcher{
					fetch: func(ctx context.Context, cond mfdomain.FetchCondition) ([]cd.FetchedEmailDTO, []mfdomain.MessageFailure, error) {
						return []cd.FetchedEmailDTO{
								{ID: "msg-1", Subject: "ok", From: "from1", To: []string{"to1"}, Date: now, Body: "body-1"},
								{ID: "msg-2", Subject: "ng", From: "from2", To: []string{"to2"}, Date: now, Body: "body-2"},
							}, []mfdomain.MessageFailure{
								{ExternalMessageID: "msg-0", Stage: mfdomain.FailureStageFetchDetail, Code: mfdomain.FailureCodeFetchDetailFailed, Message: "Gmail本文の取得に失敗しました。メールID=msg-0"},
							}, nil
					},
				}, nil
			},
		},
		&mockEmailRepository{
			saveAllIfAbsent: func(ctx context.Context, userID uint, source mfdomain.EmailSource, dtos []cd.FetchedEmailDTO) ([]mfdomain.SaveResult, []mfdomain.MessageFailure, error) {
				return []mfdomain.SaveResult{
						{EmailID: 55, ExternalMessageID: "msg-1", Status: mfdomain.SaveStatusCreated},
					}, []mfdomain.MessageFailure{
						{ExternalMessageID: "msg-2", Stage: mfdomain.FailureStageSave, Code: mfdomain.FailureCodeEmailSaveFailed, Message: "取得メール(msg-2)の保存に失敗しました。"},
					}, nil
			},
		},
		logger.NewNop(),
	)

	result, err := uc.Execute(ctx, Command{
		UserID:       7,
		ConnectionID: 9,
		Condition: mfdomain.FetchCondition{
			LabelName: "billing",
			Since:     now.Add(-time.Hour),
			Until:     now.Add(time.Hour),
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if len(result.CreatedEmails) != 1 || result.CreatedEmails[0].EmailID != 55 || result.CreatedEmails[0].Body != "body-1" {
		t.Fatalf("unexpected created emails: %+v", result.CreatedEmails)
	}
	if len(result.Failures) != 2 {
		t.Fatalf("expected 2 failures, got %+v", result.Failures)
	}
	if result.Failures[0].ExternalMessageID != "msg-0" || result.Failures[0].Stage != mfdomain.FailureStageFetchDetail {
		t.Fatalf("unexpected first failure: %+v", result.Failures[0])
	}
	if result.Failures[1].ExternalMessageID != "msg-2" || result.Failures[1].Stage != mfdomain.FailureStageSave {
		t.Fatalf("unexpected second failure: %+v", result.Failures[1])
	}
	if result.Failures[0].Message != "Gmail本文の取得に失敗しました。メールID=msg-0" || result.Failures[1].Message != "取得メール(msg-2)の保存に失敗しました。" {
		t.Fatalf("unexpected failure messages: %+v", result.Failures)
	}
}

func TestUseCaseExecute_ZeroReceivedAtBecomesNormalizeFailure(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Date(2026, 3, 23, 11, 0, 0, 0, time.UTC)

	uc := NewUseCase(
		&mockConnectionRepository{
			findUsableConnection: func(ctx context.Context, userID, connectionID uint) (mfdomain.ConnectionRef, error) {
				return mfdomain.ConnectionRef{
					ConnectionID:      9,
					UserID:            7,
					Provider:          "gmail",
					AccountIdentifier: "partial@gmail.com",
				}, nil
			},
		},
		&mockMailFetcherFactory{
			create: func(ctx context.Context, conn mfdomain.ConnectionRef) (MailFetcher, error) {
				return &mockMailFetcher{
					fetch: func(ctx context.Context, cond mfdomain.FetchCondition) ([]cd.FetchedEmailDTO, []mfdomain.MessageFailure, error) {
						return []cd.FetchedEmailDTO{
							{ID: "msg-1", Subject: "ok", From: "from1", To: []string{"to1"}, Date: now, Body: "body-1"},
							{ID: "msg-2", Subject: "ng", From: "from2", To: []string{"to2"}},
						}, nil, nil
					},
				}, nil
			},
		},
		&mockEmailRepository{
			saveAllIfAbsent: func(ctx context.Context, userID uint, source mfdomain.EmailSource, dtos []cd.FetchedEmailDTO) ([]mfdomain.SaveResult, []mfdomain.MessageFailure, error) {
				if len(dtos) != 1 || dtos[0].ID != "msg-1" {
					t.Fatalf("unexpected save targets: %+v", dtos)
				}
				return []mfdomain.SaveResult{
					{EmailID: 55, ExternalMessageID: "msg-1", Status: mfdomain.SaveStatusCreated},
				}, nil, nil
			},
		},
		logger.NewNop(),
	)

	result, err := uc.Execute(ctx, Command{
		UserID:       7,
		ConnectionID: 9,
		Condition: mfdomain.FetchCondition{
			LabelName: "billing",
			Since:     now.Add(-time.Hour),
			Until:     now.Add(time.Hour),
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if len(result.CreatedEmails) != 1 || result.CreatedEmails[0].Body != "body-1" {
		t.Fatalf("unexpected created emails: %+v", result.CreatedEmails)
	}
	if len(result.Failures) != 1 {
		t.Fatalf("expected 1 failure, got %+v", result.Failures)
	}
	if result.Failures[0].ExternalMessageID != "msg-2" || result.Failures[0].Stage != mfdomain.FailureStageNormalize {
		t.Fatalf("unexpected normalize failure: %+v", result.Failures[0])
	}
	if result.Failures[0].Message != "取得メール(msg-2)の受信日時が不正でした。" {
		t.Fatalf("unexpected normalize failure message: %+v", result.Failures[0])
	}
}

func TestUseCaseExecute_DoesNotLogAccountIdentifierOnSuccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	log := &recordingLogger{}

	uc := NewUseCase(
		&mockConnectionRepository{
			findUsableConnection: func(ctx context.Context, userID, connectionID uint) (mfdomain.ConnectionRef, error) {
				return mfdomain.ConnectionRef{
					ConnectionID:      3,
					UserID:            2,
					Provider:          "gmail",
					AccountIdentifier: "user@gmail.com",
				}, nil
			},
		},
		&mockMailFetcherFactory{
			create: func(ctx context.Context, conn mfdomain.ConnectionRef) (MailFetcher, error) {
				return &mockMailFetcher{
					fetch: func(ctx context.Context, cond mfdomain.FetchCondition) ([]cd.FetchedEmailDTO, []mfdomain.MessageFailure, error) {
						return []cd.FetchedEmailDTO{
							{ID: "msg-1", Subject: "subject", From: "from@example.com", To: []string{"to@example.com"}, Date: now},
						}, nil, nil
					},
				}, nil
			},
		},
		&mockEmailRepository{
			saveAllIfAbsent: func(ctx context.Context, userID uint, source mfdomain.EmailSource, dtos []cd.FetchedEmailDTO) ([]mfdomain.SaveResult, []mfdomain.MessageFailure, error) {
				return []mfdomain.SaveResult{
					{EmailID: 1, ExternalMessageID: "msg-1", Status: mfdomain.SaveStatusCreated},
				}, nil, nil
			},
		},
		log,
	)

	_, err := uc.Execute(ctx, Command{
		UserID:       2,
		ConnectionID: 3,
		Condition: mfdomain.FetchCondition{
			LabelName: "billing",
			Since:     now.Add(-time.Hour),
			Until:     now.Add(time.Hour),
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if hasField(log.infoEntries, "account_identifier") || hasField(log.errorEntries, "account_identifier") {
		t.Fatalf("account_identifier should not be logged: info=%+v error=%+v", log.infoEntries, log.errorEntries)
	}
}

func TestUseCaseExecute_DoesNotLogAccountIdentifierOnError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Date(2026, 3, 23, 12, 30, 0, 0, time.UTC)
	log := &recordingLogger{}

	uc := NewUseCase(
		&mockConnectionRepository{
			findUsableConnection: func(ctx context.Context, userID, connectionID uint) (mfdomain.ConnectionRef, error) {
				return mfdomain.ConnectionRef{
					ConnectionID:      4,
					UserID:            2,
					Provider:          "gmail",
					AccountIdentifier: "user@gmail.com",
				}, nil
			},
		},
		&mockMailFetcherFactory{
			create: func(ctx context.Context, conn mfdomain.ConnectionRef) (MailFetcher, error) {
				return &mockMailFetcher{
					fetch: func(ctx context.Context, cond mfdomain.FetchCondition) ([]cd.FetchedEmailDTO, []mfdomain.MessageFailure, error) {
						return []cd.FetchedEmailDTO{
							{ID: "msg-1", Subject: "subject", From: "from@example.com", To: []string{"to@example.com"}, Date: now},
						}, nil, nil
					},
				}, nil
			},
		},
		&mockEmailRepository{
			saveAllIfAbsent: func(ctx context.Context, userID uint, source mfdomain.EmailSource, dtos []cd.FetchedEmailDTO) ([]mfdomain.SaveResult, []mfdomain.MessageFailure, error) {
				return nil, nil, errors.New("db down")
			},
		},
		log,
	)

	_, err := uc.Execute(ctx, Command{
		UserID:       2,
		ConnectionID: 4,
		Condition: mfdomain.FetchCondition{
			LabelName: "billing",
			Since:     now.Add(-time.Hour),
			Until:     now.Add(time.Hour),
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}

	if hasField(log.infoEntries, "account_identifier") || hasField(log.errorEntries, "account_identifier") {
		t.Fatalf("account_identifier should not be logged: info=%+v error=%+v", log.infoEntries, log.errorEntries)
	}
}
