package application

import (
	"business/internal/auth/domain"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// Login authenticates a user with email and password, and returns a JWT token.
// Returns ErrInvalidCredentials if the email doesn't exist or password is incorrect.
// Returns an error if JWT_SECRET_KEY is not set or token generation fails.
func (uc *AuthUseCase) Login(ctx context.Context, req domain.LoginRequest) (string, error) {
	jwtSecret, err := uc.osw.GetEnv("JWT_SECRET_KEY")
	if err != nil {
		return "", fmt.Errorf("missing JWT secret: %w", err)
	}

	email, err := domain.NewEmailAddress(req.Email)
	if err != nil {
		return "", ErrInvalidCredentials
	}

	user, err := uc.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrInvalidCredentials
		}
		return "", fmt.Errorf("failed to retrieve user: %w", err)
	}

	if !user.VerifyPassword(req.Password) {
		return "", ErrInvalidCredentials
	}

	issuer, issuerErr := uc.osw.GetEnv("APP_NAME")
	if issuerErr != nil || strings.TrimSpace(issuer) == "" {
		issuer = "business"
	} else {
		issuer = strings.TrimSpace(issuer)
	}

	now := time.Now()
	claims := &domain.AuthClaims{
		UserID: user.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(uc.tokenTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT token: %w", err)
	}

	return signedToken, nil
}
