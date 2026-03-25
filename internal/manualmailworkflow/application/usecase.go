package application

import (
	commondomain "business/internal/common/domain"
	"business/internal/library/logger"
	"business/internal/library/timewrapper"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrInvalidCommand は workflow の必須入力が不足しているときに返る。
	ErrInvalidCommand = errors.New("manual mail workflow command is invalid")
	// ErrFetchConditionInvalid は fetch 条件が不正なときに返る。
	ErrFetchConditionInvalid = errors.New("manual mail workflow fetch condition is invalid")
)

// FetchCondition は workflow endpoint が受け取るメール取得条件。
type FetchCondition struct {
	LabelName string
	Since     time.Time
	Until     time.Time
}

// Normalize は文字列だけを整形し、時刻はそのまま保持する。
func (c FetchCondition) Normalize() FetchCondition {
	c.LabelName = strings.TrimSpace(c.LabelName)
	return c
}

// Validate は workflow で必要な fetch 条件の最小不変条件を検証する。
func (c FetchCondition) Validate() error {
	normalized := c.Normalize()
	if normalized.LabelName == "" {
		return fmt.Errorf("%w: label_name is required", ErrFetchConditionInvalid)
	}
	if normalized.Since.IsZero() {
		return fmt.Errorf("%w: since is required", ErrFetchConditionInvalid)
	}
	if normalized.Until.IsZero() {
		return fmt.Errorf("%w: until is required", ErrFetchConditionInvalid)
	}
	if !normalized.Since.Before(normalized.Until) {
		return fmt.Errorf("%w: since must be before until", ErrFetchConditionInvalid)
	}
	return nil
}

// Command は manual mail workflow の入力。
type Command struct {
	UserID       uint
	ConnectionID uint
	Condition    FetchCondition
}

// CreatedEmail は fetch から analysis に渡す workflow 内部 payload。
type CreatedEmail struct {
	EmailID           uint
	ExternalMessageID string
	Subject           string
	From              string
	To                []string
	ReceivedAt        time.Time
	Body              string
	BodyDigest        string
}

// FetchFailure は fetch stage から返る部分失敗。
type FetchFailure struct {
	ExternalMessageID string
	Stage             string
	Code              string
	Message           string
}

// AnalysisFailure は analysis stage から返る部分失敗。
type AnalysisFailure struct {
	EmailID           uint
	ExternalMessageID string
	Stage             string
	Code              string
	Message           string
}

// FetchResult は fetch stage の正規化済み出力。
type FetchResult struct {
	CreatedEmailIDs  []uint
	CreatedEmails    []CreatedEmail
	ExistingEmailIDs []uint
	Failures         []FetchFailure
}

// AnalyzeResult は analysis stage の正規化済み出力。
type AnalyzeResult struct {
	ParsedEmailIDs   []uint
	ParsedEmails     []ParsedEmail
	ParsedEmailCount int
	Failures         []AnalysisFailure
}

// ParsedEmail は保存済み ParsedEmail と下流で使う source email 情報を束ねた workflow 所有の型。
type ParsedEmail struct {
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	Subject           string
	From              string
	To                []string
	BodyDigest        string
	Data              commondomain.ParsedEmail
}

// ResolvedItem は vendor 解決に成功した 1 件分の結果。
type ResolvedItem struct {
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	VendorID          uint
	VendorName        string
	MatchedBy         string
	Data              commondomain.ParsedEmail
}

// VendorResolutionFailure は vendorresolution stage の部分失敗。
type VendorResolutionFailure struct {
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	Stage             string
	Code              string
	Message           string
}

// VendorResolutionResult は vendorresolution stage の正規化済み出力。
type VendorResolutionResult struct {
	ResolvedItems   []ResolvedItem
	ResolvedCount   int
	UnresolvedItems []UnresolvedItem
	UnresolvedCount int
	Failures        []VendorResolutionFailure
}

// UnresolvedItem is a vendorresolution business failure returned by the stage.
type UnresolvedItem struct {
	ParsedEmailID       uint
	EmailID             uint
	ExternalMessageID   string
	ReasonCode          string
	Message             string
	CandidateVendorName string
}

// EligibleItem は billingeligibility stage で請求成立と判定された 1 件分の結果。
type EligibleItem struct {
	ParsedEmailID      uint
	EmailID            uint
	ExternalMessageID  string
	VendorID           uint
	VendorName         string
	MatchedBy          string
	ProductNameDisplay *string
	BillingNumber      string
	InvoiceNumber      *string
	Amount             float64
	BillingDate        *time.Time
	Currency           string
	PaymentCycle       string
}

// IneligibleItem は billingeligibility stage の業務上の非成立結果。
type IneligibleItem struct {
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	VendorID          uint
	VendorName        string
	MatchedBy         string
	ReasonCode        string
	Message           string
}

// BillingEligibilityFailure は billingeligibility stage の部分失敗。
type BillingEligibilityFailure struct {
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	Code              string
	Message           string
}

// BillingEligibilityResult は billingeligibility stage の正規化済み出力。
type BillingEligibilityResult struct {
	EligibleItems   []EligibleItem
	EligibleCount   int
	IneligibleItems []IneligibleItem
	IneligibleCount int
	Failures        []BillingEligibilityFailure
}

// BillingCreatedItem is a successfully created billing result.
type BillingCreatedItem struct {
	BillingID         uint
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	VendorID          uint
	VendorName        string
	BillingNumber     string
}

// BillingDuplicateItem is a duplicate billing result mapped to an existing billing row.
type BillingDuplicateItem struct {
	ExistingBillingID uint
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	VendorID          uint
	VendorName        string
	BillingNumber     string
	ReasonCode        string
	Message           string
}

// BillingFailure is a billing stage failure for a single target.
type BillingFailure struct {
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	Stage             string
	Code              string
	Message           string
}

// BillingResult is the billing stage output.
type BillingResult struct {
	CreatedItems   []BillingCreatedItem
	CreatedCount   int
	DuplicateItems []BillingDuplicateItem
	DuplicateCount int
	Failures       []BillingFailure
}

// Result は workflow 全体の統合結果。
type Result struct {
	Fetch              FetchResult
	Analysis           AnalyzeResult
	VendorResolution   VendorResolutionResult
	BillingEligibility BillingEligibilityResult
	Billing            BillingResult
}

// FetchCommand は workflow が所有する fetch stage 入力。
type FetchCommand struct {
	UserID       uint
	ConnectionID uint
	Condition    FetchCondition
}

// AnalyzeCommand は workflow が所有する analysis stage 入力。
type AnalyzeCommand struct {
	UserID uint
	Emails []CreatedEmail
}

// VendorResolutionCommand は workflow が所有する vendorresolution stage 入力。
type VendorResolutionCommand struct {
	UserID       uint
	ParsedEmails []ParsedEmail
}

// BillingEligibilityCommand は workflow が所有する billingeligibility stage 入力。
type BillingEligibilityCommand struct {
	UserID        uint
	ResolvedItems []ResolvedItem
}

// BillingCommand is the workflow-owned billing stage input.
type BillingCommand struct {
	UserID        uint
	EligibleItems []EligibleItem
}

// FetchStage は workflow から mailfetch stage を実行する。
type FetchStage interface {
	Execute(ctx context.Context, cmd FetchCommand) (FetchResult, error)
}

// AnalyzeStage は workflow から mailanalysis stage を実行する。
type AnalyzeStage interface {
	Execute(ctx context.Context, cmd AnalyzeCommand) (AnalyzeResult, error)
}

// VendorResolutionStage は workflow から vendorresolution stage を実行する。
type VendorResolutionStage interface {
	Execute(ctx context.Context, cmd VendorResolutionCommand) (VendorResolutionResult, error)
}

// BillingEligibilityStage は workflow から billingeligibility stage を実行する。
type BillingEligibilityStage interface {
	Execute(ctx context.Context, cmd BillingEligibilityCommand) (BillingEligibilityResult, error)
}

// BillingStage is the workflow adapter for the billing stage.
type BillingStage interface {
	Execute(ctx context.Context, cmd BillingCommand) (BillingResult, error)
}

// UseCase は manual mail workflow を実行する。
type UseCase interface {
	Execute(ctx context.Context, job DispatchJob) (Result, error)
}

type useCase struct {
	fetchStage              FetchStage
	analyzeStage            AnalyzeStage
	vendorResolutionStage   VendorResolutionStage
	billingEligibilityStage BillingEligibilityStage
	billingStage            BillingStage
	repository              WorkflowStatusRepository
	clock                   timewrapper.ClockInterface
	log                     logger.Interface
}

// NewUseCase は manual mail workflow の usecase を生成する。
func NewUseCase(
	fetchStage FetchStage,
	analyzeStage AnalyzeStage,
	vendorResolutionStage VendorResolutionStage,
	billingEligibilityStage BillingEligibilityStage,
	billingStage BillingStage,
	repository WorkflowStatusRepository,
	clock timewrapper.ClockInterface,
	log logger.Interface,
) UseCase {
	if clock == nil {
		clock = timewrapper.NewClock()
	}
	if log == nil {
		log = logger.NewNop()
	}

	return &useCase{
		fetchStage:              fetchStage,
		analyzeStage:            analyzeStage,
		vendorResolutionStage:   vendorResolutionStage,
		billingEligibilityStage: billingEligibilityStage,
		billingStage:            billingStage,
		repository:              repository,
		clock:                   clock,
		log:                     log.With(logger.Component("manual_mail_workflow_usecase")),
	}
}

// Execute は fetch -> analysis -> vendorresolution -> billingeligibility -> billing の順で workflow を進める。
func (uc *useCase) Execute(ctx context.Context, job DispatchJob) (result Result, err error) {
	if ctx == nil {
		return Result{}, logger.ErrNilContext
	}
	if err := validateDispatchJob(job); err != nil {
		return Result{}, err
	}
	if err := uc.validateDependencies(); err != nil {
		return Result{}, err
	}

	reqLog := uc.log
	if withContext, err := uc.log.WithContext(ctx); err == nil {
		reqLog = withContext
	}

	currentStage := ""
	defer func() {
		if recovered := recover(); recovered != nil {
			reqLog.Error("manual_mail_workflow_panicked",
				logger.String("workflow_id", job.WorkflowID),
				logger.Recovered(recovered),
				logger.StackTrace(),
				logger.Uint("connection_id", job.ConnectionID),
			)
			err = uc.failWorkflow(ctx, job.HistoryID, currentStage, fmt.Errorf("manual mail workflow panicked: %v", recovered), reqLog)
		}
	}()

	job.Condition = job.Condition.Normalize()

	currentStage = workflowStageFetch
	if err := uc.repository.MarkRunning(ctx, job.HistoryID, currentStage); err != nil {
		return Result{}, uc.failWorkflow(ctx, job.HistoryID, currentStage, err, reqLog)
	}

	fetchResult, err := uc.fetchStage.Execute(ctx, FetchCommand{
		UserID:       job.UserID,
		ConnectionID: job.ConnectionID,
		Condition:    job.Condition,
	})
	if err != nil {
		return Result{}, uc.failWorkflow(ctx, job.HistoryID, currentStage, err, reqLog)
	}

	result = Result{Fetch: fetchResult}
	if err := uc.repository.SaveStageProgress(ctx, buildFetchStageProgress(job.HistoryID, fetchResult)); err != nil {
		return result, uc.failWorkflow(ctx, job.HistoryID, currentStage, err, reqLog)
	}
	if len(fetchResult.CreatedEmails) == 0 {
		finalStatus := workflowStatusForResult(result)
		if err := uc.repository.Complete(ctx, job.HistoryID, finalStatus, uc.clock.Now().UTC()); err != nil {
			return result, uc.failWorkflow(ctx, job.HistoryID, currentStage, err, reqLog)
		}
		reqLog.Info("manual_mail_workflow_completed",
			logger.UserID(job.UserID),
			logger.Uint("connection_id", job.ConnectionID),
			logger.String("workflow_id", job.WorkflowID),
			logger.String("status", finalStatus),
			logger.Int("created_email_count", len(fetchResult.CreatedEmailIDs)),
			logger.Int("parsed_email_count", 0),
			logger.Int("resolved_vendor_count", 0),
			logger.Int("unresolved_vendor_count", 0),
			logger.Int("eligible_billing_count", 0),
			logger.Int("ineligible_billing_count", 0),
			logger.Int("created_billing_count", 0),
			logger.Int("duplicate_billing_count", 0),
			logger.Int("fetch_business_failure_count", 0),
			logger.Int("fetch_technical_failure_count", len(fetchResult.Failures)),
			logger.Int("analysis_business_failure_count", 0),
			logger.Int("analysis_technical_failure_count", 0),
			logger.Int("vendor_resolution_business_failure_count", 0),
			logger.Int("vendor_resolution_technical_failure_count", 0),
			logger.Int("billing_eligibility_business_failure_count", 0),
			logger.Int("billing_eligibility_technical_failure_count", 0),
			logger.Int("billing_business_failure_count", 0),
			logger.Int("billing_technical_failure_count", 0),
		)
		return result, nil
	}

	emails := make([]CreatedEmail, len(fetchResult.CreatedEmails))
	copy(emails, fetchResult.CreatedEmails)

	currentStage = workflowStageAnalysis
	if err := uc.repository.MarkRunning(ctx, job.HistoryID, currentStage); err != nil {
		return result, uc.failWorkflow(ctx, job.HistoryID, currentStage, err, reqLog)
	}

	analysisResult, err := uc.analyzeStage.Execute(ctx, AnalyzeCommand{
		UserID: job.UserID,
		Emails: emails,
	})
	if err != nil {
		return result, uc.failWorkflow(ctx, job.HistoryID, currentStage, err, reqLog)
	}

	result.Analysis = analysisResult
	if err := uc.repository.SaveStageProgress(ctx, buildAnalysisStageProgress(job.HistoryID, analysisResult)); err != nil {
		return result, uc.failWorkflow(ctx, job.HistoryID, currentStage, err, reqLog)
	}
	if len(analysisResult.ParsedEmails) > 0 {
		currentStage = workflowStageVendorResolution
		if err := uc.repository.MarkRunning(ctx, job.HistoryID, currentStage); err != nil {
			return result, uc.failWorkflow(ctx, job.HistoryID, currentStage, err, reqLog)
		}
		vendorResolutionResult, err := uc.vendorResolutionStage.Execute(ctx, VendorResolutionCommand{
			UserID:       job.UserID,
			ParsedEmails: append([]ParsedEmail(nil), analysisResult.ParsedEmails...),
		})
		if err != nil {
			return result, uc.failWorkflow(ctx, job.HistoryID, currentStage, err, reqLog)
		}
		result.VendorResolution = vendorResolutionResult
		if err := uc.repository.SaveStageProgress(ctx, buildVendorResolutionStageProgress(job.HistoryID, analysisResult.ParsedEmails, vendorResolutionResult)); err != nil {
			return result, uc.failWorkflow(ctx, job.HistoryID, currentStage, err, reqLog)
		}
		if len(vendorResolutionResult.ResolvedItems) > 0 {
			currentStage = workflowStageBillingEligibility
			if err := uc.repository.MarkRunning(ctx, job.HistoryID, currentStage); err != nil {
				return result, uc.failWorkflow(ctx, job.HistoryID, currentStage, err, reqLog)
			}
			billingEligibilityResult, err := uc.billingEligibilityStage.Execute(ctx, BillingEligibilityCommand{
				UserID:        job.UserID,
				ResolvedItems: append([]ResolvedItem(nil), vendorResolutionResult.ResolvedItems...),
			})
			if err != nil {
				return result, uc.failWorkflow(ctx, job.HistoryID, currentStage, err, reqLog)
			}
			result.BillingEligibility = billingEligibilityResult
			if err := uc.repository.SaveStageProgress(ctx, buildBillingEligibilityStageProgress(job.HistoryID, billingEligibilityResult)); err != nil {
				return result, uc.failWorkflow(ctx, job.HistoryID, currentStage, err, reqLog)
			}
			if len(billingEligibilityResult.EligibleItems) > 0 {
				currentStage = workflowStageBilling
				if err := uc.repository.MarkRunning(ctx, job.HistoryID, currentStage); err != nil {
					return result, uc.failWorkflow(ctx, job.HistoryID, currentStage, err, reqLog)
				}
				billingResult, err := uc.billingStage.Execute(ctx, BillingCommand{
					UserID:        job.UserID,
					EligibleItems: append([]EligibleItem(nil), billingEligibilityResult.EligibleItems...),
				})
				if err != nil {
					return result, uc.failWorkflow(ctx, job.HistoryID, currentStage, err, reqLog)
				}
				result.Billing = billingResult
				if err := uc.repository.SaveStageProgress(ctx, buildBillingStageProgress(job.HistoryID, billingResult)); err != nil {
					return result, uc.failWorkflow(ctx, job.HistoryID, currentStage, err, reqLog)
				}
			}
		}
	}

	finalStatus := workflowStatusForResult(result)
	if err := uc.repository.Complete(ctx, job.HistoryID, finalStatus, uc.clock.Now().UTC()); err != nil {
		return result, uc.failWorkflow(ctx, job.HistoryID, currentStage, err, reqLog)
	}

	reqLog.Info("manual_mail_workflow_completed",
		logger.UserID(job.UserID),
		logger.Uint("connection_id", job.ConnectionID),
		logger.String("workflow_id", job.WorkflowID),
		logger.String("status", finalStatus),
		logger.Int("created_email_count", len(fetchResult.CreatedEmailIDs)),
		logger.Int("parsed_email_count", len(analysisResult.ParsedEmailIDs)),
		logger.Int("resolved_vendor_count", result.VendorResolution.ResolvedCount),
		logger.Int("unresolved_vendor_count", result.VendorResolution.UnresolvedCount),
		logger.Int("eligible_billing_count", result.BillingEligibility.EligibleCount),
		logger.Int("ineligible_billing_count", result.BillingEligibility.IneligibleCount),
		logger.Int("created_billing_count", result.Billing.CreatedCount),
		logger.Int("duplicate_billing_count", result.Billing.DuplicateCount),
		logger.Int("fetch_business_failure_count", 0),
		logger.Int("fetch_technical_failure_count", len(fetchResult.Failures)),
		logger.Int("analysis_business_failure_count", 0),
		logger.Int("analysis_technical_failure_count", len(analysisResult.Failures)),
		logger.Int("vendor_resolution_business_failure_count", result.VendorResolution.UnresolvedCount),
		logger.Int("vendor_resolution_technical_failure_count", len(result.VendorResolution.Failures)),
		logger.Int("billing_eligibility_business_failure_count", result.BillingEligibility.IneligibleCount),
		logger.Int("billing_eligibility_technical_failure_count", len(result.BillingEligibility.Failures)),
		logger.Int("billing_business_failure_count", result.Billing.DuplicateCount),
		logger.Int("billing_technical_failure_count", len(result.Billing.Failures)),
	)

	return result, nil
}

func (uc *useCase) validateDependencies() error {
	if uc.fetchStage == nil {
		return errors.New("fetch_stage is not configured")
	}
	if uc.analyzeStage == nil {
		return errors.New("analyze_stage is not configured")
	}
	if uc.vendorResolutionStage == nil {
		return errors.New("vendor_resolution_stage is not configured")
	}
	if uc.billingEligibilityStage == nil {
		return errors.New("billing_eligibility_stage is not configured")
	}
	if uc.billingStage == nil {
		return errors.New("billing_stage is not configured")
	}
	if uc.repository == nil {
		return errors.New("workflow_status_repository is not configured")
	}
	return nil
}

func (uc *useCase) failWorkflow(
	ctx context.Context,
	historyID uint64,
	currentStage string,
	runErr error,
	reqLog logger.Interface,
) error {
	finishedAt := uc.clock.Now().UTC()
	if err := uc.repository.Fail(ctx, historyID, currentStage, finishedAt, localizedWorkflowErrorMessage(currentStage, runErr)); err != nil {
		reqLog.Error("manual_mail_workflow_fail_persist_failed",
			logger.String("current_stage", currentStage),
			logger.Uint("history_id", uint(historyID)),
			logger.Err(err),
		)
	}
	return runErr
}

func validateDispatchJob(job DispatchJob) error {
	if job.HistoryID == 0 {
		return fmt.Errorf("%w: history_id is required", ErrInvalidCommand)
	}
	if strings.TrimSpace(job.WorkflowID) == "" {
		return fmt.Errorf("%w: workflow_id is required", ErrInvalidCommand)
	}
	return validateCommand(Command{
		UserID:       job.UserID,
		ConnectionID: job.ConnectionID,
		Condition:    job.Condition,
	})
}

func validateCommand(cmd Command) error {
	if cmd.UserID == 0 {
		return fmt.Errorf("%w: user_id is required", ErrInvalidCommand)
	}
	if cmd.ConnectionID == 0 {
		return fmt.Errorf("%w: connection_id is required", ErrInvalidCommand)
	}
	if err := cmd.Condition.Validate(); err != nil {
		return err
	}
	return nil
}
