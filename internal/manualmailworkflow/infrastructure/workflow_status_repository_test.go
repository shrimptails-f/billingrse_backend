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
	require.NoError(t, mysqlConn.DB.AutoMigrate(
		&emailCredentialSnapshotRecord{},
		&manualMailWorkflowHistoryRecord{},
		&manualMailWorkflowStageFailureRecord{},
	))

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
	mustCreateCredentialSnapshot(t, env.db, emailCredentialSnapshotRecord{
		ID:           20,
		UserID:       10,
		Type:         "gmail",
		GmailAddress: "billing@example.com",
	})
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
	require.Equal(t, "gmail", history.Provider)
	require.Equal(t, "billing@example.com", history.AccountIdentifier)
	require.Equal(t, "billing", history.LabelName)
	require.Equal(t, manualapp.WorkflowStatusPartialSuccess, history.Status)
	require.Nil(t, history.CurrentStage)
	require.True(t, history.QueuedAt.Equal(queuedAt))
	require.NotNil(t, history.FinishedAt)
	require.True(t, history.FinishedAt.Equal(finishedAt))
	require.Nil(t, history.ErrorMessage)
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

func TestGormWorkflowStatusRepository_SaveStageProgress_AllowsNonCountedSkipRecord(t *testing.T) {
	t.Parallel()

	env := newWorkflowStatusRepoTestEnv(t)
	defer env.clean()

	ctx := context.Background()
	mustCreateCredentialSnapshot(t, env.db, emailCredentialSnapshotRecord{
		ID:           22,
		UserID:       12,
		Type:         "gmail",
		GmailAddress: "skip@example.com",
	})
	ref, err := env.repo.CreateQueued(ctx, manualapp.QueuedWorkflowHistory{
		WorkflowID:   "01JQ0B7N0M7H3X9C2J5K8V6P7",
		UserID:       12,
		ConnectionID: 22,
		LabelName:    "billing",
		SinceAt:      time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
		UntilAt:      time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
		QueuedAt:     time.Date(2026, 3, 25, 14, 56, 0, 0, time.UTC),
	})
	require.NoError(t, err)

	require.NoError(t, env.repo.SaveStageProgress(ctx, manualapp.StageProgress{
		HistoryID:             ref.HistoryID,
		Stage:                 "fetch",
		SuccessCount:          2,
		BusinessFailureCount:  0,
		TechnicalFailureCount: 0,
		FailureRecords: []manualapp.StageFailureRecord{
			{
				Stage:      "fetch",
				ReasonCode: "existing_emails_skipped",
				Message:    "対象メールはすでに取得済みだったため、後続の処理をスキップしました。",
			},
		},
	}))

	var history manualMailWorkflowHistoryRecord
	require.NoError(t, env.db.WithContext(ctx).First(&history, ref.HistoryID).Error)
	require.Equal(t, 2, history.FetchSuccessCount)
	require.Equal(t, 0, history.FetchBusinessFailureCount)
	require.Equal(t, 0, history.FetchTechnicalFailureCount)

	var failures []manualMailWorkflowStageFailureRecord
	require.NoError(t, env.db.WithContext(ctx).
		Where("workflow_history_id = ?", ref.HistoryID).
		Find(&failures).Error)
	require.Len(t, failures, 1)
	require.Equal(t, "existing_emails_skipped", failures[0].ReasonCode)
	require.Equal(t, "対象メールはすでに取得済みだったため、後続の処理をスキップしました。", failures[0].Message)
}

func TestGormWorkflowStatusRepository_Fail(t *testing.T) {
	t.Parallel()

	env := newWorkflowStatusRepoTestEnv(t)
	defer env.clean()

	ctx := context.Background()
	mustCreateCredentialSnapshot(t, env.db, emailCredentialSnapshotRecord{
		ID:           2,
		UserID:       1,
		Type:         "gmail",
		GmailAddress: "failed@example.com",
	})
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
	require.NoError(t, env.repo.Fail(ctx, ref.HistoryID, "analysis", finishedAt, "failed to create gmail service: invalid_grant"))

	var history manualMailWorkflowHistoryRecord
	require.NoError(t, env.db.WithContext(ctx).First(&history, ref.HistoryID).Error)
	require.Equal(t, manualapp.WorkflowStatusFailed, history.Status)
	require.NotNil(t, history.CurrentStage)
	require.Equal(t, "analysis", *history.CurrentStage)
	require.NotNil(t, history.FinishedAt)
	require.True(t, history.FinishedAt.Equal(finishedAt))
	require.NotNil(t, history.ErrorMessage)
	require.Equal(t, "failed to create gmail service: invalid_grant", *history.ErrorMessage)
}

func TestGormWorkflowStatusRepository_List(t *testing.T) {
	t.Parallel()

	env := newWorkflowStatusRepoTestEnv(t)
	defer env.clean()

	ctx := context.Background()
	queuedAt := time.Date(2026, 3, 25, 14, 0, 0, 0, time.UTC)
	firstHistory := workflowHistoryRecordFixture(10, "wf-first", queuedAt, manualapp.WorkflowStatusPartialSuccess)
	firstHistory.Provider = "gmail"
	firstHistory.AccountIdentifier = "first@example.com"
	firstHistory.ErrorMessage = stringPtr("Gmail連携が無効になっています。再連携してください。")
	firstHistory.FetchSuccessCount = 2
	firstHistory.FetchTechnicalFailureCount = 2
	firstHistory.VendorResolutionSuccessCount = 1
	firstHistory.VendorResolutionBusinessFailureCount = 1

	secondHistory := workflowHistoryRecordFixture(10, "wf-second", queuedAt, manualapp.WorkflowStatusSucceeded)
	secondHistory.Provider = "gmail"
	secondHistory.AccountIdentifier = "second@example.com"

	otherUserHistory := workflowHistoryRecordFixture(20, "wf-other-user", queuedAt.Add(time.Hour), manualapp.WorkflowStatusFailed)
	otherUserHistory.Provider = "gmail"
	otherUserHistory.AccountIdentifier = "hidden@example.com"

	require.NoError(t, env.db.WithContext(ctx).Create(&firstHistory).Error)
	require.NoError(t, env.db.WithContext(ctx).Create(&secondHistory).Error)
	require.NoError(t, env.db.WithContext(ctx).Create(&otherUserHistory).Error)

	failures := []manualMailWorkflowStageFailureRecord{
		{
			WorkflowHistoryID: firstHistory.ID,
			Stage:             "fetch",
			ExternalMessageID: stringPtr("msg-1"),
			ReasonCode:        "fetch_detail_failed",
			Message:           "メールの取得に失敗しました。",
			CreatedAt:         queuedAt.Add(2 * time.Minute),
		},
		{
			WorkflowHistoryID: firstHistory.ID,
			Stage:             "fetch",
			ExternalMessageID: stringPtr("msg-2"),
			ReasonCode:        "email_save_failed",
			Message:           "メールの保存に失敗しました。",
			CreatedAt:         queuedAt.Add(3 * time.Minute),
		},
		{
			WorkflowHistoryID: firstHistory.ID,
			Stage:             "vendorresolution",
			ExternalMessageID: stringPtr("msg-3"),
			ReasonCode:        "vendor_unresolved",
			Message:           "支払先を特定できませんでした。",
			CreatedAt:         queuedAt.Add(4 * time.Minute),
		},
		{
			WorkflowHistoryID: otherUserHistory.ID,
			Stage:             "billing",
			ExternalMessageID: stringPtr("msg-hidden"),
			ReasonCode:        "duplicate_billing",
			Message:           "同じ請求番号の請求が既に存在します。",
			CreatedAt:         queuedAt.Add(5 * time.Minute),
		},
	}
	require.NoError(t, env.db.WithContext(ctx).Create(&failures).Error)

	queryCount := 0
	callbackName := "test:manual_mail_workflow_list_query_count"
	require.NoError(t, env.db.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		queryCount++
	}))
	defer func() {
		_ = env.db.Callback().Query().Remove(callbackName)
	}()

	result, err := env.repo.List(ctx, manualapp.ListQuery{
		UserID:    10,
		Limit:     10,
		Offset:    0,
		HasLimit:  true,
		HasOffset: true,
	})
	require.NoError(t, err)
	require.EqualValues(t, 2, result.TotalCount)
	require.Len(t, result.Items, 2)
	require.Equal(t, 3, queryCount)

	require.Equal(t, "wf-second", result.Items[0].WorkflowID)
	require.Equal(t, "wf-first", result.Items[1].WorkflowID)

	firstItem := result.Items[1]
	require.Equal(t, "gmail", firstItem.Provider)
	require.Equal(t, "first@example.com", firstItem.AccountIdentifier)
	require.NotNil(t, firstItem.ErrorMessage)
	require.Equal(t, "Gmail連携が無効になっています。再連携してください。", *firstItem.ErrorMessage)
	require.Equal(t, 2, firstItem.Fetch.SuccessCount)
	require.Equal(t, 0, firstItem.Fetch.BusinessFailureCount)
	require.Equal(t, 2, firstItem.Fetch.TechnicalFailureCount)
	require.Len(t, firstItem.Fetch.Failures, 2)
	require.Equal(t, "msg-1", *firstItem.Fetch.Failures[0].ExternalMessageID)
	require.Equal(t, "fetch_detail_failed", firstItem.Fetch.Failures[0].ReasonCode)
	require.Equal(t, "msg-2", *firstItem.Fetch.Failures[1].ExternalMessageID)
	require.Len(t, firstItem.VendorResolution.Failures, 1)
	require.Equal(t, "vendor_unresolved", firstItem.VendorResolution.Failures[0].ReasonCode)
	require.Empty(t, firstItem.Analysis.Failures)
	require.Empty(t, firstItem.BillingEligibility.Failures)
	require.Empty(t, firstItem.Billing.Failures)
}

func TestGormWorkflowStatusRepository_List_WithStatusFilterAndEmptyPage(t *testing.T) {
	t.Parallel()

	env := newWorkflowStatusRepoTestEnv(t)
	defer env.clean()

	ctx := context.Background()
	firstHistory := workflowHistoryRecordFixture(10, "wf-partial", time.Date(2026, 3, 25, 14, 0, 0, 0, time.UTC), manualapp.WorkflowStatusPartialSuccess)
	secondHistory := workflowHistoryRecordFixture(10, "wf-succeeded", time.Date(2026, 3, 25, 13, 0, 0, 0, time.UTC), manualapp.WorkflowStatusSucceeded)
	require.NoError(t, env.db.WithContext(ctx).Create(&firstHistory).Error)
	require.NoError(t, env.db.WithContext(ctx).Create(&secondHistory).Error)

	filtered, err := env.repo.List(ctx, manualapp.ListQuery{
		UserID:    10,
		Limit:     10,
		Offset:    0,
		Status:    stringPtr(manualapp.WorkflowStatusPartialSuccess),
		HasLimit:  true,
		HasOffset: true,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, filtered.TotalCount)
	require.Len(t, filtered.Items, 1)
	require.Equal(t, "wf-partial", filtered.Items[0].WorkflowID)

	queryCount := 0
	callbackName := "test:manual_mail_workflow_list_empty_page_query_count"
	require.NoError(t, env.db.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		queryCount++
	}))
	defer func() {
		_ = env.db.Callback().Query().Remove(callbackName)
	}()

	emptyPage, err := env.repo.List(ctx, manualapp.ListQuery{
		UserID:    10,
		Limit:     10,
		Offset:    10,
		HasLimit:  true,
		HasOffset: true,
	})
	require.NoError(t, err)
	require.EqualValues(t, 2, emptyPage.TotalCount)
	require.Empty(t, emptyPage.Items)
	require.Equal(t, 2, queryCount)
}

func workflowHistoryRecordFixture(userID uint, workflowID string, queuedAt time.Time, status string) manualMailWorkflowHistoryRecord {
	return manualMailWorkflowHistoryRecord{
		WorkflowID:        workflowID,
		UserID:            userID,
		Provider:          "gmail",
		AccountIdentifier: "billing@example.com",
		LabelName:         "billing",
		SinceAt:           time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
		UntilAt:           time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
		Status:            status,
		QueuedAt:          queuedAt,
		CreatedAt:         queuedAt,
		UpdatedAt:         queuedAt,
	}
}

func mustCreateCredentialSnapshot(t *testing.T, db *gorm.DB, record emailCredentialSnapshotRecord) {
	t.Helper()
	require.NoError(t, db.Create(&record).Error)
}

func stringPtr(value string) *string {
	return &value
}
