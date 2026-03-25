package application

import (
	cd "business/internal/common/domain"
	"business/internal/library/logger"
	mfdomain "business/internal/mailfetch/domain"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
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

// CreatedEmail is a downstream-facing payload for newly persisted emails.
type CreatedEmail struct {
	EmailID           uint
	ExternalMessageID string
	Subject           string
	From              string
	To                []string
	Date              time.Time
	Body              string
	BodyDigest        string
}

// Result is the output contract for the manual mail fetch stage.
type Result struct {
	Provider            string
	AccountIdentifier   string
	MatchedMessageCount int
	CreatedEmailIDs     []uint
	CreatedEmails       []CreatedEmail
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
	saveTargetsByMessageID := make(map[string]cd.FetchedEmailDTO, len(dtos))

	for _, dto := range dtos {
		externalMessageID := strings.TrimSpace(dto.ID)
		if externalMessageID == "" || dto.Date.IsZero() {
			result.Failures = append(result.Failures, mfdomain.MessageFailure{
				ExternalMessageID: externalMessageID,
				Stage:             mfdomain.FailureStageNormalize,
				Code:              mfdomain.FailureCodeInvalidFetchedEmail,
				Message:           normalizeFailureMessage(externalMessageID, dto.Date.IsZero()),
			})
			continue
		}
		if _, seen := seenMessageIDs[externalMessageID]; seen {
			result.Failures = append(result.Failures, mfdomain.MessageFailure{
				ExternalMessageID: externalMessageID,
				Stage:             mfdomain.FailureStageNormalize,
				Code:              mfdomain.FailureCodeDuplicateExternalMessageID,
				Message:           duplicateFailureMessage(externalMessageID),
			})
			continue
		}
		seenMessageIDs[externalMessageID] = struct{}{}
		dto.ID = externalMessageID
		dto.BodyDigest = computeBodyDigest(dto.Body)
		saveTargets = append(saveTargets, dto)
		saveTargetsByMessageID[externalMessageID] = dto
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
			dto, ok := saveTargetsByMessageID[saveResult.ExternalMessageID]
			if !ok {
				reqLog.Error("manual_mail_fetch_created_email_payload_missing",
					logger.Uint("email_id", saveResult.EmailID),
					logger.String("external_message_id", saveResult.ExternalMessageID),
				)
				continue
			}
			result.CreatedEmails = append(result.CreatedEmails, CreatedEmail{
				EmailID:           saveResult.EmailID,
				ExternalMessageID: saveResult.ExternalMessageID,
				Subject:           dto.Subject,
				From:              dto.From,
				To:                append([]string(nil), dto.To...),
				Date:              dto.Date,
				Body:              dto.Body,
				BodyDigest:        dto.BodyDigest,
			})
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
				Message:           saveStatusFailureMessage(saveResult.ExternalMessageID, string(saveResult.Status)),
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

func computeBodyDigest(body string) string {
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:])
}

func normalizeFailureMessage(externalMessageID string, missingDate bool) string {
	id := externalMessageIDOrFallback(externalMessageID)
	if missingDate {
		return "取得メール(" + id + ")の受信日時が不正でした。"
	}
	return "取得メール(" + id + ")のIDまたは必須項目が不正でした。"
}

func duplicateFailureMessage(externalMessageID string) string {
	return "取得結果に重複したメールID(" + externalMessageIDOrFallback(externalMessageID) + ")が含まれていました。"
}

func saveStatusFailureMessage(externalMessageID, saveStatus string) string {
	return "取得メール(" + externalMessageIDOrFallback(externalMessageID) + ")の保存結果が不正でした。status=" + strings.TrimSpace(saveStatus)
}

func externalMessageIDOrFallback(externalMessageID string) string {
	externalMessageID = strings.TrimSpace(externalMessageID)
	if externalMessageID == "" {
		return "unknown"
	}
	return externalMessageID
}
