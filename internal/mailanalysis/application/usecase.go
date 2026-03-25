package application

import (
	commondomain "business/internal/common/domain"
	"business/internal/library/logger"
	"business/internal/library/timewrapper"
	"business/internal/mailanalysis/domain"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// AnalyzerSpec は analyzer 実装を選ぶための実行コンテキスト。
type AnalyzerSpec struct {
	UserID uint
}

// AnalyzerFactory は現在の実行に使う analyzer を生成する。
type AnalyzerFactory interface {
	Create(ctx context.Context, spec AnalyzerSpec) (Analyzer, error)
}

// Analyzer は AI 解析を実行し、正規化済みの draft を返す。
type Analyzer interface {
	Analyze(ctx context.Context, email EmailForAnalysisTarget) (domain.AnalysisOutput, error)
}

// ParsedEmailRepository は ParsedEmail の履歴を永続化する。
type ParsedEmailRepository interface {
	SaveAll(ctx context.Context, input domain.SaveInput) ([]domain.ParsedEmailRecord, error)
}

// EmailForAnalysisTarget は mailanalysis が受け取る workflow 境界 DTO。
type EmailForAnalysisTarget struct {
	EmailID           uint
	ExternalMessageID string
	Subject           string
	From              string
	To                []string
	ReceivedAt        time.Time
	Body              string
	BodyDigest        string
}

// Normalize は文字列項目と宛先一覧を整形する。
func (e EmailForAnalysisTarget) Normalize() EmailForAnalysisTarget {
	e.ExternalMessageID = strings.TrimSpace(e.ExternalMessageID)
	e.Subject = strings.TrimSpace(e.Subject)
	e.From = strings.TrimSpace(e.From)
	e.Body = strings.TrimSpace(e.Body)
	e.BodyDigest = strings.TrimSpace(e.BodyDigest)

	recipients := make([]string, 0, len(e.To))
	for _, recipient := range e.To {
		trimmed := strings.TrimSpace(recipient)
		if trimmed == "" {
			continue
		}
		recipients = append(recipients, trimmed)
	}
	e.To = recipients

	return e
}

// Validate は analyzer 入力に必要な最小不変条件を検証する。
func (e EmailForAnalysisTarget) Validate() error {
	if e.EmailID == 0 {
		return fmt.Errorf("%w: email_id is required", domain.ErrEmailForAnalysisInvalid)
	}
	if strings.TrimSpace(e.ExternalMessageID) == "" {
		return fmt.Errorf("%w: external_message_id is required", domain.ErrEmailForAnalysisInvalid)
	}
	if strings.TrimSpace(e.Body) == "" {
		return fmt.Errorf("%w: body is required", domain.ErrEmailForAnalysisInvalid)
	}
	return nil
}

// Command は mailanalysis stage の入力。
type Command struct {
	UserID uint
	Emails []EmailForAnalysisTarget
}

// ParsedEmailResultItem は保存済み ParsedEmail と source email の必要情報をまとめたもの。
type ParsedEmailResultItem struct {
	ParsedEmailID     uint
	EmailID           uint
	ExternalMessageID string
	Subject           string
	From              string
	To                []string
	BodyDigest        string
	ParsedEmail       commondomain.ParsedEmail
}

// Result は mailanalysis stage の出力。
type Result struct {
	ParsedEmailIDs   []uint
	ParsedEmails     []ParsedEmailResultItem
	ParsedEmailCount int
	Failures         []domain.MessageFailure
}

// UseCase は mailanalysis stage を実行する。
type UseCase interface {
	Execute(ctx context.Context, cmd Command) (Result, error)
}

type useCase struct {
	clock           timewrapper.ClockInterface
	analyzerFactory AnalyzerFactory
	repository      ParsedEmailRepository
	log             logger.Interface
}

// NewUseCase は mailanalysis の usecase を生成する。
func NewUseCase(
	clock timewrapper.ClockInterface,
	analyzerFactory AnalyzerFactory,
	repository ParsedEmailRepository,
	log logger.Interface,
) UseCase {
	if clock == nil {
		clock = timewrapper.NewClock()
	}
	if log == nil {
		log = logger.NewNop()
	}

	return &useCase{
		clock:           clock,
		analyzerFactory: analyzerFactory,
		repository:      repository,
		log:             log.With(logger.Component("email_analysis_usecase")),
	}
}

// Execute は入力検証、AI 解析、ParsedEmail 履歴保存を順に実行する。
func (uc *useCase) Execute(ctx context.Context, cmd Command) (Result, error) {
	if ctx == nil {
		return Result{}, logger.ErrNilContext
	}
	if err := validateCommand(cmd); err != nil {
		return Result{}, err
	}
	if len(cmd.Emails) == 0 {
		return Result{}, nil
	}
	if err := uc.validateDependencies(); err != nil {
		return Result{}, err
	}

	reqLog := uc.log
	if withContext, err := uc.log.WithContext(ctx); err == nil {
		reqLog = withContext
	}

	analyzer, err := uc.analyzerFactory.Create(ctx, AnalyzerSpec{UserID: cmd.UserID})
	if err != nil {
		return Result{}, fmt.Errorf("failed to create analyzer: %w", err)
	}

	result := Result{}
	for _, email := range cmd.Emails {
		email = email.Normalize()
		if err := email.Validate(); err != nil {
			result.Failures = append(result.Failures, domain.MessageFailure{
				EmailID:           email.EmailID,
				ExternalMessageID: email.ExternalMessageID,
				Stage:             domain.FailureStageNormalizeInput,
				Code:              domain.FailureCodeInvalidEmailInput,
				Message:           messageForInvalidEmailInput(email),
			})
			continue
		}

		output, err := analyzer.Analyze(ctx, email)
		if err != nil {
			reqLog.Error("email_analysis_failed",
				logger.UserID(cmd.UserID),
				logger.Uint("email_id", email.EmailID),
				logger.String("external_message_id", email.ExternalMessageID),
				logger.Err(err),
			)
			result.Failures = append(result.Failures, failureForAnalyzeError(email, err))
			continue
		}

		output = output.Normalize()
		output.ParsedEmails = applyFallbackBillingNumbers(output.ParsedEmails, email.BodyDigest)

		if len(output.ParsedEmails) == 0 {
			result.Failures = append(result.Failures, domain.MessageFailure{
				EmailID:           email.EmailID,
				ExternalMessageID: email.ExternalMessageID,
				Stage:             domain.FailureStageResponseParse,
				Code:              domain.FailureCodeAnalysisResponseEmpty,
				Message:           messageForAnalysisResponseEmpty(email),
			})
			continue
		}

		records, err := uc.repository.SaveAll(ctx, domain.SaveInput{
			UserID:        cmd.UserID,
			EmailID:       email.EmailID,
			AnalysisRunID: uuid.NewString(),
			PositionBase:  0,
			ExtractedAt:   uc.clock.Now().UTC(),
			PromptVersion: output.PromptVersion,
			ParsedEmails:  output.ParsedEmails,
		})
		if err != nil {
			reqLog.Error("parsed_email_save_failed",
				logger.UserID(cmd.UserID),
				logger.Uint("email_id", email.EmailID),
				logger.String("external_message_id", email.ExternalMessageID),
				logger.Err(err),
			)
			result.Failures = append(result.Failures, domain.MessageFailure{
				EmailID:           email.EmailID,
				ExternalMessageID: email.ExternalMessageID,
				Stage:             domain.FailureStageSave,
				Code:              domain.FailureCodeParsedEmailSaveFailed,
				Message:           messageForParsedEmailSaveFailed(email),
			})
			continue
		}

		for _, record := range records {
			result.ParsedEmailIDs = append(result.ParsedEmailIDs, record.ID)
		}
		// SaveAll の返却順は analyzer の出力順と同じ前提で、保存済み ID を ParsedEmail にひも付ける。
		for idx, record := range records {
			if idx >= len(output.ParsedEmails) {
				reqLog.Error("parsed_email_record_count_mismatch",
					logger.UserID(cmd.UserID),
					logger.Uint("email_id", email.EmailID),
					logger.Int("saved_record_count", len(records)),
					logger.Int("analyzed_parsed_email_count", len(output.ParsedEmails)),
				)
				break
			}
			result.ParsedEmails = append(result.ParsedEmails, ParsedEmailResultItem{
				ParsedEmailID:     record.ID,
				EmailID:           record.EmailID,
				ExternalMessageID: email.ExternalMessageID,
				Subject:           email.Subject,
				From:              email.From,
				To:                append([]string(nil), email.To...),
				BodyDigest:        email.BodyDigest,
				ParsedEmail:       output.ParsedEmails[idx],
			})
		}
		result.ParsedEmailCount += len(records)
	}

	reqLog.Info("email_analysis_succeeded",
		logger.UserID(cmd.UserID),
		logger.Int("input_email_count", len(cmd.Emails)),
		logger.Int("parsed_email_count", result.ParsedEmailCount),
		logger.Int("failure_count", len(result.Failures)),
	)

	return result, nil
}

func (uc *useCase) validateDependencies() error {
	if uc.analyzerFactory == nil {
		return errors.New("analyzer_factory is not configured")
	}
	if uc.repository == nil {
		return errors.New("parsed_email_repository is not configured")
	}
	return nil
}

func validateCommand(cmd Command) error {
	if cmd.UserID == 0 {
		return fmt.Errorf("%w: user_id is required", domain.ErrInvalidCommand)
	}
	return nil
}

const fallbackBillingNumberPrefix = "digest_"

func applyFallbackBillingNumbers(parsedEmails []commondomain.ParsedEmail, bodyDigest string) []commondomain.ParsedEmail {
	bodyDigest = strings.TrimSpace(bodyDigest)
	if bodyDigest == "" {
		return parsedEmails
	}

	fallback := fallbackBillingNumberPrefix + bodyDigest
	for idx := range parsedEmails {
		if parsedEmails[idx].BillingNumber != nil && strings.TrimSpace(*parsedEmails[idx].BillingNumber) != "" {
			continue
		}
		parsedEmails[idx].BillingNumber = fallbackStringPtr(fallback)
	}

	return parsedEmails
}

func fallbackStringPtr(value string) *string {
	return &value
}

func failureForAnalyzeError(email EmailForAnalysisTarget, err error) domain.MessageFailure {
	if errors.Is(err, domain.ErrAnalysisResponseInvalid) {
		return domain.MessageFailure{
			EmailID:           email.EmailID,
			ExternalMessageID: email.ExternalMessageID,
			Stage:             domain.FailureStageResponseParse,
			Code:              domain.FailureCodeAnalysisResponseInvalid,
			Message:           messageForAnalysisResponseInvalid(email),
		}
	}

	return domain.MessageFailure{
		EmailID:           email.EmailID,
		ExternalMessageID: email.ExternalMessageID,
		Stage:             domain.FailureStageAnalyze,
		Code:              domain.FailureCodeAnalysisFailed,
		Message:           messageForAnalysisFailed(email),
	}
}

func messageForInvalidEmailInput(email EmailForAnalysisTarget) string {
	return describeEmailReference(email) + " の入力が不正です。件名、本文、外部メッセージIDを確認してください。"
}

func messageForAnalysisFailed(email EmailForAnalysisTarget) string {
	return describeEmailReference(email) + " の解析に失敗しました。しばらく時間をおいて再実行してください。"
}

func messageForAnalysisResponseInvalid(email EmailForAnalysisTarget) string {
	return describeEmailReference(email) + " の解析結果の形式が不正でした。"
}

func messageForAnalysisResponseEmpty(email EmailForAnalysisTarget) string {
	return describeEmailReference(email) + " の解析結果を取得できませんでした。"
}

func messageForParsedEmailSaveFailed(email EmailForAnalysisTarget) string {
	return describeEmailReference(email) + " の解析結果の保存に失敗しました。"
}

func describeEmailReference(email EmailForAnalysisTarget) string {
	switch {
	case email.ExternalMessageID != "" && email.EmailID != 0:
		return "メールID " + fmt.Sprint(email.EmailID) + " (" + email.ExternalMessageID + ")"
	case email.ExternalMessageID != "":
		return "メール " + email.ExternalMessageID
	case email.EmailID != 0:
		return "メールID " + fmt.Sprint(email.EmailID)
	default:
		return "メール"
	}
}
