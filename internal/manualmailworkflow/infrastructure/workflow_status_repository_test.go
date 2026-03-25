package infrastructure

import (
	"business/internal/library/logger"
	"business/internal/library/mysql"
	manualapp "business/internal/manualmailworkflow/application"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type workflowStatusRepoTestEnv struct {
	repo   *GormWorkflowStatusRepository
	db     *gorm.DB
	nowUTC time.Time
	clean  func() error
}

type workflowStatusRepoFixedClock struct {
	now time.Time
}

func (c *workflowStatusRepoFixedClock) Now() time.Time {
	return c.now
}

func (c *workflowStatusRepoFixedClock) After(d time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	ch <- c.now.Add(d)
	return ch
}

func newWorkflowStatusRepoTestEnv(t *testing.T) *workflowStatusRepoTestEnv {
	t.Helper()

	mysqlConn, cleanup, err := mysql.CreateNewTestDB()
	if err != nil {
		skipIfWorkflowStatusRepoDBUnavailable(t, err)
	}
	require.NoError(t, err)
	require.NoError(t, mysqlConn.DB.AutoMigrate(&manualMailWorkflowHistoryRecord{}, &manualMailWorkflowStageFailureRecord{}))

	nowUTC := time.Date(2026, 3, 25, 15, 0, 0, 0, time.UTC)
	return &workflowStatusRepoTestEnv{
		repo:   NewGormWorkflowStatusRepository(mysqlConn.DB, &workflowStatusRepoFixedClock{now: nowUTC}, logger.NewNop()),
		db:     mysqlConn.DB,
		nowUTC: nowUTC,
		clean:  cleanup,
	}
}

func skipIfWorkflowStatusRepoDBUnavailable(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "dial tcp") || strings.Contains(err.Error(), "lookup mysql") {
		t.Skipf("Skipping repository integration test: %v", err)
	}
}

func TestGormWorkflowStatusRepository_CreateSaveProgressAndComplete(t *testing.T) {
	t.Parallel()

	env := newWorkflowStatusRepoTestEnv(t)
	defer env.clean()

	ctx := context.Background()
	queuedAt := time.Date(2026, 3, 25, 14, 55, 0, 0, time.UTC)
	externalMessageID := "msg-2"
	ref, err := env.repo.CreateQueued(ctx, manualapp.QueuedWorkflowHistory{
		WorkflowID:   "01JQ0B7N0M7H3X9C2J5K8V6P4",
		UserID:       10,
		ConnectionID: 20,
		LabelName:    "billing",
		SinceAt:      time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
		UntilAt:      time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
		QueuedAt:     queuedAt,
	})
	require.NoError(t, err)
	require.NotZero(t, ref.HistoryID)

	require.NoError(t, env.repo.MarkRunning(ctx, ref.HistoryID, "fetch"))

	require.NoError(t, env.repo.SaveStageProgress(ctx, manualapp.StageProgress{
		HistoryID:             ref.HistoryID,
		Stage:                 "fetch",
		SuccessCount:          2,
		BusinessFailureCount:  0,
		TechnicalFailureCount: 1,
		FailureRecords: []manualapp.StageFailureRecord{
			{
				Stage:             "fetch",
				ExternalMessageID: &externalMessageID,
				ReasonCode:        "email_save_failed",
				Message:           "取得したメールの保存に失敗しました。",
			},
		},
	}))

	finishedAt := time.Date(2026, 3, 25, 15, 10, 0, 0, time.UTC)
	require.NoError(t, env.repo.Complete(ctx, ref.HistoryID, manualapp.WorkflowStatusPartialSuccess, finishedAt))

	var history manualMailWorkflowHistoryRecord
	require.NoError(t, env.db.WithContext(ctx).First(&history, ref.HistoryID).Error)
	require.Equal(t, "01JQ0B7N0M7H3X9C2J5K8V6P4", history.WorkflowID)
	require.Equal(t, uint(10), history.UserID)
	require.Equal(t, uint(20), history.ConnectionID)
	require.Equal(t, "billing", history.LabelName)
	require.Equal(t, manualapp.WorkflowStatusPartialSuccess, history.Status)
	require.Nil(t, history.CurrentStage)
	require.True(t, history.QueuedAt.Equal(queuedAt))
	require.NotNil(t, history.FinishedAt)
	require.True(t, history.FinishedAt.Equal(finishedAt))
	require.Equal(t, 2, history.FetchSuccessCount)
	require.Equal(t, 0, history.FetchBusinessFailureCount)
	require.Equal(t, 1, history.FetchTechnicalFailureCount)
	require.True(t, history.CreatedAt.Equal(env.nowUTC))
	require.True(t, history.UpdatedAt.Equal(env.nowUTC))

	var failures []manualMailWorkflowStageFailureRecord
	require.NoError(t, env.db.WithContext(ctx).
		Where("workflow_history_id = ?", ref.HistoryID).
		Find(&failures).Error)
	require.Len(t, failures, 1)
	require.Equal(t, "fetch", failures[0].Stage)
	require.NotNil(t, failures[0].ExternalMessageID)
	require.Equal(t, "msg-2", *failures[0].ExternalMessageID)
	require.Equal(t, "email_save_failed", failures[0].ReasonCode)
	require.Equal(t, "取得したメールの保存に失敗しました。", failures[0].Message)
	require.True(t, failures[0].CreatedAt.Equal(env.nowUTC))
}

func TestGormWorkflowStatusRepository_Fail(t *testing.T) {
	t.Parallel()

	env := newWorkflowStatusRepoTestEnv(t)
	defer env.clean()

	ctx := context.Background()
	ref, err := env.repo.CreateQueued(ctx, manualapp.QueuedWorkflowHistory{
		WorkflowID:   "01JQ0B7N0M7H3X9C2J5K8V6P5",
		UserID:       1,
		ConnectionID: 2,
		LabelName:    "billing",
		SinceAt:      time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
		UntilAt:      time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
		QueuedAt:     time.Date(2026, 3, 25, 14, 55, 0, 0, time.UTC),
	})
	require.NoError(t, err)

	require.NoError(t, env.repo.MarkRunning(ctx, ref.HistoryID, "analysis"))

	finishedAt := time.Date(2026, 3, 25, 15, 20, 0, 0, time.UTC)
	require.NoError(t, env.repo.Fail(ctx, ref.HistoryID, "analysis", finishedAt))

	var history manualMailWorkflowHistoryRecord
	require.NoError(t, env.db.WithContext(ctx).First(&history, ref.HistoryID).Error)
	require.Equal(t, manualapp.WorkflowStatusFailed, history.Status)
	require.NotNil(t, history.CurrentStage)
	require.Equal(t, "analysis", *history.CurrentStage)
	require.NotNil(t, history.FinishedAt)
	require.True(t, history.FinishedAt.Equal(finishedAt))
}
