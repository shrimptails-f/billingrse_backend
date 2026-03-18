package infrastructure

import (
	"business/internal/auth/domain"
	"business/internal/library/logger"
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// CreateRefreshToken stores a new refresh token record.
func (r *Repository) CreateRefreshToken(ctx context.Context, token domain.RefreshToken) (domain.RefreshToken, error) {
	if ctx == nil {
		return domain.RefreshToken{}, logger.ErrNilContext
	}

	reqLog, logErr := r.logger.WithContext(ctx)
	if logErr != nil {
		return domain.RefreshToken{}, logErr
	}

	record := refreshTokenRecord{
		UserID:            token.UserID,
		TokenDigest:       token.TokenDigest,
		ExpiresAt:         token.ExpiresAt,
		LastUsedAt:        token.LastUsedAt,
		RevokedAt:         token.RevokedAt,
		ReplacedByTokenID: token.ReplacedByTokenID,
		CreatedAt:         token.CreatedAt,
	}

	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		logDBQueryFailed(reqLog, "auth_refresh_tokens", "create_refresh_token", err, logger.Uint("user_id", token.UserID))
		return domain.RefreshToken{}, fmt.Errorf("failed to create refresh token: %w", err)
	}

	token.ID = record.ID
	return token, nil
}

// FindActiveRefreshTokenByDigest retrieves an active refresh token by digest.
func (r *Repository) FindActiveRefreshTokenByDigest(ctx context.Context, digest string, now time.Time) (domain.RefreshToken, error) {
	if ctx == nil {
		return domain.RefreshToken{}, logger.ErrNilContext
	}

	reqLog, logErr := r.logger.WithContext(ctx)
	if logErr != nil {
		return domain.RefreshToken{}, logErr
	}

	var record refreshTokenRecord
	err := r.db.WithContext(ctx).
		Where("token_digest = ? AND revoked_at IS NULL AND expires_at > ?", digest, now).
		First(&record).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.RefreshToken{}, gorm.ErrRecordNotFound
		}
		logDBQueryFailed(reqLog, "auth_refresh_tokens", "find_active_refresh_token_by_digest", err)
		return domain.RefreshToken{}, fmt.Errorf("failed to find active refresh token: %w", err)
	}

	return refreshTokenRecordToDomain(record), nil
}

// RotateRefreshToken revokes the current refresh token and stores the replacement in a transaction.
func (r *Repository) RotateRefreshToken(ctx context.Context, currentID uint, next domain.RefreshToken, now time.Time) (domain.RefreshToken, error) {
	if ctx == nil {
		return domain.RefreshToken{}, logger.ErrNilContext
	}

	reqLog, logErr := r.logger.WithContext(ctx)
	if logErr != nil {
		return domain.RefreshToken{}, logErr
	}

	var created domain.RefreshToken
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current refreshTokenRecord
		if err := tx.Where("id = ? AND revoked_at IS NULL AND expires_at > ?", currentID, now).First(&current).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return gorm.ErrRecordNotFound
			}
			logDBQueryFailed(reqLog, "auth_refresh_tokens", "find_current_refresh_token", err, logger.Uint("token_id", currentID))
			return fmt.Errorf("failed to load current refresh token: %w", err)
		}

		nextRecord := refreshTokenRecord{
			UserID:      next.UserID,
			TokenDigest: next.TokenDigest,
			ExpiresAt:   next.ExpiresAt,
			CreatedAt:   next.CreatedAt,
		}

		if err := tx.Create(&nextRecord).Error; err != nil {
			logDBQueryFailed(reqLog, "auth_refresh_tokens", "create_refresh_token_rotation", err, logger.Uint("token_id", currentID))
			return fmt.Errorf("failed to create replacement refresh token: %w", err)
		}

		replacedByID := nextRecord.ID
		if err := tx.Model(&refreshTokenRecord{}).
			Where("id = ?", currentID).
			Updates(map[string]interface{}{
				"revoked_at":           now,
				"last_used_at":         now,
				"replaced_by_token_id": replacedByID,
			}).Error; err != nil {
			logDBQueryFailed(reqLog, "auth_refresh_tokens", "revoke_refresh_token_rotation", err, logger.Uint("token_id", currentID))
			return fmt.Errorf("failed to revoke current refresh token: %w", err)
		}

		created = refreshTokenRecordToDomain(nextRecord)
		created.Token = next.Token
		created.ReplacedByTokenID = nil
		return nil
	})

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.RefreshToken{}, gorm.ErrRecordNotFound
		}
		return domain.RefreshToken{}, err
	}

	return created, nil
}

// RevokeRefreshTokenByDigest revokes the refresh token matching the digest.
func (r *Repository) RevokeRefreshTokenByDigest(ctx context.Context, digest string, now time.Time) error {
	if ctx == nil {
		return logger.ErrNilContext
	}

	reqLog, logErr := r.logger.WithContext(ctx)
	if logErr != nil {
		return logErr
	}

	result := r.db.WithContext(ctx).
		Model(&refreshTokenRecord{}).
		Where("token_digest = ? AND revoked_at IS NULL", digest).
		Updates(map[string]interface{}{
			"revoked_at":   now,
			"last_used_at": now,
		})
	if result.Error != nil {
		logDBQueryFailed(reqLog, "auth_refresh_tokens", "revoke_refresh_token_by_digest", result.Error)
		return fmt.Errorf("failed to revoke refresh token: %w", result.Error)
	}

	return nil
}

func refreshTokenRecordToDomain(record refreshTokenRecord) domain.RefreshToken {
	return domain.RefreshToken{
		ID:                record.ID,
		UserID:            record.UserID,
		TokenDigest:       record.TokenDigest,
		ExpiresAt:         record.ExpiresAt,
		LastUsedAt:        record.LastUsedAt,
		RevokedAt:         record.RevokedAt,
		ReplacedByTokenID: record.ReplacedByTokenID,
		CreatedAt:         record.CreatedAt,
	}
}
