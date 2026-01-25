package application

import (
	"business/internal/auth/domain"
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// VerifyEmail verifies a user's email address using the provided token.
// It checks if the token exists, is not expired, and has not been consumed.
func (uc *AuthUseCase) VerifyEmail(ctx context.Context, req domain.VerifyEmailRequest) (domain.User, error) {
	// Get the token
	token, err := uc.repo.GetEmailVerificationToken(ctx, req.Token)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.User{}, ErrVerificationTokenInvalid
		}
		return domain.User{}, fmt.Errorf("failed to retrieve token: %w", err)
	}

	// Check if already consumed
	if token.ConsumedAt != nil {
		return domain.User{}, ErrVerificationTokenConsumed
	}

	// Check if expired
	now := uc.clock()
	if now.After(token.ExpiresAt) {
		return domain.User{}, ErrVerificationTokenExpired
	}

	// Consume token and verify user
	user, err := uc.repo.ConsumeTokenAndVerifyUser(ctx, token.ID, token.UserID, now)
	if err != nil {
		return domain.User{}, fmt.Errorf("failed to verify user: %w", err)
	}

	return user, nil
}
