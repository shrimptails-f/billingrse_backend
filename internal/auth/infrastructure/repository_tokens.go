package infrastructure

import (
	"business/internal/auth/domain"
	"business/internal/library/logger"
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GetActiveTokenForUser retrieves an active (unconsumed and not expired) token for a user.
func (r *Repository) GetActiveTokenForUser(ctx context.Context, userID uint, now time.Time) (domain.EmailVerificationToken, error) {
	var record emailVerificationTokenRecord

	if ctx == nil {
		return domain.EmailVerificationToken{}, logger.ErrNilContext
	}

	reqLog, logErr := r.logger.WithContext(ctx)
	if logErr != nil {
		return domain.EmailVerificationToken{}, logErr
	}

	err := r.db.
		WithContext(ctx).
		Where("user_id = ? AND consumed_at IS NULL AND expires_at > ?", userID, now).
		Order("created_at DESC").
		First(&record).
		Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.EmailVerificationToken{}, gorm.ErrRecordNotFound
		}
		logDBQueryFailed(reqLog, "email_verification_tokens", "get_active_token_for_user", err,
			logger.Uint("user_id", userID),
			logger.String("now", now.Format(time.RFC3339)),
		)
		return domain.EmailVerificationToken{}, fmt.Errorf("failed to get active token: %w", err)
	}

	return domain.EmailVerificationToken{
		ID:                    record.ID,
		UserID:                record.UserID,
		Token:                 record.Token,
		ExpiresAt:             record.ExpiresAt,
		CreatedAt:             record.CreatedAt,
		ResendWindowStartedAt: record.ResendWindowStartedAt,
		ResendCount:           record.ResendCount,
		ConsumedAt:            record.ConsumedAt,
	}, nil
}

// CreateEmailVerificationToken creates or updates an email verification token.
// If a token already exists for the user, it will be replaced (upsert behavior).
func (r *Repository) CreateEmailVerificationToken(ctx context.Context, token domain.EmailVerificationToken) (domain.EmailVerificationToken, error) {
	if ctx == nil {
		return domain.EmailVerificationToken{}, logger.ErrNilContext
	}

	reqLog, logErr := r.logger.WithContext(ctx)
	if logErr != nil {
		return domain.EmailVerificationToken{}, logErr
	}

	record := emailVerificationTokenRecord{
		UserID:                token.UserID,
		Token:                 token.Token,
		ExpiresAt:             token.ExpiresAt,
		CreatedAt:             token.CreatedAt,
		ResendWindowStartedAt: token.ResendWindowStartedAt,
		ResendCount:           token.ResendCount,
		ConsumedAt:            token.ConsumedAt,
	}

	// Use Clauses(clause.OnConflict) for upsert behavior on user_id unique constraint
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"token":                    record.Token,
				"expires_at":               record.ExpiresAt,
				"created_at":               record.CreatedAt,
				"resend_window_started_at": record.ResendWindowStartedAt,
				"resend_count":             record.ResendCount,
				"consumed_at":              record.ConsumedAt,
			}),
		}).
		Create(&record).
		Error

	if err != nil {
		logDBQueryFailed(reqLog, "email_verification_tokens", "create_email_verification_token", err, logger.Uint("user_id", token.UserID))
		return domain.EmailVerificationToken{}, fmt.Errorf("failed to create email verification token: %w", err)
	}

	// Retrieve the inserted/updated record
	err = r.db.WithContext(ctx).
		Where("user_id = ?", token.UserID).
		First(&record).
		Error

	if err != nil {
		logDBQueryFailed(reqLog, "email_verification_tokens", "get_email_verification_token_after_upsert", err, logger.Uint("user_id", token.UserID))
		return domain.EmailVerificationToken{}, fmt.Errorf("failed to retrieve token after upsert: %w", err)
	}

	return domain.EmailVerificationToken{
		ID:                    record.ID,
		UserID:                record.UserID,
		Token:                 record.Token,
		ExpiresAt:             record.ExpiresAt,
		CreatedAt:             record.CreatedAt,
		ResendWindowStartedAt: record.ResendWindowStartedAt,
		ResendCount:           record.ResendCount,
		ConsumedAt:            record.ConsumedAt,
	}, nil
}

// InvalidateActiveTokens invalidates all active tokens for a user by setting consumed_at.
func (r *Repository) InvalidateActiveTokens(ctx context.Context, userID uint, consumedAt time.Time) error {
	if ctx == nil {
		return logger.ErrNilContext
	}

	reqLog, logErr := r.logger.WithContext(ctx)
	if logErr != nil {
		return logErr
	}

	err := r.db.WithContext(ctx).
		Model(&emailVerificationTokenRecord{}).
		Where("user_id = ? AND consumed_at IS NULL", userID).
		Update("consumed_at", consumedAt).
		Error

	if err != nil {
		logDBQueryFailed(reqLog, "email_verification_tokens", "invalidate_active_tokens", err, logger.Uint("user_id", userID))
		return fmt.Errorf("failed to invalidate active tokens: %w", err)
	}

	return nil
}

// GetEmailVerificationToken retrieves a verification token by token string.
func (r *Repository) GetEmailVerificationToken(ctx context.Context, token string) (domain.EmailVerificationToken, error) {
	var record emailVerificationTokenRecord
	if ctx == nil {
		return domain.EmailVerificationToken{}, logger.ErrNilContext
	}

	reqLog, logErr := r.logger.WithContext(ctx)
	if logErr != nil {
		return domain.EmailVerificationToken{}, logErr
	}

	err := r.db.
		WithContext(ctx).
		Where("token = ?", token).
		First(&record).
		Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.EmailVerificationToken{}, gorm.ErrRecordNotFound
		}
		logDBQueryFailed(reqLog, "email_verification_tokens", "get_email_verification_token", err)
		return domain.EmailVerificationToken{}, fmt.Errorf("failed to get email verification token: %w", err)
	}

	return domain.EmailVerificationToken{
		ID:                    record.ID,
		UserID:                record.UserID,
		Token:                 record.Token,
		ExpiresAt:             record.ExpiresAt,
		CreatedAt:             record.CreatedAt,
		ResendWindowStartedAt: record.ResendWindowStartedAt,
		ResendCount:           record.ResendCount,
		ConsumedAt:            record.ConsumedAt,
	}, nil
}

// ConsumeTokenAndVerifyUser consumes the token and marks the user as verified in a transaction.
func (r *Repository) ConsumeTokenAndVerifyUser(ctx context.Context, tokenID uint, userID uint, consumedAt time.Time) (domain.User, error) {
	var user domain.User
	if ctx == nil {
		return domain.User{}, logger.ErrNilContext
	}

	reqLog, logErr := r.logger.WithContext(ctx)
	if logErr != nil {
		return domain.User{}, logErr
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Update token as consumed
		err := tx.Model(&emailVerificationTokenRecord{}).
			Where("id = ?", tokenID).
			Update("consumed_at", consumedAt).
			Error
		if err != nil {
			logDBQueryFailed(reqLog, "email_verification_tokens", "consume_token", err, logger.Uint("token_id", tokenID))
			return fmt.Errorf("failed to consume token: %w", err)
		}

		// Update user as verified
		err = tx.Model(&userRecord{}).
			Where("id = ?", userID).
			Updates(map[string]interface{}{
				"email_verified":    true,
				"email_verified_at": consumedAt,
			}).
			Error
		if err != nil {
			logDBQueryFailed(reqLog, "users", "verify_user", err, logger.Uint("user_id", userID))
			return fmt.Errorf("failed to verify user: %w", err)
		}

		// Retrieve updated user
		var record userRecord
		err = tx.
			Select("id, name, email, password, email_verified, email_verified_at, created_at, updated_at").
			Where("id = ?", userID).
			First(&record).
			Error
		if err != nil {
			logDBQueryFailed(reqLog, "users", "get_verified_user", err, logger.Uint("user_id", userID))
			return fmt.Errorf("failed to retrieve user: %w", err)
		}

		name, err := domain.NewUserName(record.Name)
		if err != nil {
			return fmt.Errorf("invalid user name: %w", err)
		}
		emailAddress, err := domain.NewEmailAddress(record.Email)
		if err != nil {
			return fmt.Errorf("invalid email address: %w", err)
		}

		user = domain.User{
			ID:              record.ID,
			Name:            name,
			Email:           emailAddress,
			PasswordHash:    domain.NewPasswordHashFromHash(record.Password),
			EmailVerifiedAt: record.EmailVerifiedAt,
			CreatedAt:       record.CreatedAt,
			UpdatedAt:       record.UpdatedAt,
		}

		return nil
	})

	if err != nil {
		return domain.User{}, err
	}

	return user, nil
}

// GetLatestTokenForUser retrieves the latest token for a user.
func (r *Repository) GetLatestTokenForUser(ctx context.Context, userID uint) (domain.EmailVerificationToken, error) {
	var record emailVerificationTokenRecord

	if ctx == nil {
		return domain.EmailVerificationToken{}, logger.ErrNilContext
	}

	reqLog, logErr := r.logger.WithContext(ctx)
	if logErr != nil {
		return domain.EmailVerificationToken{}, logErr
	}

	err := r.db.
		WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		First(&record).
		Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.EmailVerificationToken{}, gorm.ErrRecordNotFound
		}
		logDBQueryFailed(reqLog, "email_verification_tokens", "get_latest_token_for_user", err, logger.Uint("user_id", userID))
		return domain.EmailVerificationToken{}, fmt.Errorf("failed to get latest token: %w", err)
	}

	return domain.EmailVerificationToken{
		ID:                    record.ID,
		UserID:                record.UserID,
		Token:                 record.Token,
		ExpiresAt:             record.ExpiresAt,
		CreatedAt:             record.CreatedAt,
		ResendWindowStartedAt: record.ResendWindowStartedAt,
		ResendCount:           record.ResendCount,
		ConsumedAt:            record.ConsumedAt,
	}, nil
}

// UpdateVerificationEmailResendWindow updates the fixed-window resend state for a token.
func (r *Repository) UpdateVerificationEmailResendWindow(ctx context.Context, tokenID uint, windowStartedAt time.Time, resendCount int) error {
	if ctx == nil {
		return logger.ErrNilContext
	}

	reqLog, logErr := r.logger.WithContext(ctx)
	if logErr != nil {
		return logErr
	}

	err := r.db.WithContext(ctx).
		Model(&emailVerificationTokenRecord{}).
		Where("id = ?", tokenID).
		Updates(map[string]interface{}{
			"resend_window_started_at": windowStartedAt,
			"resend_count":             resendCount,
		}).
		Error

	if err != nil {
		logDBQueryFailed(reqLog, "email_verification_tokens", "update_verification_email_resend_window", err, logger.Uint("token_id", tokenID))
		return fmt.Errorf("failed to update verification email resend window: %w", err)
	}

	return nil
}

// DeleteTokenByID deletes a token by ID.
func (r *Repository) DeleteTokenByID(ctx context.Context, tokenID uint) error {
	if ctx == nil {
		return logger.ErrNilContext
	}

	reqLog, logErr := r.logger.WithContext(ctx)
	if logErr != nil {
		return logErr
	}

	err := r.db.WithContext(ctx).
		Delete(&emailVerificationTokenRecord{}, tokenID).
		Error

	if err != nil {
		logDBQueryFailed(reqLog, "email_verification_tokens", "delete_token_by_id", err, logger.Uint("token_id", tokenID))
		return fmt.Errorf("failed to delete token: %w", err)
	}

	return nil
}
