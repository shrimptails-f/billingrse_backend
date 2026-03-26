package application

import (
	mfdomain "business/internal/mailfetch/domain"
	"context"
	"errors"
	"strings"
	"time"
)

const (
	// WorkflowStatusQueued indicates the workflow has been accepted for background execution.
	WorkflowStatusQueued = "queued"
	// WorkflowStatusRunning indicates the workflow is currently executing in background.
	WorkflowStatusRunning = "running"
	// WorkflowStatusSucceeded indicates the workflow finished without any stage failures.
	WorkflowStatusSucceeded = "succeeded"
	// WorkflowStatusPartialSuccess indicates the workflow finished with business/technical failures.
	WorkflowStatusPartialSuccess = "partial_success"
	// WorkflowStatusFailed indicates the workflow could not complete because of a top-level failure.
	WorkflowStatusFailed = "failed"
)

const (
	workflowStageFetch              = "fetch"
	workflowStageAnalysis           = "analysis"
	workflowStageVendorResolution   = "vendorresolution"
	workflowStageBillingEligibility = "billingeligibility"
	workflowStageBilling            = "billing"

	reasonCodeVendorUnresolved = "vendor_unresolved"
	reasonCodeDuplicateBilling = "duplicate_billing"
)

// WorkflowHistoryRef identifies a persisted workflow header row.
type WorkflowHistoryRef struct {
	HistoryID  uint64
	WorkflowID string
}

// QueuedWorkflowHistory is the header snapshot persisted when the workflow is accepted.
type QueuedWorkflowHistory struct {
	WorkflowID   string
	UserID       uint
	ConnectionID uint
	LabelName    string
	SinceAt      time.Time
	UntilAt      time.Time
	QueuedAt     time.Time
}

// StageFailureRecord is the append-only failure row persisted for one workflow stage.
type StageFailureRecord struct {
	Stage             string
	ExternalMessageID *string
	ReasonCode        string
	Message           string
}

// StageProgress is the per-stage summary persisted in the workflow header row.
type StageProgress struct {
	HistoryID             uint64
	Stage                 string
	SuccessCount          int
	BusinessFailureCount  int
	TechnicalFailureCount int
	FailureRecords        []StageFailureRecord
}

// WorkflowStatusRepository persists workflow header/failure rows.
type WorkflowStatusRepository interface {
	CreateQueued(ctx context.Context, cmd QueuedWorkflowHistory) (WorkflowHistoryRef, error)
	MarkRunning(ctx context.Context, historyID uint64, currentStage string) error
	SaveStageProgress(ctx context.Context, progress StageProgress) error
	Complete(ctx context.Context, historyID uint64, status string, finishedAt time.Time) error
	Fail(ctx context.Context, historyID uint64, currentStage string, finishedAt time.Time, errorMessage string) error
}

func localizedWorkflowErrorMessage(currentStage string, err error) string {
	if err == nil {
		return "メール取得ワークフローの実行に失敗しました。"
	}

	switch strings.TrimSpace(currentStage) {
	case "":
		return "メール取得ワークフローの起動に失敗しました。"
	case workflowStageFetch:
		return localizedFetchWorkflowErrorMessage(err)
	case workflowStageAnalysis:
		return "メール解析に失敗しました。"
	case workflowStageVendorResolution:
		return "支払先解決に失敗しました。"
	case workflowStageBillingEligibility:
		return "請求成立判定に失敗しました。"
	case workflowStageBilling:
		return "請求作成に失敗しました。"
	default:
		return "メール取得ワークフローの実行に失敗しました。"
	}
}

func localizedFetchWorkflowErrorMessage(err error) string {
	raw := strings.TrimSpace(err.Error())
	switch {
	case errors.Is(err, mfdomain.ErrConnectionNotFound):
		return "指定したGmail連携が見つかりませんでした。"
	case errors.Is(err, mfdomain.ErrConnectionUnavailable):
		return "指定したGmail連携は利用できません。再連携をおねがいします。"
	case errors.Is(err, mfdomain.ErrProviderUnsupported):
		return "指定したメール連携サービスには対応していません。"
	case errors.Is(err, mfdomain.ErrProviderLabelNotFound):
		return "指定したGmailラベルが見つかりませんでした。"
	case errors.Is(err, mfdomain.ErrProviderListFailed):
		return "Gmailからメール一覧を取得できませんでした。時間をおいて再試行してください。"
	case errors.Is(err, mfdomain.ErrProviderSessionBuildFailed):
		return localizedGmailSessionBuildErrorMessage(raw)
	case strings.Contains(raw, "failed to decrypt access token"),
		strings.Contains(raw, "failed to decrypt refresh token"),
		strings.Contains(raw, "failed to load gmail oauth config"),
		strings.Contains(raw, "failed to create gmail service"):
		return localizedGmailSessionBuildErrorMessage(raw)
	default:
		return "メール取得に失敗しました。"
	}
}

func localizedGmailSessionBuildErrorMessage(raw string) string {
	raw = strings.TrimSpace(raw)
	switch {
	case strings.Contains(raw, "failed to decrypt access token"):
		return "Gmail連携のアクセストークンを復号できませんでした。再連携をおねがいします。"
	case strings.Contains(raw, "failed to decrypt refresh token"):
		return "Gmail連携のリフレッシュトークンを復号できませんでした。再連携をおねがいします。"
	case strings.Contains(raw, "failed to load gmail oauth config"):
		return "Gmail OAuth設定の読み込みに失敗しました。システム設定を確認してください。"
	case strings.Contains(raw, "failed to create gmail service") && strings.Contains(raw, "invalid_grant"):
		return "Gmail連携が無効になっています。再連携してください。"
	case strings.Contains(raw, "failed to create gmail service"):
		return "Gmail連携の初期化に失敗しました。連携設定を確認してください。"
	default:
		return "Gmail連携の初期化に失敗しました。連携設定を確認してください。"
	}
}

func buildFetchStageProgress(historyID uint64, result FetchResult) StageProgress {
	failureRecords := make([]StageFailureRecord, 0, len(result.Failures))
	for _, failure := range result.Failures {
		failureRecords = append(failureRecords, stageFailureRecord(
			workflowStageFetch,
			failure.ExternalMessageID,
			failure.Code,
			stageMessageOrFallback(failure.Message, messageForFetchFailure(failure.Code)),
		))
	}

	return StageProgress{
		HistoryID:             historyID,
		Stage:                 workflowStageFetch,
		SuccessCount:          len(result.CreatedEmails) + len(result.ExistingEmailIDs),
		BusinessFailureCount:  0,
		TechnicalFailureCount: len(failureRecords),
		FailureRecords:        failureRecords,
	}
}

func buildAnalysisStageProgress(historyID uint64, result AnalyzeResult) StageProgress {
	failureRecords := make([]StageFailureRecord, 0, len(result.Failures))
	for _, failure := range result.Failures {
		failureRecords = append(failureRecords, stageFailureRecord(
			workflowStageAnalysis,
			failure.ExternalMessageID,
			failure.Code,
			stageMessageOrFallback(failure.Message, messageForAnalysisFailure(failure.Code)),
		))
	}

	return StageProgress{
		HistoryID:             historyID,
		Stage:                 workflowStageAnalysis,
		SuccessCount:          result.ParsedEmailCount,
		BusinessFailureCount:  0,
		TechnicalFailureCount: len(failureRecords),
		FailureRecords:        failureRecords,
	}
}

func buildVendorResolutionStageProgress(historyID uint64, inputs []ParsedEmail, result VendorResolutionResult) StageProgress {
	failureRecords := make([]StageFailureRecord, 0, result.UnresolvedCount+len(result.Failures))

	if len(result.UnresolvedItems) > 0 {
		for _, item := range result.UnresolvedItems {
			failureRecords = append(failureRecords, stageFailureRecord(
				workflowStageVendorResolution,
				item.ExternalMessageID,
				item.ReasonCode,
				stageMessageOrFallback(item.Message, messageForVendorResolutionFailure(item.ReasonCode)),
			))
		}
	} else {
		resolvedByParsedEmailID := make(map[uint]struct{}, len(result.ResolvedItems))
		for _, item := range result.ResolvedItems {
			resolvedByParsedEmailID[item.ParsedEmailID] = struct{}{}
		}

		failedByParsedEmailID := make(map[uint]struct{}, len(result.Failures))
		for _, failure := range result.Failures {
			if failure.ParsedEmailID != 0 {
				failedByParsedEmailID[failure.ParsedEmailID] = struct{}{}
			}
		}

		for _, input := range inputs {
			if _, resolved := resolvedByParsedEmailID[input.ParsedEmailID]; resolved {
				continue
			}
			if _, failed := failedByParsedEmailID[input.ParsedEmailID]; failed {
				continue
			}
			failureRecords = append(failureRecords, stageFailureRecord(
				workflowStageVendorResolution,
				input.ExternalMessageID,
				reasonCodeVendorUnresolved,
				inferredVendorUnresolvedMessage(input),
			))
		}
	}

	for _, failure := range result.Failures {
		failureRecords = append(failureRecords, stageFailureRecord(
			workflowStageVendorResolution,
			failure.ExternalMessageID,
			failure.Code,
			stageMessageOrFallback(failure.Message, messageForVendorResolutionFailure(failure.Code)),
		))
	}

	return StageProgress{
		HistoryID:             historyID,
		Stage:                 workflowStageVendorResolution,
		SuccessCount:          result.ResolvedCount,
		BusinessFailureCount:  len(failureRecords) - len(result.Failures),
		TechnicalFailureCount: len(result.Failures),
		FailureRecords:        failureRecords,
	}
}

func buildBillingEligibilityStageProgress(historyID uint64, result BillingEligibilityResult) StageProgress {
	failureRecords := make([]StageFailureRecord, 0, len(result.IneligibleItems)+len(result.Failures))
	for _, item := range result.IneligibleItems {
		failureRecords = append(failureRecords, stageFailureRecord(
			workflowStageBillingEligibility,
			item.ExternalMessageID,
			item.ReasonCode,
			stageMessageOrFallback(item.Message, messageForBillingEligibilityReason(item.ReasonCode)),
		))
	}
	for _, failure := range result.Failures {
		failureRecords = append(failureRecords, stageFailureRecord(
			workflowStageBillingEligibility,
			failure.ExternalMessageID,
			failure.Code,
			stageMessageOrFallback(failure.Message, messageForBillingEligibilityFailure(failure.Code)),
		))
	}

	return StageProgress{
		HistoryID:             historyID,
		Stage:                 workflowStageBillingEligibility,
		SuccessCount:          result.EligibleCount,
		BusinessFailureCount:  result.IneligibleCount,
		TechnicalFailureCount: len(result.Failures),
		FailureRecords:        failureRecords,
	}
}

func buildBillingStageProgress(historyID uint64, result BillingResult) StageProgress {
	failureRecords := make([]StageFailureRecord, 0, len(result.DuplicateItems)+len(result.Failures))
	for _, item := range result.DuplicateItems {
		reasonCode := item.ReasonCode
		if reasonCode == "" {
			reasonCode = reasonCodeDuplicateBilling
		}
		failureRecords = append(failureRecords, stageFailureRecord(
			workflowStageBilling,
			item.ExternalMessageID,
			reasonCode,
			stageMessageOrFallback(item.Message, messageForBillingFailure(reasonCode)),
		))
	}
	for _, failure := range result.Failures {
		failureRecords = append(failureRecords, stageFailureRecord(
			workflowStageBilling,
			failure.ExternalMessageID,
			failure.Code,
			stageMessageOrFallback(failure.Message, messageForBillingFailure(failure.Code)),
		))
	}

	return StageProgress{
		HistoryID:             historyID,
		Stage:                 workflowStageBilling,
		SuccessCount:          result.CreatedCount,
		BusinessFailureCount:  len(result.DuplicateItems),
		TechnicalFailureCount: len(result.Failures),
		FailureRecords:        failureRecords,
	}
}

func workflowStatusForResult(result Result) string {
	if hasWorkflowFailures(result) {
		return WorkflowStatusPartialSuccess
	}
	return WorkflowStatusSucceeded
}

func hasWorkflowFailures(result Result) bool {
	return len(result.Fetch.Failures) > 0 ||
		len(result.Analysis.Failures) > 0 ||
		result.VendorResolution.UnresolvedCount+len(result.VendorResolution.Failures) > 0 ||
		result.BillingEligibility.IneligibleCount+len(result.BillingEligibility.Failures) > 0 ||
		result.Billing.DuplicateCount+len(result.Billing.Failures) > 0
}

func stageFailureRecord(stage string, externalMessageID string, reasonCode string, message string) StageFailureRecord {
	record := StageFailureRecord{
		Stage:      stage,
		ReasonCode: reasonCode,
		Message:    message,
	}
	if externalMessageID != "" {
		record.ExternalMessageID = workflowStringPtr(externalMessageID)
	}
	return record
}

func workflowStringPtr(value string) *string {
	return &value
}

func stageMessageOrFallback(message string, fallback string) string {
	if message != "" {
		return message
	}
	return fallback
}

func inferredVendorUnresolvedMessage(input ParsedEmail) string {
	candidateVendorName := ""
	if input.Data.VendorName != nil {
		candidateVendorName = *input.Data.VendorName
	}
	if candidateVendorName == "" {
		return externalMessageIDText(input.ExternalMessageID) + " の支払先を特定できませんでした。"
	}
	return externalMessageIDText(input.ExternalMessageID) + " の候補「" + candidateVendorName + "」を支払先として特定できませんでした。"
}

func externalMessageIDText(externalMessageID string) string {
	if externalMessageID == "" {
		return "不明なメッセージ"
	}
	return externalMessageID
}

func messageForFetchFailure(code string) string {
	switch code {
	case "fetch_detail_failed":
		return "メールの取得に失敗しました。"
	case "invalid_fetched_email":
		return "取得したメールの形式が不正でした。"
	case "duplicate_external_message_id":
		return "取得結果に重複したメールIDが含まれていました。"
	case "email_save_failed":
		return "取得したメールの保存に失敗しました。"
	default:
		return "メール取得中にエラーが発生しました。"
	}
}

func messageForAnalysisFailure(code string) string {
	switch code {
	case "invalid_email_input":
		return "解析対象メールの入力が不正でした。"
	case "analysis_failed":
		return "メール解析に失敗しました。"
	case "analysis_response_invalid":
		return "解析結果の形式が不正でした。"
	case "analysis_response_empty":
		return "解析結果を抽出できませんでした。"
	case "parsed_email_save_failed":
		return "解析結果の保存に失敗しました。"
	default:
		return "メール解析中にエラーが発生しました。"
	}
}

func messageForVendorResolutionFailure(code string) string {
	switch code {
	case reasonCodeVendorUnresolved:
		return "支払先を特定できませんでした。"
	case "invalid_resolution_target":
		return "支払先解決の入力が不正でした。"
	case "vendor_resolution_failed":
		return "支払先の解決に失敗しました。"
	case "vendor_registration_failed":
		return "支払先の登録に失敗しました。"
	default:
		return "支払先解決中にエラーが発生しました。"
	}
}

func messageForBillingEligibilityReason(reasonCode string) string {
	switch reasonCode {
	case "product_name_empty":
		return "商品名が不足しているため請求を作成できませんでした。"
	case "amount_empty":
		return "金額が不足しているため請求を作成できませんでした。"
	case "amount_invalid":
		return "金額が不正なため請求を作成できませんでした。"
	case "currency_empty":
		return "通貨が不足しているため請求を作成できませんでした。"
	case "currency_invalid":
		return "通貨が不正なため請求を作成できませんでした。"
	case "billing_number_empty":
		return "請求番号が不足しているため請求を作成できませんでした。"
	case "payment_cycle_empty":
		return "支払周期が不足しているため請求を作成できませんでした。"
	case "payment_cycle_invalid":
		return "支払周期が不正なため請求を作成できませんでした。"
	default:
		return "請求成立条件を満たさないため請求を作成できませんでした。"
	}
}

func messageForBillingEligibilityFailure(code string) string {
	switch code {
	case "invalid_eligibility_target":
		return "請求成立判定の入力が不正でした。"
	case "billing_eligibility_failed":
		return "請求成立判定に失敗しました。"
	default:
		return "請求成立判定中にエラーが発生しました。"
	}
}

func messageForBillingFailure(code string) string {
	switch code {
	case reasonCodeDuplicateBilling:
		return "同じ請求番号の請求が既に存在します。"
	case "invalid_creation_target":
		return "請求作成の入力が不正でした。"
	case "billing_construct_failed":
		return "請求データの生成に失敗しました。"
	case "billing_persist_failed":
		return "請求の保存に失敗しました。"
	default:
		return "請求作成中にエラーが発生しました。"
	}
}
