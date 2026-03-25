package model

import "time"

// ManualMailWorkflowHistory represents the manual_mail_workflow_histories table.
type ManualMailWorkflowHistory struct {
	ID                                      uint64    `gorm:"primaryKey;autoIncrement"`
	WorkflowID                              string    `gorm:"type:char(26);not null;uniqueIndex:uni_manual_mail_workflow_histories_workflow_id"`
	UserID                                  uint      `gorm:"not null;index:idx_manual_mail_workflow_histories_user_queued_at,priority:1;index:idx_manual_mail_workflow_histories_user_status_queued_at,priority:1"`
	Provider                                string    `gorm:"size:50;not null"`
	AccountIdentifier                       string    `gorm:"size:255;not null"`
	LabelName                               string    `gorm:"size:255;not null"`
	SinceAt                                 time.Time `gorm:"not null"`
	UntilAt                                 time.Time `gorm:"not null"`
	Status                                  string    `gorm:"size:32;not null;index:idx_manual_mail_workflow_histories_user_status_queued_at,priority:2"`
	CurrentStage                            *string   `gorm:"size:32"`
	QueuedAt                                time.Time `gorm:"not null;index:idx_manual_mail_workflow_histories_user_queued_at,priority:2;index:idx_manual_mail_workflow_histories_user_status_queued_at,priority:3"`
	FinishedAt                              *time.Time
	ErrorMessage                            *string `gorm:"type:text"`
	FetchSuccessCount                       int     `gorm:"not null;default:0"`
	FetchBusinessFailureCount               int     `gorm:"not null;default:0"`
	FetchTechnicalFailureCount              int     `gorm:"not null;default:0"`
	AnalysisSuccessCount                    int     `gorm:"not null;default:0"`
	AnalysisBusinessFailureCount            int     `gorm:"not null;default:0"`
	AnalysisTechnicalFailureCount           int     `gorm:"not null;default:0"`
	VendorResolutionSuccessCount            int     `gorm:"not null;default:0"`
	VendorResolutionBusinessFailureCount    int     `gorm:"not null;default:0"`
	VendorResolutionTechnicalFailureCount   int     `gorm:"not null;default:0"`
	BillingEligibilitySuccessCount          int     `gorm:"not null;default:0"`
	BillingEligibilityBusinessFailureCount  int     `gorm:"not null;default:0"`
	BillingEligibilityTechnicalFailureCount int     `gorm:"not null;default:0"`
	BillingSuccessCount                     int     `gorm:"not null;default:0"`
	BillingBusinessFailureCount             int     `gorm:"not null;default:0"`
	BillingTechnicalFailureCount            int     `gorm:"not null;default:0"`
	CreatedAt                               time.Time
	UpdatedAt                               time.Time
}

// TableName specifies the table name for the ManualMailWorkflowHistory model.
func (ManualMailWorkflowHistory) TableName() string {
	return "manual_mail_workflow_histories"
}

// ManualMailWorkflowStageFailure represents the manual_mail_workflow_stage_failures table.
type ManualMailWorkflowStageFailure struct {
	WorkflowHistoryID uint64    `gorm:"not null;index:idx_manual_mail_workflow_stage_failures_history_stage_created_at,priority:1"`
	Stage             string    `gorm:"size:32;not null;index:idx_manual_mail_workflow_stage_failures_history_stage_created_at,priority:2"`
	ExternalMessageID *string   `gorm:"size:255"`
	ReasonCode        string    `gorm:"size:64;not null"`
	Message           string    `gorm:"size:255;not null"`
	CreatedAt         time.Time `gorm:"not null;index:idx_manual_mail_workflow_stage_failures_history_stage_created_at,priority:3"`
}

// TableName specifies the table name for the ManualMailWorkflowStageFailure model.
func (ManualMailWorkflowStageFailure) TableName() string {
	return "manual_mail_workflow_stage_failures"
}
