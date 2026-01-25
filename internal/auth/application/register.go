package application

import (
	"business/internal/auth/domain"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Register creates a new user ensuring no duplicate emails and storing hashed passwords.
// After creating the user, it checks for an existing active token and reuses it if available.
// Otherwise, it generates a new verification token and sends a verification email.
func (uc *AuthUseCase) Register(ctx context.Context, req domain.RegisterRequest) (domain.User, error) {
	email, err := domain.NewEmailAddress(req.Email)
	if err != nil {
		return domain.User{}, ErrInvalidInput
	}

	name, err := domain.NewUserName(req.Name)
	if err != nil {
		return domain.User{}, ErrInvalidInput
	}

	_, err = uc.repo.GetUserByEmail(ctx, email)
	if err == nil {
		return domain.User{}, ErrEmailAlreadyExists
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.User{}, fmt.Errorf("failed to check existing user: %w", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, fmt.Errorf("failed to hash password: %w", err)
	}

	user, err := uc.repo.CreateUser(ctx, domain.User{
		Name:         name,
		Email:        email,
		PasswordHash: string(hashedPassword),
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return domain.User{}, ErrEmailAlreadyExists
		}
		return domain.User{}, fmt.Errorf("failed to create user: %w", err)
	}

	// Check for existing active token
	now := uc.clock()
	existingToken, err := uc.repo.GetActiveTokenForUser(ctx, user.ID, now)
	var tokenToUse domain.EmailVerificationToken

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		// Unexpected error
		_ = uc.repo.DeleteUserByID(ctx, user.ID)
		return domain.User{}, fmt.Errorf("failed to check existing token: %w", err)
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
			// Delete the user if token creation fails
			_ = uc.repo.DeleteUserByID(ctx, user.ID)
			return domain.User{}, fmt.Errorf("failed to create verification token: %w", err)
		}
		tokenToUse = createdToken
	}

	// Send verification email
	baseURL, envErr := uc.osw.GetEnv("FRONT_DOMAIN")
	if envErr != nil || strings.TrimSpace(baseURL) == "" {
		baseURL = "https://local.auth.example.com"
	} else {
		baseURL = strings.TrimSpace(baseURL)
	}
	verifyURL := fmt.Sprintf("%s/signup/verify?token=%s", baseURL, tokenToUse.Token)

	err = uc.mailer.SendVerificationEmail(ctx, user, verifyURL)
	if err != nil {
		// Delete the token and user if email sending fails
		_ = uc.repo.DeleteTokenByID(ctx, tokenToUse.ID)
		_ = uc.repo.DeleteUserByID(ctx, user.ID)
		return domain.User{}, ErrMailSendFailed
	}

	return user, nil
}
