package application

import (
	"business/internal/auth/domain"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ResendVerificationEmail resends the verification email after authenticating the user
// and checking rate limits (15 minutes since last token).
// It checks for an existing active token and reuses it if available.
func (uc *AuthUseCase) ResendVerificationEmail(ctx context.Context, req domain.ResendVerificationRequest) error {
	// Authenticate the user
	email, err := domain.NewEmailAddress(req.Email)
	if err != nil {
		return ErrInvalidCredentials
	}

	user, err := uc.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInvalidCredentials
		}
		return fmt.Errorf("failed to retrieve user: %w", err)
	}

	if !user.VerifyPassword(req.Password) {
		return ErrInvalidCredentials
	}

	// Check if already verified
	if user.EmailVerified {
		return ErrEmailAlreadyVerified
	}

	// Check rate limiting
	now := uc.clock()
	latestToken, err := uc.repo.GetLatestTokenForUser(ctx, user.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to get latest token: %w", err)
	}

	if err == nil {
		// Token exists, check rate limit
		if now.Sub(latestToken.CreatedAt) < 15*time.Minute {
			return ErrResendRateLimited
		}
	}

	// Check for existing active token
	existingToken, err := uc.repo.GetActiveTokenForUser(ctx, user.ID, now)
	var tokenToUse domain.EmailVerificationToken

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to check existing token: %w", err)
	}

	if err == nil {
		// Reuse existing active token
		tokenToUse = existingToken
	} else {
		// Generate new verification token (upsert)
		tokenString := uuid.New().String()
		token := domain.EmailVerificationToken{
			UserID:    user.ID,
			Token:     tokenString,
			ExpiresAt: now.Add(3 * time.Hour),
			CreatedAt: now,
		}

		createdToken, err := uc.repo.CreateEmailVerificationToken(ctx, token)
		if err != nil {
			return fmt.Errorf("failed to create verification token: %w", err)
		}
		tokenToUse = createdToken
	}

	// Send verification email
	baseURL, envErr := uc.osw.GetEnv("EMAIL_VERIFICATION_BASE_URL")
	if envErr != nil || strings.TrimSpace(baseURL) == "" {
		baseURL = "https://local.auth.example.com"
	} else {
		baseURL = strings.TrimSpace(baseURL)
	}
	verifyURL := fmt.Sprintf("%s/auth/email/verify?token=%s", baseURL, tokenToUse.Token)

	err = uc.mailer.SendVerificationEmail(ctx, user, verifyURL)
	if err != nil {
		// Delete the token if email sending fails
		_ = uc.repo.DeleteTokenByID(ctx, tokenToUse.ID)
		return ErrMailSendFailed
	}

	return nil
}
