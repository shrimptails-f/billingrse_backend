package application

import (
	cd "business/internal/common/domain"
	"business/internal/library/logger"
	mfdomain "business/internal/mailfetch/domain"
	"context"
	"fmt"
	"strings"
)

// ConnectionRepository resolves a user-owned and fetchable mail-account connection.
type ConnectionRepository interface {
	FindUsableConnection(ctx context.Context, userID, connectionID uint) (mfdomain.ConnectionRef, error)
}

// MailFetcherFactory creates a provider-specific fetcher for the resolved connection.
type MailFetcherFactory interface {
	Create(ctx context.Context, conn mfdomain.ConnectionRef) (MailFetcher, error)
}

// MailFetcher loads provider messages and normalizes them into fetched-email DTOs.
type MailFetcher interface {
	Fetch(ctx context.Context, cond mfdomain.FetchCondition) ([]cd.FetchedEmailDTO, []mfdomain.MessageFailure, error)
}

// EmailRepository persists fetched email metadata idempotently.
type EmailRepository interface {
	SaveAllIfAbsent(ctx context.Context, userID uint, source mfdomain.EmailSource, dtos []cd.FetchedEmailDTO) ([]mfdomain.SaveResult, []mfdomain.MessageFailure, error)
}

// Command is the input contract for the manual mail fetch stage.
type Command struct {
	UserID       uint
	ConnectionID uint
	Condition    mfdomain.FetchCondition
}

// Result is the output contract for the manual mail fetch stage.
type Result struct {
	Provider            string
	AccountIdentifier   string
	MatchedMessageCount int
	CreatedEmailIDs     []uint
	ExistingEmailIDs    []uint
	Failures            []mfdomain.MessageFailure
}

// UseCase executes the manual mail fetch stage.
type UseCase interface {
	Execute(ctx context.Context, cmd Command) (Result, error)
}

type useCase struct {
	connectionRepo ConnectionRepository
	fetcherFactory MailFetcherFactory
	emailRepo      EmailRepository
	log            logger.Interface
}

// NewUseCase creates a manual mail fetch use case.
func NewUseCase(
	connectionRepo ConnectionRepository,
	fetcherFactory MailFetcherFactory,
	emailRepo EmailRepository,
	log logger.Interface,
) UseCase {
	if log == nil {
		log = logger.NewNop()
	}
	return &useCase{
		connectionRepo: connectionRepo,
		fetcherFactory: fetcherFactory,
		emailRepo:      emailRepo,
		log:            log.With(logger.Component("manual_mail_fetch_usecase")),
	}
}

// Execute validates the command, fetches provider emails, and saves them idempotently.
func (uc *useCase) Execute(ctx context.Context, cmd Command) (Result, error) {
	if ctx == nil {
		return Result{}, logger.ErrNilContext
	}

	reqLog := uc.log
	if withContext, err := uc.log.WithContext(ctx); err == nil {
		reqLog = withContext
	}

	if err := validateCommand(cmd); err != nil {
		return Result{}, err
	}

	cmd.Condition = cmd.Condition.Normalize()

	conn, err := uc.connectionRepo.FindUsableConnection(ctx, cmd.UserID, cmd.ConnectionID)
	if err != nil {
		return Result{}, err
	}

	fetcher, err := uc.fetcherFactory.Create(ctx, conn)
	if err != nil {
		return Result{}, err
	}

	dtos, failures, err := fetcher.Fetch(ctx, cmd.Condition)
	if err != nil {
		return Result{}, err
	}

	result := Result{
		Provider:            conn.Provider,
		AccountIdentifier:   conn.AccountIdentifier,
		MatchedMessageCount: len(dtos),
		Failures:            append([]mfdomain.MessageFailure{}, failures...),
	}

	seenMessageIDs := make(map[string]struct{}, len(dtos))
	source := conn.Source()
	saveTargets := make([]cd.FetchedEmailDTO, 0, len(dtos))

	for _, dto := range dtos {
		externalMessageID := strings.TrimSpace(dto.ID)
		if externalMessageID == "" || dto.Date.IsZero() {
			result.Failures = append(result.Failures, mfdomain.MessageFailure{
				ExternalMessageID: externalMessageID,
				Stage:             mfdomain.FailureStageNormalize,
				Code:              mfdomain.FailureCodeInvalidFetchedEmail,
			})
			continue
		}
		if _, seen := seenMessageIDs[externalMessageID]; seen {
			continue
		}
		seenMessageIDs[externalMessageID] = struct{}{}
		dto.ID = externalMessageID
		saveTargets = append(saveTargets, dto)
	}

	saveResults, saveFailures, saveErr := uc.emailRepo.SaveAllIfAbsent(ctx, cmd.UserID, source, saveTargets)
	if saveErr != nil {
		reqLog.Error("manual_mail_fetch_save_failed",
			logger.UserID(cmd.UserID),
			logger.Uint("connection_id", cmd.ConnectionID),
			logger.String("provider", conn.Provider),
			logger.Err(saveErr),
		)
		return Result{}, fmt.Errorf("failed to save fetched emails: %w", saveErr)
	}

	result.Failures = append(result.Failures, saveFailures...)

	for _, saveResult := range saveResults {
		switch saveResult.Status {
		case mfdomain.SaveStatusCreated:
			result.CreatedEmailIDs = append(result.CreatedEmailIDs, saveResult.EmailID)
		case mfdomain.SaveStatusExisting:
			result.ExistingEmailIDs = append(result.ExistingEmailIDs, saveResult.EmailID)
		default:
			reqLog.Error("manual_mail_fetch_save_status_invalid",
				logger.String("external_message_id", saveResult.ExternalMessageID),
				logger.String("save_status", string(saveResult.Status)),
			)
			result.Failures = append(result.Failures, mfdomain.MessageFailure{
				ExternalMessageID: saveResult.ExternalMessageID,
				Stage:             mfdomain.FailureStageSave,
				Code:              mfdomain.FailureCodeEmailSaveFailed,
			})
		}
	}

	reqLog.Info("manual_mail_fetch_succeeded",
		logger.UserID(cmd.UserID),
		logger.Uint("connection_id", cmd.ConnectionID),
		logger.String("provider", result.Provider),
		logger.Int("matched_message_count", result.MatchedMessageCount),
		logger.Int("created_email_count", len(result.CreatedEmailIDs)),
		logger.Int("existing_email_count", len(result.ExistingEmailIDs)),
		logger.Int("failure_count", len(result.Failures)),
	)

	return result, nil
}

func validateCommand(cmd Command) error {
	if cmd.UserID == 0 {
		return fmt.Errorf("%w: user_id is required", mfdomain.ErrInvalidCommand)
	}
	if cmd.ConnectionID == 0 {
		return fmt.Errorf("%w: connection_id is required", mfdomain.ErrInvalidCommand)
	}
	if err := cmd.Condition.Validate(); err != nil {
		return err
	}
	return nil
}
