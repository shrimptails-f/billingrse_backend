package application

import (
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

// AnalyzerSpec describes the context used to resolve an analyzer implementation.
type AnalyzerSpec struct {
	UserID uint
}

// AnalyzerFactory creates analyzers for the current execution.
type AnalyzerFactory interface {
	Create(ctx context.Context, spec AnalyzerSpec) (Analyzer, error)
}

// Analyzer runs the AI analysis and returns normalized drafts.
type Analyzer interface {
	Analyze(ctx context.Context, email EmailForAnalysisTarget) (domain.AnalysisOutput, error)
}

// ParsedEmailRepository persists ParsedEmail history records.
type ParsedEmailRepository interface {
	SaveAll(ctx context.Context, input domain.SaveInput) ([]domain.ParsedEmailRecord, error)
}

// EmailForAnalysisTarget is the workflow boundary DTO consumed by mailanalysis.
type EmailForAnalysisTarget struct {
	EmailID           uint
	ExternalMessageID string
	Subject           string
	From              string
	To                []string
	ReceivedAt        time.Time
	Body              string
}

// Normalize trims string fields and recipient values.
func (e EmailForAnalysisTarget) Normalize() EmailForAnalysisTarget {
	e.ExternalMessageID = strings.TrimSpace(e.ExternalMessageID)
	e.Subject = strings.TrimSpace(e.Subject)
	e.From = strings.TrimSpace(e.From)
	e.Body = strings.TrimSpace(e.Body)

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

// Validate enforces the minimum invariants for analyzer input.
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

// Command is the input contract for the mailanalysis stage.
type Command struct {
	UserID uint
	Emails []EmailForAnalysisTarget
}

// Result is the output contract for the mailanalysis stage.
type Result struct {
	ParsedEmailIDs     []uint
	AnalyzedEmailCount int
	ParsedEmailCount   int
	Failures           []domain.MessageFailure
}

// UseCase executes the mailanalysis stage.
type UseCase interface {
	Execute(ctx context.Context, cmd Command) (Result, error)
}

type useCase struct {
	clock           timewrapper.ClockInterface
	analyzerFactory AnalyzerFactory
	repository      ParsedEmailRepository
	log             logger.Interface
}

// NewUseCase creates a mailanalysis use case.
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

// Execute validates the command, analyzes each email, and persists ParsedEmail history.
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
		result.AnalyzedEmailCount++

		if len(output.ParsedEmails) == 0 {
			result.Failures = append(result.Failures, domain.MessageFailure{
				EmailID:           email.EmailID,
				ExternalMessageID: email.ExternalMessageID,
				Stage:             domain.FailureStageResponseParse,
				Code:              domain.FailureCodeAnalysisResponseEmpty,
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
			})
			continue
		}

		for _, record := range records {
			result.ParsedEmailIDs = append(result.ParsedEmailIDs, record.ID)
		}
		result.ParsedEmailCount += len(records)
	}

	reqLog.Info("email_analysis_succeeded",
		logger.UserID(cmd.UserID),
		logger.Int("input_email_count", len(cmd.Emails)),
		logger.Int("analyzed_email_count", result.AnalyzedEmailCount),
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

func failureForAnalyzeError(email EmailForAnalysisTarget, err error) domain.MessageFailure {
	if errors.Is(err, domain.ErrAnalysisResponseInvalid) {
		return domain.MessageFailure{
			EmailID:           email.EmailID,
			ExternalMessageID: email.ExternalMessageID,
			Stage:             domain.FailureStageResponseParse,
			Code:              domain.FailureCodeAnalysisResponseInvalid,
		}
	}

	return domain.MessageFailure{
		EmailID:           email.EmailID,
		ExternalMessageID: email.ExternalMessageID,
		Stage:             domain.FailureStageAnalyze,
		Code:              domain.FailureCodeAnalysisFailed,
	}
}
