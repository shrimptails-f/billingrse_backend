package infrastructure

import (
	"business/internal/emailcredential/application"
	"business/internal/emailcredential/domain"
	"business/internal/library/logger"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Repository provides database access for email credentials.
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
		log: log.With(logger.Component("email_credential_repository")),
	}
}

var _ application.Repository = (*Repository)(nil)

// pendingStateRecord maps to the oauth_pending_states table.
type pendingStateRecord struct {
	ID         uint       `gorm:"column:id;primaryKey;autoIncrement"`
	UserID     uint       `gorm:"column:user_id;not null"`
	State      string     `gorm:"column:state;type:varchar(255);not null;uniqueIndex"`
	ExpiresAt  time.Time  `gorm:"column:expires_at;not null"`
	ConsumedAt *time.Time `gorm:"column:consumed_at"`
	CreatedAt  time.Time  `gorm:"column:created_at;not null"`
}

func (pendingStateRecord) TableName() string {
	return "oauth_pending_states"
}

// credentialRecord maps to the email_credentials table.
type credentialRecord struct {
	ID                 uint       `gorm:"column:id;primaryKey;autoIncrement"`
	UserID             uint       `gorm:"column:user_id;not null"`
	Type               string     `gorm:"column:type;not null"`
	GmailAddress       string     `gorm:"column:gmail_address;not null"`
	KeyVersion         int16      `gorm:"column:key_version;not null;default:1"`
	AccessToken        string     `gorm:"column:access_token;not null"`
	AccessTokenDigest  string     `gorm:"column:access_token_digest;not null"`
	RefreshToken       string     `gorm:"column:refresh_token;not null"`
	RefreshTokenDigest string     `gorm:"column:refresh_token_digest;not null"`
	TokenExpiry        *time.Time `gorm:"column:token_expiry"`
	CreatedAt          time.Time  `gorm:"column:created_at"`
	UpdatedAt          time.Time  `gorm:"column:updated_at"`
}

func (credentialRecord) TableName() string {
	return "email_credentials"
}

func (r *Repository) SavePendingState(ctx context.Context, ps domain.OAuthPendingState) error {
	rec := pendingStateRecord{
		UserID:    ps.UserID,
		State:     ps.State,
		ExpiresAt: ps.ExpiresAt,
		CreatedAt: ps.CreatedAt,
	}
	if err := r.db.WithContext(ctx).Create(&rec).Error; err != nil {
		logDBQueryFailed(r.log, "oauth_pending_states", "create", err)
		return fmt.Errorf("failed to save pending state: %w", err)
	}
	return nil
}

func (r *Repository) FindPendingStateByState(ctx context.Context, state string) (domain.OAuthPendingState, error) {
	var rec pendingStateRecord
	err := r.db.WithContext(ctx).Where("state = ?", state).First(&rec).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.OAuthPendingState{}, domain.ErrPendingStateNotFound
		}
		logDBQueryFailed(r.log, "oauth_pending_states", "find_by_state", err)
		return domain.OAuthPendingState{}, fmt.Errorf("failed to find pending state: %w", err)
	}
	return domain.OAuthPendingState{
		ID:         rec.ID,
		UserID:     rec.UserID,
		State:      rec.State,
		ExpiresAt:  rec.ExpiresAt,
		ConsumedAt: rec.ConsumedAt,
		CreatedAt:  rec.CreatedAt,
	}, nil
}

func (r *Repository) ConsumePendingState(ctx context.Context, id uint, consumedAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&pendingStateRecord{}).
		Where("id = ? AND consumed_at IS NULL", id).
		Update("consumed_at", consumedAt)
	if result.Error != nil {
		logDBQueryFailed(r.log, "oauth_pending_states", "consume", result.Error)
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
		Where("user_id = ? AND type = ? AND gmail_address = ?", userID, "gmail", normalized).
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

func toCredentialRecord(cred domain.EmailCredential) credentialRecord {
	return credentialRecord{
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
