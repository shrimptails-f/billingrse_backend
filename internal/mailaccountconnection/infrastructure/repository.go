package infrastructure

import (
	"business/internal/library/logger"
	"business/internal/mailaccountconnection/application"
	"business/internal/mailaccountconnection/domain"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Repository provides database access for mail account connections.
type Repository struct {
	db  *gorm.DB
	log logger.Interface
}

// NewRepository creates a new Repository.
func NewRepository(db *gorm.DB, log logger.Interface) *Repository {
	if log == nil {
		log = logger.NewNop()
	}
	return &Repository{
		db:  db,
		log: log.With(logger.Component("mail_account_connection_repository")),
	}
}

var _ application.Repository = (*Repository)(nil)

const pendingGmailAddressPrefix = "__pending_state__:"

// credentialRecord maps to the email_credentials table.
type credentialRecord struct {
	ID                  uint       `gorm:"column:id;primaryKey;autoIncrement"`
	UserID              uint       `gorm:"column:user_id;not null"`
	Type                string     `gorm:"column:type;not null"`
	GmailAddress        string     `gorm:"column:gmail_address;not null"`
	KeyVersion          int16      `gorm:"column:key_version;not null;default:1"`
	AccessToken         string     `gorm:"column:access_token;not null"`
	AccessTokenDigest   string     `gorm:"column:access_token_digest;not null"`
	RefreshToken        string     `gorm:"column:refresh_token;not null"`
	RefreshTokenDigest  string     `gorm:"column:refresh_token_digest;not null"`
	TokenExpiry         *time.Time `gorm:"column:token_expiry"`
	OAuthState          *string    `gorm:"column:o_auth_state"`
	OAuthStateExpiresAt *time.Time `gorm:"column:o_auth_state_expires_at"`
	CreatedAt           time.Time  `gorm:"column:created_at"`
	UpdatedAt           time.Time  `gorm:"column:updated_at"`
}

func (credentialRecord) TableName() string {
	return "email_credentials"
}

func (r *Repository) SavePendingState(ctx context.Context, ps domain.OAuthPendingState) error {
	rec := credentialRecord{
		UserID:              ps.UserID,
		Type:                "gmail",
		GmailAddress:        pendingGmailAddressForState(ps.State),
		KeyVersion:          1,
		AccessToken:         "",
		AccessTokenDigest:   "",
		RefreshToken:        "",
		RefreshTokenDigest:  "",
		OAuthState:          stringPtr(ps.State),
		OAuthStateExpiresAt: timePtr(ps.ExpiresAt),
		CreatedAt:           ps.CreatedAt,
		UpdatedAt:           ps.CreatedAt,
	}
	if err := r.db.WithContext(ctx).Create(&rec).Error; err != nil {
		logDBQueryFailed(r.log, "email_credentials", "create_pending_state", err)
		return fmt.Errorf("failed to save pending state: %w", err)
	}
	return nil
}

func (r *Repository) FindPendingStateByState(ctx context.Context, state string) (domain.OAuthPendingState, error) {
	var rec credentialRecord
	err := r.db.WithContext(ctx).
		Where("o_auth_state = ?", state).
		First(&rec).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.OAuthPendingState{}, domain.ErrPendingStateNotFound
		}
		logDBQueryFailed(r.log, "email_credentials", "find_pending_state_by_state", err)
		return domain.OAuthPendingState{}, fmt.Errorf("failed to find pending state: %w", err)
	}
	if rec.OAuthState == nil || rec.OAuthStateExpiresAt == nil {
		return domain.OAuthPendingState{}, domain.ErrPendingStateNotFound
	}
	return domain.OAuthPendingState{
		ID:        rec.ID,
		UserID:    rec.UserID,
		State:     *rec.OAuthState,
		ExpiresAt: *rec.OAuthStateExpiresAt,
		CreatedAt: rec.CreatedAt,
	}, nil
}

func (r *Repository) ConsumePendingState(ctx context.Context, id uint, consumedAt time.Time) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND o_auth_state IS NOT NULL", id).
		Delete(&credentialRecord{})
	if result.Error != nil {
		logDBQueryFailed(r.log, "email_credentials", "consume_pending_state", result.Error)
		return fmt.Errorf("failed to consume pending state: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("pending state already consumed or not found")
	}
	return nil
}

func (r *Repository) FindCredentialByUserAndGmail(ctx context.Context, userID uint, gmailAddress string) (domain.EmailCredential, error) {
	normalized := strings.ToLower(strings.TrimSpace(gmailAddress))
	var rec credentialRecord
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND type = ? AND gmail_address = ? AND o_auth_state IS NULL", userID, "gmail", normalized).
		First(&rec).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.EmailCredential{}, domain.ErrCredentialNotFound
		}
		logDBQueryFailed(r.log, "email_credentials", "find_by_user_gmail", err)
		return domain.EmailCredential{}, fmt.Errorf("failed to find credential: %w", err)
	}
	return toDomainCredential(rec), nil
}

func (r *Repository) ListCredentialsByUser(ctx context.Context, userID uint) ([]domain.EmailCredential, error) {
	var records []credentialRecord
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND o_auth_state IS NULL", userID).
		Order("created_at ASC, id ASC").
		Find(&records).Error; err != nil {
		logDBQueryFailed(r.log, "email_credentials", "list_by_user", err)
		return nil, fmt.Errorf("failed to list credentials: %w", err)
	}

	credentials := make([]domain.EmailCredential, 0, len(records))
	for _, record := range records {
		credentials = append(credentials, toDomainCredential(record))
	}
	return credentials, nil
}

func (r *Repository) DeleteCredentialByIDAndUser(ctx context.Context, credentialID, userID uint) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ? AND o_auth_state IS NULL", credentialID, userID).
		Delete(&credentialRecord{})
	if result.Error != nil {
		logDBQueryFailed(r.log, "email_credentials", "delete_by_id_and_user", result.Error)
		return fmt.Errorf("failed to delete credential: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrCredentialNotFound
	}
	return nil
}

func (r *Repository) CreateCredential(ctx context.Context, cred domain.EmailCredential) error {
	rec := toCredentialRecord(cred)
	if err := r.db.WithContext(ctx).Create(&rec).Error; err != nil {
		logDBQueryFailed(r.log, "email_credentials", "create", err)
		return fmt.Errorf("failed to create credential: %w", err)
	}
	return nil
}

func (r *Repository) UpdateCredentialTokens(ctx context.Context, cred domain.EmailCredential) error {
	result := r.db.WithContext(ctx).
		Model(&credentialRecord{}).
		Where("id = ?", cred.ID).
		Updates(map[string]interface{}{
			"access_token":         cred.AccessToken,
			"access_token_digest":  cred.AccessTokenDigest,
			"refresh_token":        cred.RefreshToken,
			"refresh_token_digest": cred.RefreshTokenDigest,
			"token_expiry":         cred.TokenExpiry,
			"updated_at":           cred.UpdatedAt,
		})
	if result.Error != nil {
		logDBQueryFailed(r.log, "email_credentials", "update", result.Error)
		return fmt.Errorf("failed to update credential: %w", result.Error)
	}
	return nil
}

func toDomainCredential(rec credentialRecord) domain.EmailCredential {
	return domain.EmailCredential{
		ID:                 rec.ID,
		UserID:             rec.UserID,
		Type:               rec.Type,
		GmailAddress:       rec.GmailAddress,
		KeyVersion:         rec.KeyVersion,
		AccessToken:        rec.AccessToken,
		AccessTokenDigest:  rec.AccessTokenDigest,
		RefreshToken:       rec.RefreshToken,
		RefreshTokenDigest: rec.RefreshTokenDigest,
		TokenExpiry:        rec.TokenExpiry,
		CreatedAt:          rec.CreatedAt,
		UpdatedAt:          rec.UpdatedAt,
	}
}

func pendingGmailAddressForState(state string) string {
	return pendingGmailAddressPrefix + state
}

func stringPtr(v string) *string {
	return &v
}

func timePtr(v time.Time) *time.Time {
	return &v
}

func toCredentialRecord(cred domain.EmailCredential) credentialRecord {
	return credentialRecord{
		ID:                 cred.ID,
		UserID:             cred.UserID,
		Type:               cred.Type,
		GmailAddress:       cred.GmailAddress,
		KeyVersion:         cred.KeyVersion,
		AccessToken:        cred.AccessToken,
		AccessTokenDigest:  cred.AccessTokenDigest,
		RefreshToken:       cred.RefreshToken,
		RefreshTokenDigest: cred.RefreshTokenDigest,
		TokenExpiry:        cred.TokenExpiry,
		CreatedAt:          cred.CreatedAt,
		UpdatedAt:          cred.UpdatedAt,
	}
}

func logDBQueryFailed(log logger.Interface, table, operation string, err error) {
	if log == nil {
		log = logger.NewNop()
	}
	log.Error("db_query_failed",
		logger.String("table", table),
		logger.String("operation", operation),
		logger.Err(err),
	)
}
