package application

import (
	commondomain "business/internal/common/domain"
	"business/internal/library/logger"
	"business/internal/vendorresolution/domain"
	"context"
	"errors"
	"fmt"
	"strings"
)

// VendorResolutionRepository は判定に必要な材料を DB から収集する。
type VendorResolutionRepository interface {
	FetchFacts(ctx context.Context, plan domain.VendorResolutionFetchPlan) (domain.VendorResolutionFacts, error)
}

// VendorRegistrationRepository は未解決の候補 vendor 名を canonical Vendor として補完する。
type VendorRegistrationRepository interface {
	EnsureByPlan(ctx context.Context, plan domain.VendorRegistrationPlan) (*commondomain.Vendor, error)
}

// ResolutionTarget は workflow から渡される 1 件分の vendor 解決入力。
type ResolutionTarget struct {
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	Subject           string
	From              string
	To                []string
	BodyDigest        string
	ParsedEmail       commondomain.ParsedEmail
}

// Normalize は自由入力文字列を整形し、ParsedEmail も正規化する。
func (t ResolutionTarget) Normalize() ResolutionTarget {
	t.ExternalMessageID = strings.TrimSpace(t.ExternalMessageID)
	t.Subject = strings.TrimSpace(t.Subject)
	t.From = strings.TrimSpace(t.From)
	t.To = normalizeStrings(t.To)
	t.BodyDigest = strings.TrimSpace(t.BodyDigest)
	t.ParsedEmail = t.ParsedEmail.Normalize()
	return t
}

// Validate は vendor 解決に必要な最低限の入力を検証する。
func (t ResolutionTarget) Validate() error {
	if t.ParsedEmailID == 0 {
		return errors.New("parsed_email_id is required")
	}
	if t.EmailID == 0 {
		return errors.New("email_id is required")
	}
	return nil
}

// Command は vendorresolution stage の入力。
type Command struct {
	UserID       uint
	ParsedEmails []ResolutionTarget
}

// Result は vendorresolution stage の出力。
type Result struct {
	ResolvedItems   []domain.ResolvedItem
	ResolvedCount   int
	UnresolvedItems []domain.UnresolvedItem
	UnresolvedCount int
	Failures        []domain.Failure
}

// UseCase は workflow から渡されたデータで vendor 正規化を実行する。
type UseCase interface {
	Execute(ctx context.Context, cmd Command) (Result, error)
}

type useCase struct {
	resolutionRepository   VendorResolutionRepository
	registrationRepository VendorRegistrationRepository
	policy                 commondomain.VendorResolutionPolicy
	log                    logger.Interface
}

// NewUseCase は vendorresolution の usecase を生成する。
func NewUseCase(resolutionRepository VendorResolutionRepository, registrationRepository VendorRegistrationRepository, log logger.Interface) UseCase {
	if log == nil {
		log = logger.NewNop()
	}

	return &useCase{
		resolutionRepository:   resolutionRepository,
		registrationRepository: registrationRepository,
		policy:                 commondomain.VendorResolutionPolicy{},
		log:                    log.With(logger.Component("vendor_resolution_usecase")),
	}
}

// Execute は渡された ParsedEmail 群から canonical Vendor を解決する。
func (uc *useCase) Execute(ctx context.Context, cmd Command) (Result, error) {
	if ctx == nil {
		return Result{}, logger.ErrNilContext
	}
	if err := validateCommand(cmd); err != nil {
		return Result{}, err
	}
	if len(cmd.ParsedEmails) == 0 {
		return Result{}, nil
	}
	if err := uc.validateDependencies(); err != nil {
		return Result{}, err
	}

	reqLog := uc.log
	if withContext, err := uc.log.WithContext(ctx); err == nil {
		reqLog = withContext
	}

	result := Result{}
	for _, target := range cmd.ParsedEmails {
		// workflow から渡された値をここで最終整形し、policy と repository に揺れの少ない入力だけを渡す。
		target = target.Normalize()
		if err := target.Validate(); err != nil {
			result.Failures = append(result.Failures, domain.Failure{
				ParsedEmailID:     target.ParsedEmailID,
				EmailID:           target.EmailID,
				ExternalMessageID: target.ExternalMessageID,
				Stage:             domain.FailureStageNormalizeInput,
				Code:              domain.FailureCodeInvalidResolutionTarget,
				Message:           messageForResolutionFailure(target, domain.FailureCodeInvalidResolutionTarget),
			})
			continue
		}

		decision, failure, err := uc.resolveTarget(ctx, cmd.UserID, target, reqLog)
		if err != nil {
			// read/write repository の内部エラーは全体失敗にせず、対象メール単位の failure として集約する。
			result.Failures = append(result.Failures, *failure)
			continue
		}

		if !decision.Resolution.IsResolved() {
			result.UnresolvedItems = append(result.UnresolvedItems, domain.UnresolvedItem{
				ParsedEmailID:       target.ParsedEmailID,
				EmailID:             target.EmailID,
				ExternalMessageID:   target.ExternalMessageID,
				ReasonCode:          domain.ReasonCodeVendorUnresolved,
				Message:             messageForUnresolvedItem(target),
				CandidateVendorName: stringValue(target.ParsedEmail.VendorName),
			})
			result.UnresolvedCount = len(result.UnresolvedItems)
			reqLog.Warn("vendor_resolution_unresolved",
				logger.UserID(cmd.UserID),
				logger.Uint("parsed_email_id", target.ParsedEmailID),
				logger.Uint("email_id", target.EmailID),
				logger.String("external_message_id", target.ExternalMessageID),
				logger.String("candidate_vendor_name", stringValue(target.ParsedEmail.VendorName)),
				logger.String("sender_domain", uc.policy.BuildFetchPlan(buildVendorResolutionInput(target)).SenderDomainValue),
			)
			continue
		}

		if err := decision.Resolution.Validate(); err != nil {
			result.Failures = append(result.Failures, domain.Failure{
				ParsedEmailID:     target.ParsedEmailID,
				EmailID:           target.EmailID,
				ExternalMessageID: target.ExternalMessageID,
				Stage:             domain.FailureStageResolveVendor,
				Code:              domain.FailureCodeVendorResolveFail,
				Message:           messageForResolutionFailure(target, domain.FailureCodeVendorResolveFail),
			})
			continue
		}

		result.ResolvedItems = append(result.ResolvedItems, domain.ResolvedItem{
			ParsedEmailID:     target.ParsedEmailID,
			EmailID:           target.EmailID,
			ExternalMessageID: target.ExternalMessageID,
			VendorID:          decision.Resolution.ResolvedVendor.ID,
			VendorName:        decision.Resolution.ResolvedVendor.Name,
			MatchedBy:         decision.MatchedBy,
		})
		result.ResolvedCount++
	}

	reqLog.Info("vendor_resolution_succeeded",
		logger.UserID(cmd.UserID),
		logger.Int("input_parsed_email_count", len(cmd.ParsedEmails)),
		logger.Int("resolved_count", result.ResolvedCount),
		logger.Int("unresolved_count", result.UnresolvedCount),
		logger.Int("failure_count", len(result.Failures)),
	)

	return result, nil
}

func (uc *useCase) validateDependencies() error {
	if uc.resolutionRepository == nil {
		return errors.New("vendor_resolution_repository is not configured")
	}
	if uc.registrationRepository == nil {
		return errors.New("vendor_registration_repository is not configured")
	}
	return nil
}

// validateCommand は stage 全体として成立する最低条件だけを検証する。
func validateCommand(cmd Command) error {
	if cmd.UserID == 0 {
		return fmt.Errorf("%w: user_id is required", domain.ErrInvalidCommand)
	}
	return nil
}

// stringValue は nil な候補 vendor 名をログ出力用の空文字に寄せる。
func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// normalizeStrings は空要素を除去しつつ文字列スライスを trim する。
func normalizeStrings(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	return normalized
}
