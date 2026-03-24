package application

import (
	"business/internal/library/logger"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrInvalidCommand is returned when required workflow command fields are missing.
	ErrInvalidCommand = errors.New("manual mail workflow command is invalid")
	// ErrFetchConditionInvalid is returned when the workflow fetch condition is malformed.
	ErrFetchConditionInvalid = errors.New("manual mail workflow fetch condition is invalid")
)

// FetchCondition represents the mail-fetch condition accepted by the workflow endpoint.
type FetchCondition struct {
	LabelName string
	Since     time.Time
	Until     time.Time
}

// Normalize trims free-form fields while preserving timestamps.
func (c FetchCondition) Normalize() FetchCondition {
	c.LabelName = strings.TrimSpace(c.LabelName)
	return c
}

// Validate enforces the minimum fetch-condition invariants for the workflow.
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

// Command is the input contract for the manual mail workflow.
type Command struct {
	UserID       uint
	ConnectionID uint
	Condition    FetchCondition
}

// CreatedEmail is the workflow-internal payload passed from fetch to analysis.
type CreatedEmail struct {
	EmailID           uint
	ExternalMessageID string
	Subject           string
	From              string
	To                []string
	ReceivedAt        time.Time
	Body              string
}

// FetchFailure describes a partial failure returned from the fetch stage.
type FetchFailure struct {
	ExternalMessageID string
	Stage             string
	Code              string
}

// AnalysisFailure describes a partial failure returned from the analysis stage.
type AnalysisFailure struct {
	EmailID           uint
	ExternalMessageID string
	Stage             string
	Code              string
}

// FetchResult is the normalized output contract of the fetch stage.
type FetchResult struct {
	Provider            string
	AccountIdentifier   string
	MatchedMessageCount int
	CreatedEmailIDs     []uint
	CreatedEmails       []CreatedEmail
	ExistingEmailIDs    []uint
	Failures            []FetchFailure
}

// AnalyzeResult is the normalized output contract of the analysis stage.
type AnalyzeResult struct {
	ParsedEmailIDs     []uint
	AnalyzedEmailCount int
	ParsedEmailCount   int
	Failures           []AnalysisFailure
}

// Result is the combined output returned by the workflow.
type Result struct {
	Fetch    FetchResult
	Analysis AnalyzeResult
}

// FetchCommand is the fetch-stage input contract owned by the workflow.
type FetchCommand struct {
	UserID       uint
	ConnectionID uint
	Condition    FetchCondition
}

// AnalyzeCommand is the analysis-stage input contract owned by the workflow.
type AnalyzeCommand struct {
	UserID uint
	Emails []CreatedEmail
}

// FetchStage executes the mailfetch stage for the workflow.
type FetchStage interface {
	Execute(ctx context.Context, cmd FetchCommand) (FetchResult, error)
}

// AnalyzeStage executes the mailanalysis stage for the workflow.
type AnalyzeStage interface {
	Execute(ctx context.Context, cmd AnalyzeCommand) (AnalyzeResult, error)
}

// UseCase executes the manual mail workflow.
type UseCase interface {
	Execute(ctx context.Context, cmd Command) (Result, error)
}

type useCase struct {
	fetchStage   FetchStage
	analyzeStage AnalyzeStage
	log          logger.Interface
}

// NewUseCase creates a manual mail workflow use case.
func NewUseCase(fetchStage FetchStage, analyzeStage AnalyzeStage, log logger.Interface) UseCase {
	if log == nil {
		log = logger.NewNop()
	}

	return &useCase{
		fetchStage:   fetchStage,
		analyzeStage: analyzeStage,
		log:          log.With(logger.Component("manual_mail_workflow_usecase")),
	}
}

// Execute runs mailfetch and then mailanalysis using the newly created emails.
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
			logger.Int("fetch_failure_count", len(fetchResult.Failures)),
			logger.Int("analysis_failure_count", 0),
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

	reqLog.Info("manual_mail_workflow_succeeded",
		logger.UserID(cmd.UserID),
		logger.Uint("connection_id", cmd.ConnectionID),
		logger.Int("created_email_count", len(fetchResult.CreatedEmailIDs)),
		logger.Int("parsed_email_count", len(analysisResult.ParsedEmailIDs)),
		logger.Int("fetch_failure_count", len(fetchResult.Failures)),
		logger.Int("analysis_failure_count", len(analysisResult.Failures)),
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
