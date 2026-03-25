package seeders

import (
	model "business/tools/migrations/models"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

type workflowHistorySeed struct {
	WorkflowID                              string
	UserEmail                               string
	GmailAddress                            string
	LabelName                               string
	SinceAt                                 time.Time
	UntilAt                                 time.Time
	Status                                  string
	CurrentStage                            *string
	QueuedAt                                time.Time
	FinishedAt                              *time.Time
	FetchSuccessCount                       int
	FetchBusinessFailureCount               int
	FetchTechnicalFailureCount              int
	AnalysisSuccessCount                    int
	AnalysisBusinessFailureCount            int
	AnalysisTechnicalFailureCount           int
	VendorResolutionSuccessCount            int
	VendorResolutionBusinessFailureCount    int
	VendorResolutionTechnicalFailureCount   int
	BillingEligibilitySuccessCount          int
	BillingEligibilityBusinessFailureCount  int
	BillingEligibilityTechnicalFailureCount int
	BillingSuccessCount                     int
	BillingBusinessFailureCount             int
	BillingTechnicalFailureCount            int
	Failures                                []workflowStageFailureSeed
}

type workflowStageFailureSeed struct {
	Stage             string
	ExternalMessageID *string
	ReasonCode        string
	Message           string
	CreatedAt         time.Time
}

// CreateManualMailWorkflowHistories は一覧確認用の workflow 履歴サンプルを投入する。
func CreateManualMailWorkflowHistories(tx *gorm.DB) error {
	if tx == nil {
		return fmt.Errorf("gorm db is not configured")
	}

	stageAnalysis := "analysis"
	finishedSucceeded := time.Date(2026, 3, 25, 8, 6, 0, 0, time.UTC)
	finishedPartial := time.Date(2026, 3, 25, 9, 7, 0, 0, time.UTC)

	seeds := []workflowHistorySeed{
		{
			WorkflowID:                     "01ARZ3NDEKTSV4RRFFQ69G5FAV",
			UserEmail:                      adminUserEmail,
			GmailAddress:                   "admin.billing@gmail.com",
			LabelName:                      "billing",
			SinceAt:                        time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			UntilAt:                        time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
			Status:                         "succeeded",
			QueuedAt:                       time.Date(2026, 3, 25, 8, 0, 0, 0, time.UTC),
			FinishedAt:                     &finishedSucceeded,
			FetchSuccessCount:              2,
			AnalysisSuccessCount:           2,
			VendorResolutionSuccessCount:   2,
			BillingEligibilitySuccessCount: 2,
			BillingSuccessCount:            2,
		},
		{
			WorkflowID:                             "01ARZ3NDEKTSV4RRFFQ69G5FB0",
			UserEmail:                              testUserEmail,
			GmailAddress:                           "test.billing@gmail.com",
			LabelName:                              "billing",
			SinceAt:                                time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
			UntilAt:                                time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
			Status:                                 "partial_success",
			QueuedAt:                               time.Date(2026, 3, 25, 9, 0, 0, 0, time.UTC),
			FinishedAt:                             &finishedPartial,
			FetchSuccessCount:                      4,
			AnalysisSuccessCount:                   4,
			VendorResolutionSuccessCount:           3,
			VendorResolutionBusinessFailureCount:   1,
			BillingEligibilitySuccessCount:         2,
			BillingEligibilityBusinessFailureCount: 1,
			BillingSuccessCount:                    1,
			BillingBusinessFailureCount:            1,
			Failures: []workflowStageFailureSeed{
				{
					Stage:             "vendorresolution",
					ExternalMessageID: stringPtr("seed-test-slack-001"),
					ReasonCode:        "vendor_unresolved",
					Message:           "seed-test-slack-001 の候補「Slack」を支払先として特定できませんでした。",
					CreatedAt:         time.Date(2026, 3, 25, 9, 4, 0, 0, time.UTC),
				},
				{
					Stage:             "billingeligibility",
					ExternalMessageID: stringPtr("seed-test-notion-001"),
					ReasonCode:        "amount_invalid",
					Message:           "金額が不正なため請求を作成できませんでした。",
					CreatedAt:         time.Date(2026, 3, 25, 9, 5, 0, 0, time.UTC),
				},
				{
					Stage:             "billing",
					ExternalMessageID: stringPtr("seed-test-aws-001"),
					ReasonCode:        "duplicate_billing",
					Message:           "同じ請求番号の請求が既に存在します。",
					CreatedAt:         time.Date(2026, 3, 25, 9, 6, 0, 0, time.UTC),
				},
			},
		},
		{
			WorkflowID:        "01ARZ3NDEKTSV4RRFFQ69G5FB1",
			UserEmail:         adminUserEmail,
			GmailAddress:      "admin.subscriptions@gmail.com",
			LabelName:         "subscriptions",
			SinceAt:           time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC),
			UntilAt:           time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
			Status:            "running",
			CurrentStage:      &stageAnalysis,
			QueuedAt:          time.Date(2026, 3, 25, 10, 0, 0, 0, time.UTC),
			FetchSuccessCount: 5,
		},
		{
			WorkflowID:   "01ARZ3NDEKTSV4RRFFQ69G5FB2",
			UserEmail:    testUserEmail,
			GmailAddress: "test.billing@gmail.com",
			LabelName:    "billing",
			SinceAt:      time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC),
			UntilAt:      time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
			Status:       "queued",
			QueuedAt:     time.Date(2026, 3, 25, 11, 0, 0, 0, time.UTC),
		},
	}

	for _, seed := range seeds {
		user, err := findUserByEmail(tx, seed.UserEmail)
		if err != nil {
			return err
		}

		credential, err := findCredentialByUserAndGmail(tx, user.ID, seed.GmailAddress)
		if err != nil {
			return err
		}

		record := model.ManualMailWorkflowHistory{
			WorkflowID:                              seed.WorkflowID,
			UserID:                                  user.ID,
			ConnectionID:                            credential.ID,
			LabelName:                               seed.LabelName,
			SinceAt:                                 seed.SinceAt.UTC(),
			UntilAt:                                 seed.UntilAt.UTC(),
			Status:                                  strings.TrimSpace(seed.Status),
			CurrentStage:                            cloneSeedString(seed.CurrentStage),
			QueuedAt:                                seed.QueuedAt.UTC(),
			FinishedAt:                              cloneSeedTime(seed.FinishedAt),
			FetchSuccessCount:                       seed.FetchSuccessCount,
			FetchBusinessFailureCount:               seed.FetchBusinessFailureCount,
			FetchTechnicalFailureCount:              seed.FetchTechnicalFailureCount,
			AnalysisSuccessCount:                    seed.AnalysisSuccessCount,
			AnalysisBusinessFailureCount:            seed.AnalysisBusinessFailureCount,
			AnalysisTechnicalFailureCount:           seed.AnalysisTechnicalFailureCount,
			VendorResolutionSuccessCount:            seed.VendorResolutionSuccessCount,
			VendorResolutionBusinessFailureCount:    seed.VendorResolutionBusinessFailureCount,
			VendorResolutionTechnicalFailureCount:   seed.VendorResolutionTechnicalFailureCount,
			BillingEligibilitySuccessCount:          seed.BillingEligibilitySuccessCount,
			BillingEligibilityBusinessFailureCount:  seed.BillingEligibilityBusinessFailureCount,
			BillingEligibilityTechnicalFailureCount: seed.BillingEligibilityTechnicalFailureCount,
			BillingSuccessCount:                     seed.BillingSuccessCount,
			BillingBusinessFailureCount:             seed.BillingBusinessFailureCount,
			BillingTechnicalFailureCount:            seed.BillingTechnicalFailureCount,
			CreatedAt:                               seed.QueuedAt.UTC(),
			UpdatedAt:                               seed.QueuedAt.UTC(),
		}

		historyID, err := upsertWorkflowHistoryByWorkflowID(tx, record)
		if err != nil {
			return err
		}
		if err := replaceWorkflowStageFailures(tx, historyID, seed.Failures); err != nil {
			return err
		}
	}

	return nil
}

func upsertWorkflowHistoryByWorkflowID(tx *gorm.DB, record model.ManualMailWorkflowHistory) (uint64, error) {
	var existing model.ManualMailWorkflowHistory
	err := tx.Where("workflow_id = ?", record.WorkflowID).First(&existing).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		if err := tx.Create(&record).Error; err != nil {
			return 0, fmt.Errorf("failed to create workflow history %s: %w", record.WorkflowID, err)
		}
		return record.ID, nil
	case err != nil:
		return 0, fmt.Errorf("failed to find workflow history %s: %w", record.WorkflowID, err)
	default:
		if err := tx.Model(&existing).Updates(map[string]interface{}{
			"user_id":                                     record.UserID,
			"connection_id":                               record.ConnectionID,
			"label_name":                                  record.LabelName,
			"since_at":                                    record.SinceAt,
			"until_at":                                    record.UntilAt,
			"status":                                      record.Status,
			"current_stage":                               record.CurrentStage,
			"queued_at":                                   record.QueuedAt,
			"finished_at":                                 record.FinishedAt,
			"fetch_success_count":                         record.FetchSuccessCount,
			"fetch_business_failure_count":                record.FetchBusinessFailureCount,
			"fetch_technical_failure_count":               record.FetchTechnicalFailureCount,
			"analysis_success_count":                      record.AnalysisSuccessCount,
			"analysis_business_failure_count":             record.AnalysisBusinessFailureCount,
			"analysis_technical_failure_count":            record.AnalysisTechnicalFailureCount,
			"vendor_resolution_success_count":             record.VendorResolutionSuccessCount,
			"vendor_resolution_business_failure_count":    record.VendorResolutionBusinessFailureCount,
			"vendor_resolution_technical_failure_count":   record.VendorResolutionTechnicalFailureCount,
			"billing_eligibility_success_count":           record.BillingEligibilitySuccessCount,
			"billing_eligibility_business_failure_count":  record.BillingEligibilityBusinessFailureCount,
			"billing_eligibility_technical_failure_count": record.BillingEligibilityTechnicalFailureCount,
			"billing_success_count":                       record.BillingSuccessCount,
			"billing_business_failure_count":              record.BillingBusinessFailureCount,
			"billing_technical_failure_count":             record.BillingTechnicalFailureCount,
			"updated_at":                                  record.UpdatedAt,
		}).Error; err != nil {
			return 0, fmt.Errorf("failed to update workflow history %s: %w", record.WorkflowID, err)
		}
		return existing.ID, nil
	}
}

func replaceWorkflowStageFailures(tx *gorm.DB, workflowHistoryID uint64, seeds []workflowStageFailureSeed) error {
	if err := tx.Where("workflow_history_id = ?", workflowHistoryID).Delete(&model.ManualMailWorkflowStageFailure{}).Error; err != nil {
		return fmt.Errorf("failed to delete workflow stage failures for history_id=%d: %w", workflowHistoryID, err)
	}
	if len(seeds) == 0 {
		return nil
	}

	records := make([]model.ManualMailWorkflowStageFailure, 0, len(seeds))
	for _, seed := range seeds {
		records = append(records, model.ManualMailWorkflowStageFailure{
			WorkflowHistoryID: workflowHistoryID,
			Stage:             seed.Stage,
			ExternalMessageID: cloneSeedString(seed.ExternalMessageID),
			ReasonCode:        seed.ReasonCode,
			Message:           seed.Message,
			CreatedAt:         seed.CreatedAt.UTC(),
		})
	}

	if err := tx.Create(&records).Error; err != nil {
		return fmt.Errorf("failed to create workflow stage failures for history_id=%d: %w", workflowHistoryID, err)
	}

	return nil
}
