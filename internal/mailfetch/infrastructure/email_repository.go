package infrastructure

import (
	cd "business/internal/common/domain"
	"business/internal/library/logger"
	"business/internal/library/timewrapper"
	mfdomain "business/internal/mailfetch/domain"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const saveBatchSize = 20

type emailRecord struct {
	ID                uint      `gorm:"column:id;primaryKey;autoIncrement"`
	UserID            uint      `gorm:"column:user_id;not null;uniqueIndex:uni_emails_user_message"`
	Provider          string    `gorm:"column:provider;size:50;not null"`
	AccountIdentifier string    `gorm:"column:account_identifier;size:255;not null"`
	ExternalMessageID string    `gorm:"column:external_message_id;size:255;not null;uniqueIndex:uni_emails_user_message"`
	Subject           string    `gorm:"column:subject;type:text;not null"`
	FromRaw           string    `gorm:"column:from_raw;type:text;not null"`
	ToJSON            string    `gorm:"column:to_json;type:json;not null"`
	BodyDigest        string    `gorm:"column:body_digest;size:64;not null"`
	ReceivedAt        time.Time `gorm:"column:received_at;not null"`
	CreatedRunID      *string   `gorm:"column:created_run_id;size:36"`
	CreatedAt         time.Time `gorm:"column:created_at;not null"`
	UpdatedAt         time.Time `gorm:"column:updated_at;not null"`
}

func (emailRecord) TableName() string {
	return "emails"
}

// GormEmailRepositoryAdapter persists fetched email metadata into the emails table.
type GormEmailRepositoryAdapter struct {
	db    *gorm.DB
	clock timewrapper.ClockInterface
	log   logger.Interface
}

// NewGormEmailRepositoryAdapter creates a Gorm-backed email repository adapter.
func NewGormEmailRepositoryAdapter(
	db *gorm.DB,
	clock timewrapper.ClockInterface,
	log logger.Interface,
) *GormEmailRepositoryAdapter {
	if clock == nil {
		clock = timewrapper.NewClock()
	}
	if log == nil {
		log = logger.NewNop()
	}
	return &GormEmailRepositoryAdapter{
		db:    db,
		clock: clock,
		log:   log.With(logger.Component("manual_mail_fetch_email_repository")),
	}
}

type preparedEmail struct {
	email  cd.Email
	toJSON string
}

type batchInsertError struct {
	err error
}

func (e *batchInsertError) Error() string {
	return e.err.Error()
}

func (e *batchInsertError) Unwrap() error {
	return e.err
}

// SaveAllIfAbsent persists fetched emails once per user/external-message tuple.
func (r *GormEmailRepositoryAdapter) SaveAllIfAbsent(ctx context.Context, userID uint, source mfdomain.EmailSource, dtos []cd.FetchedEmailDTO) ([]mfdomain.SaveResult, []mfdomain.MessageFailure, error) {
	if err := source.Validate(); err != nil {
		return nil, nil, err
	}
	if len(dtos) == 0 {
		return nil, nil, nil
	}

	prepared, failures, err := prepareEmails(userID, dtos)
	if err != nil {
		return nil, failures, err
	}
	if len(prepared) == 0 {
		return nil, failures, nil
	}

	results, saveFailures, err := r.savePreparedEmailChunks(ctx, userID, source, prepared)
	if err != nil {
		return nil, append(failures, saveFailures...), err
	}
	return results, append(failures, saveFailures...), nil
}

func prepareEmails(userID uint, dtos []cd.FetchedEmailDTO) ([]preparedEmail, []mfdomain.MessageFailure, error) {
	prepared := make([]preparedEmail, 0, len(dtos))
	failures := make([]mfdomain.MessageFailure, 0)
	seenMessageIDs := make(map[string]struct{}, len(dtos))

	for _, dto := range dtos {
		dto.ID = strings.TrimSpace(dto.ID)
		if dto.ID == "" {
			failures = append(failures, mfdomain.MessageFailure{
				Stage:   mfdomain.FailureStageNormalize,
				Code:    mfdomain.FailureCodeInvalidFetchedEmail,
				Message: invalidFetchedEmailMessage("unknown"),
			})
			continue
		}
		if _, seen := seenMessageIDs[dto.ID]; seen {
			failures = append(failures, mfdomain.MessageFailure{
				ExternalMessageID: dto.ID,
				Stage:             mfdomain.FailureStageNormalize,
				Code:              mfdomain.FailureCodeDuplicateExternalMessageID,
				Message:           duplicateExternalMessageMessage(dto.ID),
			})
			continue
		}
		seenMessageIDs[dto.ID] = struct{}{}

		email, err := cd.NewEmailFromFetchedDTO(userID, dto)
		if err != nil {
			failures = append(failures, mfdomain.MessageFailure{
				ExternalMessageID: dto.ID,
				Stage:             mfdomain.FailureStageNormalize,
				Code:              mfdomain.FailureCodeInvalidFetchedEmail,
				Message:           invalidFetchedEmailMessage(dto.ID),
			})
			continue
		}

		toJSON, err := marshalRecipients(email.To)
		if err != nil {
			return nil, failures, fmt.Errorf("failed to encode recipients: %w", err)
		}

		prepared = append(prepared, preparedEmail{
			email:  email,
			toJSON: toJSON,
		})
	}

	return prepared, failures, nil
}

func (r *GormEmailRepositoryAdapter) savePreparedEmailChunks(
	ctx context.Context,
	userID uint,
	source mfdomain.EmailSource,
	prepared []preparedEmail,
) ([]mfdomain.SaveResult, []mfdomain.MessageFailure, error) {
	reqLog := r.log
	if withContext, err := r.log.WithContext(ctx); err == nil {
		reqLog = withContext
	}

	results := make([]mfdomain.SaveResult, 0, len(prepared))
	failures := make([]mfdomain.MessageFailure, 0)

	for start := 0; start < len(prepared); start += saveBatchSize {
		end := start + saveBatchSize
		if end > len(prepared) {
			end = len(prepared)
		}

		chunk := prepared[start:end]
		//nolint:nplusonecheck // Chunked saves are intentional to allow partial success handling per batch.
		chunkResults, err := r.savePreparedEmails(ctx, userID, source, chunk)
		if err != nil {
			var insertErr *batchInsertError
			if errors.As(err, &insertErr) {
				reqLog.Error("manual_mail_fetch_email_batch_insert_failed",
					logger.UserID(userID),
					logger.String("provider", strings.TrimSpace(source.Provider)),
					logger.Int("chunk_start_index", start),
					logger.Int("chunk_size", len(chunk)),
					logger.Any("chunk_external_message_ids", chunkExternalMessageIDs(chunk)),
					logger.Err(insertErr),
				)
				failures = append(failures, buildChunkSaveFailures(chunk)...)
				continue
			}
			return nil, failures, err
		}

		results = append(results, chunkResults...)
	}

	return results, failures, nil
}

func chunkExternalMessageIDs(chunk []preparedEmail) []string {
	ids := make([]string, 0, len(chunk))
	for _, item := range chunk {
		ids = append(ids, item.email.ExternalMessageID)
	}
	return ids
}

func (r *GormEmailRepositoryAdapter) savePreparedEmails(
	ctx context.Context,
	userID uint,
	source mfdomain.EmailSource,
	prepared []preparedEmail,
) ([]mfdomain.SaveResult, error) {
	provider := strings.TrimSpace(source.Provider)
	accountIdentifier := strings.TrimSpace(source.AccountIdentifier)
	runID := uuid.NewString()
	externalMessageIDs := make([]string, 0, len(prepared))
	records := make([]emailRecord, 0, len(prepared))
	now := r.clock.Now().UTC()
	for _, item := range prepared {
		externalMessageIDs = append(externalMessageIDs, item.email.ExternalMessageID)
		records = append(records, emailRecord{
			UserID:            userID,
			Provider:          provider,
			AccountIdentifier: accountIdentifier,
			ExternalMessageID: item.email.ExternalMessageID,
			Subject:           item.email.Subject,
			FromRaw:           item.email.From,
			ToJSON:            item.toJSON,
			BodyDigest:        item.email.BodyDigest,
			ReceivedAt:        item.email.ReceivedAt,
			CreatedRunID:      &runID,
			CreatedAt:         now,
			UpdatedAt:         now,
		})
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(records) == 0 {
			return nil
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&records).Error; err != nil {
			return &batchInsertError{err: fmt.Errorf("failed to batch create emails: %w", err)}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	finalRecords, err := findByExternalIDs(r.db, ctx, userID, externalMessageIDs)
	if err != nil {
		return nil, err
	}
	finalMap := make(map[string]emailRecord, len(finalRecords))
	for _, record := range finalRecords {
		finalMap[record.ExternalMessageID] = record
	}

	results := make([]mfdomain.SaveResult, 0, len(prepared))
	for _, item := range prepared {
		record, found := finalMap[item.email.ExternalMessageID]
		if !found {
			return nil, fmt.Errorf("email was not found after save: %s", item.email.ExternalMessageID)
		}
		status := mfdomain.SaveStatusExisting
		if record.CreatedRunID != nil && *record.CreatedRunID == runID {
			status = mfdomain.SaveStatusCreated
		}
		results = append(results, mfdomain.SaveResult{
			EmailID:           record.ID,
			ExternalMessageID: record.ExternalMessageID,
			Status:            status,
		})
	}
	return results, nil
}

func buildChunkSaveFailures(chunk []preparedEmail) []mfdomain.MessageFailure {
	failures := make([]mfdomain.MessageFailure, 0, len(chunk))
	for _, item := range chunk {
		failures = append(failures, mfdomain.MessageFailure{
			ExternalMessageID: item.email.ExternalMessageID,
			Stage:             mfdomain.FailureStageSave,
			Code:              mfdomain.FailureCodeEmailSaveFailed,
			Message:           emailSaveFailureMessage(item.email.ExternalMessageID),
		})
	}
	return failures
}

func invalidFetchedEmailMessage(externalMessageID string) string {
	externalMessageID = strings.TrimSpace(externalMessageID)
	if externalMessageID == "" {
		externalMessageID = "unknown"
	}
	return "取得メール(" + externalMessageID + ")の形式が不正でした。"
}

func duplicateExternalMessageMessage(externalMessageID string) string {
	externalMessageID = strings.TrimSpace(externalMessageID)
	if externalMessageID == "" {
		externalMessageID = "unknown"
	}
	return "取得バッチ内でメールID(" + externalMessageID + ")が重複していたため、後続処理をスキップしました。"
}

func emailSaveFailureMessage(externalMessageID string) string {
	externalMessageID = strings.TrimSpace(externalMessageID)
	if externalMessageID == "" {
		externalMessageID = "unknown"
	}
	return "取得メール(" + externalMessageID + ")の保存に失敗しました。"
}

func findByExternalIDs(
	db *gorm.DB,
	ctx context.Context,
	userID uint,
	externalMessageIDs []string,
) ([]emailRecord, error) {
	if len(externalMessageIDs) == 0 {
		return nil, nil
	}

	var records []emailRecord
	err := db.WithContext(ctx).
		Where(
			"user_id = ? AND external_message_id IN ?",
			userID,
			externalMessageIDs,
		).
		Find(&records).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find emails: %w", err)
	}
	return records, nil
}

func marshalRecipients(recipients []string) (string, error) {
	if recipients == nil {
		recipients = []string{}
	}
	encoded, err := json.Marshal(recipients)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}
