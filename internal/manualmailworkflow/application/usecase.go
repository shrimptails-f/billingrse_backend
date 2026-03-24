package application

import (
	commondomain "business/internal/common/domain"
	"business/internal/library/logger"
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
}

// AnalysisFailure は analysis stage から返る部分失敗。
type AnalysisFailure struct {
	EmailID           uint
	ExternalMessageID string
	Stage             string
	Code              string
}

// FetchResult は fetch stage の正規化済み出力。
type FetchResult struct {
	Provider            string
	AccountIdentifier   string
	MatchedMessageCount int
	CreatedEmailIDs     []uint
	CreatedEmails       []CreatedEmail
	ExistingEmailIDs    []uint
	Failures            []FetchFailure
}

// AnalyzeResult は analysis stage の正規化済み出力。
type AnalyzeResult struct {
	ParsedEmailIDs     []uint
	ParsedEmails       []ParsedEmail
	AnalyzedEmailCount int
	ParsedEmailCount   int
	Failures           []AnalysisFailure
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
	BodyDigest        string
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
}

// VendorResolutionResult は vendorresolution stage の正規化済み出力。
type VendorResolutionResult struct {
	ResolvedItems                []ResolvedItem
	ResolvedCount                int
	UnresolvedCount              int
	UnresolvedExternalMessageIDs []string
	Failures                     []VendorResolutionFailure
}

// EligibleItem は billingeligibility stage で請求成立と判定された 1 件分の結果。
type EligibleItem struct {
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	VendorID          uint
	VendorName        string
	MatchedBy         string
	BillingNumber     string
	InvoiceNumber     *string
	Amount            float64
	BillingDate       *time.Time
	Currency          string
	PaymentCycle      string
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
}

// BillingEligibilityFailure は billingeligibility stage の部分失敗。
type BillingEligibilityFailure struct {
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	Stage             string
	Code              string
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
}

// BillingFailure is a billing stage failure for a single target.
type BillingFailure struct {
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	Stage             string
	Code              string
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
	Execute(ctx context.Context, cmd Command) (Result, error)
}

type useCase struct {
	fetchStage              FetchStage
	analyzeStage            AnalyzeStage
	vendorResolutionStage   VendorResolutionStage
	billingEligibilityStage BillingEligibilityStage
	billingStage            BillingStage
	log                     logger.Interface
}

// NewUseCase は manual mail workflow の usecase を生成する。
func NewUseCase(
	fetchStage FetchStage,
	analyzeStage AnalyzeStage,
	vendorResolutionStage VendorResolutionStage,
	billingEligibilityStage BillingEligibilityStage,
	billingStage BillingStage,
	log logger.Interface,
) UseCase {
	if log == nil {
		log = logger.NewNop()
	}

	return &useCase{
		fetchStage:              fetchStage,
		analyzeStage:            analyzeStage,
		vendorResolutionStage:   vendorResolutionStage,
		billingEligibilityStage: billingEligibilityStage,
		billingStage:            billingStage,
		log:                     log.With(logger.Component("manual_mail_workflow_usecase")),
	}
}

// Execute は fetch -> analysis -> vendorresolution -> billingeligibility -> billing の順で workflow を進める。
func (uc *useCase) Execute(ctx context.Context, cmd Command) (Result, error) {
	if ctx == nil {
		return Result{}, logger.ErrNilContext
	}
	if err := validateCommand(cmd); err != nil {
		return Result{}, err
	}
	if err := uc.validateDependencies(); err != nil {
		return Result{}, err
	}

	reqLog := uc.log
	if withContext, err := uc.log.WithContext(ctx); err == nil {
		reqLog = withContext
	}

	cmd.Condition = cmd.Condition.Normalize()

	fetchResult, err := uc.fetchStage.Execute(ctx, FetchCommand{
		UserID:       cmd.UserID,
		ConnectionID: cmd.ConnectionID,
		Condition:    cmd.Condition,
	})
	if err != nil {
		return Result{}, err
	}

	result := Result{Fetch: fetchResult}
	if len(fetchResult.CreatedEmails) == 0 {
		reqLog.Info("manual_mail_workflow_succeeded",
			logger.UserID(cmd.UserID),
			logger.Uint("connection_id", cmd.ConnectionID),
			logger.Int("created_email_count", len(fetchResult.CreatedEmailIDs)),
			logger.Int("parsed_email_count", 0),
			logger.Int("resolved_vendor_count", 0),
			logger.Int("unresolved_vendor_count", 0),
			logger.Int("eligible_billing_count", 0),
			logger.Int("ineligible_billing_count", 0),
			logger.Int("created_billing_count", 0),
			logger.Int("duplicate_billing_count", 0),
			logger.Int("fetch_failure_count", len(fetchResult.Failures)),
			logger.Int("analysis_failure_count", 0),
			logger.Int("vendor_resolution_failure_count", 0),
			logger.Int("billing_eligibility_failure_count", 0),
			logger.Int("billing_failure_count", 0),
		)
		return result, nil
	}

	emails := make([]CreatedEmail, len(fetchResult.CreatedEmails))
	copy(emails, fetchResult.CreatedEmails)

	analysisResult, err := uc.analyzeStage.Execute(ctx, AnalyzeCommand{
		UserID: cmd.UserID,
		Emails: emails,
	})
	if err != nil {
		return Result{}, err
	}

	result.Analysis = analysisResult
	if len(analysisResult.ParsedEmails) > 0 {
		vendorResolutionResult, err := uc.vendorResolutionStage.Execute(ctx, VendorResolutionCommand{
			UserID:       cmd.UserID,
			ParsedEmails: append([]ParsedEmail(nil), analysisResult.ParsedEmails...),
		})
		if err != nil {
			return Result{}, err
		}
		result.VendorResolution = vendorResolutionResult
		if len(vendorResolutionResult.ResolvedItems) > 0 {
			billingEligibilityResult, err := uc.billingEligibilityStage.Execute(ctx, BillingEligibilityCommand{
				UserID:        cmd.UserID,
				ResolvedItems: append([]ResolvedItem(nil), vendorResolutionResult.ResolvedItems...),
			})
			if err != nil {
				return Result{}, err
			}
			result.BillingEligibility = billingEligibilityResult
			if len(billingEligibilityResult.EligibleItems) > 0 {
				billingResult, err := uc.billingStage.Execute(ctx, BillingCommand{
					UserID:        cmd.UserID,
					EligibleItems: append([]EligibleItem(nil), billingEligibilityResult.EligibleItems...),
				})
				if err != nil {
					return Result{}, err
				}
				result.Billing = billingResult
			}
		}
	}

	reqLog.Info("manual_mail_workflow_succeeded",
		logger.UserID(cmd.UserID),
		logger.Uint("connection_id", cmd.ConnectionID),
		logger.Int("created_email_count", len(fetchResult.CreatedEmailIDs)),
		logger.Int("parsed_email_count", len(analysisResult.ParsedEmailIDs)),
		logger.Int("resolved_vendor_count", result.VendorResolution.ResolvedCount),
		logger.Int("unresolved_vendor_count", result.VendorResolution.UnresolvedCount),
		logger.Int("eligible_billing_count", result.BillingEligibility.EligibleCount),
		logger.Int("ineligible_billing_count", result.BillingEligibility.IneligibleCount),
		logger.Int("created_billing_count", result.Billing.CreatedCount),
		logger.Int("duplicate_billing_count", result.Billing.DuplicateCount),
		logger.Int("fetch_failure_count", len(fetchResult.Failures)),
		logger.Int("analysis_failure_count", len(analysisResult.Failures)),
		logger.Int("vendor_resolution_failure_count", len(result.VendorResolution.Failures)),
		logger.Int("billing_eligibility_failure_count", len(result.BillingEligibility.Failures)),
		logger.Int("billing_failure_count", len(result.Billing.Failures)),
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
	return nil
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
