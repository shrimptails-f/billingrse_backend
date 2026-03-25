package infrastructure

import (
	"business/internal/library/logger"
	"business/internal/library/timewrapper"
	manualapp "business/internal/manualmailworkflow/application"
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

type manualMailWorkflowHistoryRecord struct {
	ID                                      uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	WorkflowID                              string     `gorm:"column:workflow_id;type:char(26);not null;uniqueIndex:uni_manual_mail_workflow_histories_workflow_id"`
	UserID                                  uint       `gorm:"column:user_id;not null;index:idx_manual_mail_workflow_histories_user_queued_at,priority:1;index:idx_manual_mail_workflow_histories_user_status_queued_at,priority:1"`
	ConnectionID                            uint       `gorm:"column:connection_id;not null"`
	LabelName                               string     `gorm:"column:label_name;size:255;not null"`
	SinceAt                                 time.Time  `gorm:"column:since_at;not null"`
	UntilAt                                 time.Time  `gorm:"column:until_at;not null"`
	Status                                  string     `gorm:"column:status;size:32;not null;index:idx_manual_mail_workflow_histories_user_status_queued_at,priority:2"`
	CurrentStage                            *string    `gorm:"column:current_stage;size:32"`
	QueuedAt                                time.Time  `gorm:"column:queued_at;not null;index:idx_manual_mail_workflow_histories_user_queued_at,priority:2;index:idx_manual_mail_workflow_histories_user_status_queued_at,priority:3"`
	FinishedAt                              *time.Time `gorm:"column:finished_at"`
	FetchSuccessCount                       int        `gorm:"column:fetch_success_count;not null;default:0"`
	FetchBusinessFailureCount               int        `gorm:"column:fetch_business_failure_count;not null;default:0"`
	FetchTechnicalFailureCount              int        `gorm:"column:fetch_technical_failure_count;not null;default:0"`
	AnalysisSuccessCount                    int        `gorm:"column:analysis_success_count;not null;default:0"`
	AnalysisBusinessFailureCount            int        `gorm:"column:analysis_business_failure_count;not null;default:0"`
	AnalysisTechnicalFailureCount           int        `gorm:"column:analysis_technical_failure_count;not null;default:0"`
	VendorResolutionSuccessCount            int        `gorm:"column:vendor_resolution_success_count;not null;default:0"`
	VendorResolutionBusinessFailureCount    int        `gorm:"column:vendor_resolution_business_failure_count;not null;default:0"`
	VendorResolutionTechnicalFailureCount   int        `gorm:"column:vendor_resolution_technical_failure_count;not null;default:0"`
	BillingEligibilitySuccessCount          int        `gorm:"column:billing_eligibility_success_count;not null;default:0"`
	BillingEligibilityBusinessFailureCount  int        `gorm:"column:billing_eligibility_business_failure_count;not null;default:0"`
	BillingEligibilityTechnicalFailureCount int        `gorm:"column:billing_eligibility_technical_failure_count;not null;default:0"`
	BillingSuccessCount                     int        `gorm:"column:billing_success_count;not null;default:0"`
	BillingBusinessFailureCount             int        `gorm:"column:billing_business_failure_count;not null;default:0"`
	BillingTechnicalFailureCount            int        `gorm:"column:billing_technical_failure_count;not null;default:0"`
	CreatedAt                               time.Time  `gorm:"column:created_at;not null"`
	UpdatedAt                               time.Time  `gorm:"column:updated_at;not null"`
}

func (manualMailWorkflowHistoryRecord) TableName() string {
	return "manual_mail_workflow_histories"
}

type manualMailWorkflowStageFailureRecord struct {
	WorkflowHistoryID uint64    `gorm:"column:workflow_history_id;not null;index:idx_manual_mail_workflow_stage_failures_history_stage_created_at,priority:1"`
	Stage             string    `gorm:"column:stage;size:32;not null;index:idx_manual_mail_workflow_stage_failures_history_stage_created_at,priority:2"`
	ExternalMessageID *string   `gorm:"column:external_message_id;size:255"`
	ReasonCode        string    `gorm:"column:reason_code;size:64;not null"`
	Message           string    `gorm:"column:message;size:255;not null"`
	CreatedAt         time.Time `gorm:"column:created_at;not null;index:idx_manual_mail_workflow_stage_failures_history_stage_created_at,priority:3"`
}

func (manualMailWorkflowStageFailureRecord) TableName() string {
	return "manual_mail_workflow_stage_failures"
}

// GormWorkflowStatusRepository persists workflow history rows into MySQL.
type GormWorkflowStatusRepository struct {
	db    *gorm.DB
	clock timewrapper.ClockInterface
	log   logger.Interface
}

// NewGormWorkflowStatusRepository creates a Gorm-backed workflow history repository.
func NewGormWorkflowStatusRepository(
	db *gorm.DB,
	clock timewrapper.ClockInterface,
	log logger.Interface,
) *GormWorkflowStatusRepository {
	if clock == nil {
		clock = timewrapper.NewClock()
	}
	if log == nil {
		log = logger.NewNop()
	}

	return &GormWorkflowStatusRepository{
		db:    db,
		clock: clock,
		log:   log.With(logger.Component("manual_mail_workflow_status_repository")),
	}
}

// CreateQueued inserts the accepted workflow header row.
func (r *GormWorkflowStatusRepository) CreateQueued(ctx context.Context, cmd manualapp.QueuedWorkflowHistory) (manualapp.WorkflowHistoryRef, error) {
	if ctx == nil {
		return manualapp.WorkflowHistoryRef{}, logger.ErrNilContext
	}
	if r.db == nil {
		return manualapp.WorkflowHistoryRef{}, fmt.Errorf("gorm db is not configured")
	}

	now := r.clock.Now().UTC()
	record := manualMailWorkflowHistoryRecord{
		WorkflowID:   strings.TrimSpace(cmd.WorkflowID),
		UserID:       cmd.UserID,
		ConnectionID: cmd.ConnectionID,
		LabelName:    strings.TrimSpace(cmd.LabelName),
		SinceAt:      cmd.SinceAt.UTC(),
		UntilAt:      cmd.UntilAt.UTC(),
		Status:       manualapp.WorkflowStatusQueued,
		QueuedAt:     cmd.QueuedAt.UTC(),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		r.logDBError(ctx, "manual_mail_workflow_histories", "create", err)
		return manualapp.WorkflowHistoryRef{}, fmt.Errorf("failed to create queued workflow history: %w", err)
	}

	return manualapp.WorkflowHistoryRef{
		HistoryID:  record.ID,
		WorkflowID: record.WorkflowID,
	}, nil
}

// MarkRunning updates the workflow status/current stage when background execution advances.
func (r *GormWorkflowStatusRepository) MarkRunning(ctx context.Context, historyID uint64, currentStage string) error {
	if ctx == nil {
		return logger.ErrNilContext
	}
	if r.db == nil {
		return fmt.Errorf("gorm db is not configured")
	}

	currentStage = strings.TrimSpace(currentStage)
	if currentStage == "" {
		return fmt.Errorf("current_stage is required")
	}

	now := r.clock.Now().UTC()
	tx := r.db.WithContext(ctx).
		Model(&manualMailWorkflowHistoryRecord{}).
		Where("id = ?", historyID).
		Updates(map[string]interface{}{
			"status":        manualapp.WorkflowStatusRunning,
			"current_stage": currentStage,
			"updated_at":    now,
		})
	if tx.Error != nil {
		r.logDBError(ctx, "manual_mail_workflow_histories", "mark_running", tx.Error)
		return fmt.Errorf("failed to mark workflow running: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

// SaveStageProgress persists one stage summary and its append-only failure rows.
func (r *GormWorkflowStatusRepository) SaveStageProgress(ctx context.Context, progress manualapp.StageProgress) error {
	if ctx == nil {
		return logger.ErrNilContext
	}
	if r.db == nil {
		return fmt.Errorf("gorm db is not configured")
	}

	successColumn, businessFailureColumn, technicalFailureColumn, err := stageCountColumns(progress.Stage)
	if err != nil {
		return err
	}
	totalFailureCount := progress.BusinessFailureCount + progress.TechnicalFailureCount
	if len(progress.FailureRecords) != totalFailureCount {
		return fmt.Errorf("failure records count mismatch: stage=%s records=%d business=%d technical=%d",
			progress.Stage,
			len(progress.FailureRecords),
			progress.BusinessFailureCount,
			progress.TechnicalFailureCount,
		)
	}

	now := r.clock.Now().UTC()
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updateResult := tx.Model(&manualMailWorkflowHistoryRecord{}).
			Where("id = ?", progress.HistoryID).
			Updates(map[string]interface{}{
				successColumn:          progress.SuccessCount,
				businessFailureColumn:  progress.BusinessFailureCount,
				technicalFailureColumn: progress.TechnicalFailureCount,
				"updated_at":           now,
			})
		if updateResult.Error != nil {
			return updateResult.Error
		}
		if updateResult.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		if len(progress.FailureRecords) == 0 {
			return nil
		}

		records := make([]manualMailWorkflowStageFailureRecord, 0, len(progress.FailureRecords))
		for _, failure := range progress.FailureRecords {
			records = append(records, manualMailWorkflowStageFailureRecord{
				WorkflowHistoryID: progress.HistoryID,
				Stage:             failure.Stage,
				ExternalMessageID: cloneOptionalString(failure.ExternalMessageID),
				ReasonCode:        failure.ReasonCode,
				Message:           failure.Message,
				CreatedAt:         now,
			})
		}

		return tx.Create(&records).Error
	})
	if err != nil {
		r.logDBError(ctx, "manual_mail_workflow_histories/manual_mail_workflow_stage_failures", "save_stage_progress", err)
		return fmt.Errorf("failed to save workflow stage progress: %w", err)
	}

	return nil
}

// Complete finalizes a workflow as succeeded or partial_success.
func (r *GormWorkflowStatusRepository) Complete(ctx context.Context, historyID uint64, status string, finishedAt time.Time) error {
	return r.updateTerminalStatus(ctx, historyID, status, nil, finishedAt, "complete")
}

// Fail finalizes a workflow as failed while preserving the failed stage when present.
func (r *GormWorkflowStatusRepository) Fail(ctx context.Context, historyID uint64, currentStage string, finishedAt time.Time) error {
	currentStage = strings.TrimSpace(currentStage)
	var stagePtr *string
	if currentStage != "" {
		stagePtr = &currentStage
	}
	return r.updateTerminalStatus(ctx, historyID, manualapp.WorkflowStatusFailed, stagePtr, finishedAt, "fail")
}

func (r *GormWorkflowStatusRepository) updateTerminalStatus(
	ctx context.Context,
	historyID uint64,
	status string,
	currentStage *string,
	finishedAt time.Time,
	operation string,
) error {
	if ctx == nil {
		return logger.ErrNilContext
	}
	if r.db == nil {
		return fmt.Errorf("gorm db is not configured")
	}

	now := r.clock.Now().UTC()
	finishedAt = finishedAt.UTC()

	tx := r.db.WithContext(ctx).
		Model(&manualMailWorkflowHistoryRecord{}).
		Where("id = ?", historyID).
		Updates(map[string]interface{}{
			"status":        status,
			"current_stage": currentStage,
			"finished_at":   &finishedAt,
			"updated_at":    now,
		})
	if tx.Error != nil {
		r.logDBError(ctx, "manual_mail_workflow_histories", operation, tx.Error)
		return fmt.Errorf("failed to %s workflow history: %w", operation, tx.Error)
	}
	if tx.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *GormWorkflowStatusRepository) logDBError(ctx context.Context, table string, operation string, err error) {
	reqLog := r.log
	if withContext, withCtxErr := r.log.WithContext(ctx); withCtxErr == nil {
		reqLog = withContext
	}

	reqLog.Error("db_query_failed",
		logger.String("db_system", "mysql"),
		logger.String("table", table),
		logger.String("operation", operation),
		logger.Err(err),
	)
}

func stageCountColumns(stage string) (string, string, string, error) {
	switch strings.TrimSpace(stage) {
	case "fetch":
		return "fetch_success_count", "fetch_business_failure_count", "fetch_technical_failure_count", nil
	case "analysis":
		return "analysis_success_count", "analysis_business_failure_count", "analysis_technical_failure_count", nil
	case "vendorresolution":
		return "vendor_resolution_success_count", "vendor_resolution_business_failure_count", "vendor_resolution_technical_failure_count", nil
	case "billingeligibility":
		return "billing_eligibility_success_count", "billing_eligibility_business_failure_count", "billing_eligibility_technical_failure_count", nil
	case "billing":
		return "billing_success_count", "billing_business_failure_count", "billing_technical_failure_count", nil
	default:
		return "", "", "", fmt.Errorf("unsupported workflow stage: %s", stage)
	}
}

func cloneOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
