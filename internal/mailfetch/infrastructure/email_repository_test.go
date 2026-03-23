package infrastructure

import (
	cd "business/internal/common/domain"
	"business/internal/library/logger"
	"business/internal/library/mysql"
	mfdomain "business/internal/mailfetch/domain"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type emailRepoTestEnv struct {
	repo  *GormEmailRepositoryAdapter
	db    *gorm.DB
	clean func() error
}

type repoRecordedLogEntry struct {
	message string
	fields  []logger.Field
}

type repoRecordingLogger struct {
	errorEntries []repoRecordedLogEntry
}

func (l *repoRecordingLogger) Debug(message string, fields ...logger.Field) {}
func (l *repoRecordingLogger) Info(message string, fields ...logger.Field)  {}
func (l *repoRecordingLogger) Warn(message string, fields ...logger.Field)  {}
func (l *repoRecordingLogger) Fatal(message string, fields ...logger.Field) {}

func (l *repoRecordingLogger) Error(message string, fields ...logger.Field) {
	l.errorEntries = append(l.errorEntries, repoRecordedLogEntry{
		message: message,
		fields:  append([]logger.Field(nil), fields...),
	})
}

func (l *repoRecordingLogger) With(fields ...logger.Field) logger.Interface { return l }
func (l *repoRecordingLogger) WithContext(ctx context.Context) (logger.Interface, error) {
	return l, nil
}
func (l *repoRecordingLogger) Sync() error { return nil }

func hasRepoField(entry repoRecordedLogEntry, key string) bool {
	for _, field := range entry.fields {
		if field.Key == key {
			return true
		}
	}
	return false
}

func newEmailRepoTestEnv(t *testing.T) *emailRepoTestEnv {
	t.Helper()

	mysqlConn, cleanup, err := mysql.CreateNewTestDB()
	if err != nil {
		skipIfEmailRepoDBUnavailable(t, err)
	}
	require.NoError(t, err)
	require.NoError(t, mysqlConn.DB.AutoMigrate(&emailRecord{}))

	return &emailRepoTestEnv{
		repo:  NewGormEmailRepositoryAdapter(mysqlConn.DB, logger.NewNop()),
		db:    mysqlConn.DB,
		clean: cleanup,
	}
}

func newEmailRepoTestEnvWithLogger(t *testing.T, log logger.Interface) *emailRepoTestEnv {
	t.Helper()

	mysqlConn, cleanup, err := mysql.CreateNewTestDB()
	if err != nil {
		skipIfEmailRepoDBUnavailable(t, err)
	}
	require.NoError(t, err)
	require.NoError(t, mysqlConn.DB.AutoMigrate(&emailRecord{}))

	return &emailRepoTestEnv{
		repo:  NewGormEmailRepositoryAdapter(mysqlConn.DB, log),
		db:    mysqlConn.DB,
		clean: cleanup,
	}
}

func skipIfEmailRepoDBUnavailable(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "dial tcp") || strings.Contains(err.Error(), "lookup mysql") {
		t.Skipf("Skipping repository integration test: %v", err)
	}
}

func TestGormEmailRepositoryAdapter_SaveAllIfAbsent_CreatedAndExisting(t *testing.T) {
	t.Parallel()

	env := newEmailRepoTestEnv(t)
	defer env.clean()

	ctx := context.Background()
	source := mfdomain.EmailSource{Provider: "gmail", AccountIdentifier: "user@gmail.com"}
	dto := cd.FetchedEmailDTO{
		ID:      "msg-1",
		Subject: "subject",
		From:    "from@example.com",
		To:      []string{"to1@example.com", "to2@example.com"},
		Date:    time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC),
	}

	firstResults, firstFailures, err := env.repo.SaveAllIfAbsent(ctx, 1, source, []cd.FetchedEmailDTO{dto})
	require.NoError(t, err)
	require.Empty(t, firstFailures)
	require.Len(t, firstResults, 1)
	require.Equal(t, mfdomain.SaveStatusCreated, firstResults[0].Status)
	require.NotZero(t, firstResults[0].EmailID)

	secondResults, secondFailures, err := env.repo.SaveAllIfAbsent(ctx, 1, source, []cd.FetchedEmailDTO{dto})
	require.NoError(t, err)
	require.Empty(t, secondFailures)
	require.Len(t, secondResults, 1)
	require.Equal(t, mfdomain.SaveStatusExisting, secondResults[0].Status)
	require.Equal(t, firstResults[0].EmailID, secondResults[0].EmailID)

	var stored emailRecord
	require.NoError(t, env.db.WithContext(ctx).First(&stored, firstResults[0].EmailID).Error)
	require.Equal(t, "gmail", stored.Provider)
	require.Equal(t, "user@gmail.com", stored.AccountIdentifier)
	require.Equal(t, "msg-1", stored.ExternalMessageID)
	require.NotNil(t, stored.CreatedRunID)
	require.NotEmpty(t, *stored.CreatedRunID)

	var recipients []string
	require.NoError(t, json.Unmarshal([]byte(stored.ToJSON), &recipients))
	require.Equal(t, []string{"to1@example.com", "to2@example.com"}, recipients)
}

func TestGormEmailRepositoryAdapter_SaveAllIfAbsent_DifferentAccountIdentifierReturnsExisting(t *testing.T) {
	t.Parallel()

	env := newEmailRepoTestEnv(t)
	defer env.clean()

	ctx := context.Background()
	dto := cd.FetchedEmailDTO{
		ID:      "msg-1",
		Subject: "subject",
		From:    "from@example.com",
		To:      []string{"to@example.com"},
		Date:    time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC),
	}

	firstResults, firstFailures, err := env.repo.SaveAllIfAbsent(ctx, 1, mfdomain.EmailSource{Provider: "gmail", AccountIdentifier: "first@gmail.com"}, []cd.FetchedEmailDTO{dto})
	require.NoError(t, err)
	require.Empty(t, firstFailures)
	require.Len(t, firstResults, 1)
	require.Equal(t, mfdomain.SaveStatusCreated, firstResults[0].Status)

	secondResults, secondFailures, err := env.repo.SaveAllIfAbsent(ctx, 1, mfdomain.EmailSource{Provider: "gmail", AccountIdentifier: "second@gmail.com"}, []cd.FetchedEmailDTO{dto})
	require.NoError(t, err)
	require.Empty(t, secondFailures)
	require.Len(t, secondResults, 1)
	require.Equal(t, mfdomain.SaveStatusExisting, secondResults[0].Status)
	require.Equal(t, firstResults[0].EmailID, secondResults[0].EmailID)

	var stored emailRecord
	require.NoError(t, env.db.WithContext(ctx).First(&stored, firstResults[0].EmailID).Error)
	require.Equal(t, "first@gmail.com", stored.AccountIdentifier)
}

func TestGormEmailRepositoryAdapter_SaveAllIfAbsent_MixedBatch(t *testing.T) {
	t.Parallel()

	env := newEmailRepoTestEnv(t)
	defer env.clean()

	ctx := context.Background()
	source := mfdomain.EmailSource{Provider: "gmail", AccountIdentifier: "batch@gmail.com"}
	existingDTO := cd.FetchedEmailDTO{
		ID:      "msg-existing",
		Subject: "existing",
		From:    "from@example.com",
		To:      []string{"to@example.com"},
		Date:    time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC),
	}
	newDTO := cd.FetchedEmailDTO{
		ID:      "msg-new",
		Subject: "new",
		From:    "from@example.com",
		To:      []string{"to@example.com"},
		Date:    time.Date(2026, 3, 23, 11, 0, 0, 0, time.UTC),
	}

	initialResults, initialFailures, err := env.repo.SaveAllIfAbsent(ctx, 1, source, []cd.FetchedEmailDTO{existingDTO})
	require.NoError(t, err)
	require.Empty(t, initialFailures)
	require.Len(t, initialResults, 1)

	results, failures, err := env.repo.SaveAllIfAbsent(ctx, 1, source, []cd.FetchedEmailDTO{existingDTO, newDTO})
	require.NoError(t, err)
	require.Empty(t, failures)
	require.Len(t, results, 2)
	require.Equal(t, mfdomain.SaveStatusExisting, results[0].Status)
	require.Equal(t, initialResults[0].EmailID, results[0].EmailID)
	require.Equal(t, mfdomain.SaveStatusCreated, results[1].Status)
	require.NotZero(t, results[1].EmailID)
}

func TestGormEmailRepositoryAdapter_SaveAllIfAbsent_ContinuesAfterChunkFailure(t *testing.T) {
	t.Parallel()

	log := &repoRecordingLogger{}
	env := newEmailRepoTestEnvWithLogger(t, log)
	defer env.clean()

	ctx := context.Background()
	source := mfdomain.EmailSource{Provider: "gmail", AccountIdentifier: "user@gmail.com"}
	baseDate := time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC)

	dtos := make([]cd.FetchedEmailDTO, 0, 21)
	for i := 1; i <= 21; i++ {
		subject := fmt.Sprintf("subject-%02d", i)
		id := fmt.Sprintf("msg-%02d", i)
		if i == 2 {
			id = "msg-too-long-" + strings.Repeat("x", 280)
		}
		dtos = append(dtos, cd.FetchedEmailDTO{
			ID:      id,
			Subject: subject,
			From:    "from@example.com",
			To:      []string{"to@example.com"},
			Date:    baseDate.Add(time.Duration(i) * time.Minute),
		})
	}

	results, failures, err := env.repo.SaveAllIfAbsent(ctx, 1, source, dtos)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "msg-21", results[0].ExternalMessageID)
	require.Len(t, failures, 20)

	for idx, failure := range failures {
		expectedID := fmt.Sprintf("msg-%02d", idx+1)
		if idx == 1 {
			expectedID = "msg-too-long-" + strings.Repeat("x", 280)
		}
		require.Equal(t, expectedID, failure.ExternalMessageID)
		require.Equal(t, mfdomain.FailureStageSave, failure.Stage)
		require.Equal(t, mfdomain.FailureCodeEmailSaveFailed, failure.Code)
	}

	var stored []emailRecord
	require.NoError(t, env.db.Order("id ASC").Find(&stored).Error)
	require.Len(t, stored, 1)
	require.Equal(t, "msg-21", stored[0].ExternalMessageID)

	require.Len(t, log.errorEntries, 1)
	require.Equal(t, "manual_mail_fetch_email_batch_insert_failed", log.errorEntries[0].message)
	require.True(t, hasRepoField(log.errorEntries[0], "user_id"))
	require.True(t, hasRepoField(log.errorEntries[0], "provider"))
	require.True(t, hasRepoField(log.errorEntries[0], "chunk_start_index"))
	require.True(t, hasRepoField(log.errorEntries[0], "chunk_size"))
	require.True(t, hasRepoField(log.errorEntries[0], "chunk_external_message_ids"))
}
