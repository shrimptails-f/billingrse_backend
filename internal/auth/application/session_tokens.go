package application

import (
	"business/internal/auth/domain"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// ErrRefreshTokenInvalid is returned when a refresh token cannot be used.
var ErrRefreshTokenInvalid = errors.New("refresh token invalid")

// LoginTokens authenticates a user and returns an access token plus a refresh token.
func (uc *AuthUseCase) LoginTokens(ctx context.Context, req domain.LoginRequest) (domain.AuthTokens, error) {
	user, err := uc.authenticateUser(ctx, req)
	if err != nil {
		return domain.AuthTokens{}, err
	}

	now := uc.clock.Now()
	accessToken, err := uc.issueAccessToken(user.ID, now)
	if err != nil {
		return domain.AuthTokens{}, err
	}

	refreshToken, err := uc.newRefreshToken(user.ID, now)
	if err != nil {
		return domain.AuthTokens{}, err
	}

	createdRefreshToken, err := uc.repo.CreateRefreshToken(ctx, refreshToken)
	if err != nil {
		return domain.AuthTokens{}, fmt.Errorf("failed to create refresh token: %w", err)
	}

	return uc.toAuthTokens(accessToken, createdRefreshToken), nil
}

// Refresh rotates the refresh token and issues a new access token.
func (uc *AuthUseCase) Refresh(ctx context.Context, req domain.RefreshRequest) (domain.AuthTokens, error) {
	rawToken := strings.TrimSpace(req.RefreshToken)
	if rawToken == "" {
		return domain.AuthTokens{}, ErrInvalidInput
	}

	now := uc.clock.Now()
	current, err := uc.repo.FindActiveRefreshTokenByDigest(ctx, digestRefreshToken(rawToken), now)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.AuthTokens{}, ErrRefreshTokenInvalid
		}
		return domain.AuthTokens{}, fmt.Errorf("failed to find refresh token: %w", err)
	}

	nextToken, err := uc.newRefreshToken(current.UserID, now)
	if err != nil {
		return domain.AuthTokens{}, err
	}

	rotatedToken, err := uc.repo.RotateRefreshToken(ctx, current.ID, nextToken, now)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.AuthTokens{}, ErrRefreshTokenInvalid
		}
		return domain.AuthTokens{}, fmt.Errorf("failed to rotate refresh token: %w", err)
	}

	accessToken, err := uc.issueAccessToken(current.UserID, now)
	if err != nil {
		return domain.AuthTokens{}, err
	}

	return uc.toAuthTokens(accessToken, rotatedToken), nil
}

// Logout revokes a refresh token if one is provided.
func (uc *AuthUseCase) Logout(ctx context.Context, req domain.LogoutRequest) error {
	rawToken := strings.TrimSpace(req.RefreshToken)
	if rawToken == "" {
		return nil
	}

	now := uc.clock.Now()
	if err := uc.repo.RevokeRefreshTokenByDigest(ctx, digestRefreshToken(rawToken), now); err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}
	return nil
}

func (uc *AuthUseCase) authenticateUser(ctx context.Context, req domain.LoginRequest) (domain.User, error) {
	email, err := domain.NewEmailAddress(req.Email)
	if err != nil {
		return domain.User{}, ErrInvalidCredentials
	}

	user, err := uc.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.User{}, ErrInvalidCredentials
		}
		return domain.User{}, fmt.Errorf("failed to retrieve user: %w", err)
	}

	if !user.VerifyPassword(req.Password) {
		return domain.User{}, ErrInvalidCredentials
	}

	return user, nil
}

func (uc *AuthUseCase) issueAccessToken(userID uint, now time.Time) (string, error) {
	jwtSecret, err := uc.jwtSecret()
	if err != nil {
		return "", err
	}

	issuer := uc.issuer()
	claims := &domain.AuthClaims{
		UserID: userID,
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

func (uc *AuthUseCase) toAuthTokens(accessToken string, refreshToken domain.RefreshToken) domain.AuthTokens {
	return domain.AuthTokens{
		AccessToken:           accessToken,
		TokenType:             "Bearer",
		ExpiresIn:             int64(uc.tokenTTL / time.Second),
		RefreshToken:          refreshToken.Token,
		RefreshTokenExpiresIn: int64(uc.refreshTokenTTL / time.Second),
	}
}

func (uc *AuthUseCase) newRefreshToken(userID uint, now time.Time) (domain.RefreshToken, error) {
	rawToken, err := generateRefreshToken()
	if err != nil {
		return domain.RefreshToken{}, err
	}

	return domain.RefreshToken{
		UserID:      userID,
		Token:       rawToken,
		TokenDigest: digestRefreshToken(rawToken),
		ExpiresAt:   now.Add(uc.refreshTokenTTL),
		CreatedAt:   now,
	}, nil
}

func (uc *AuthUseCase) jwtSecret() (string, error) {
	if uc.osw == nil {
		return "", fmt.Errorf("missing JWT secret: os wrapper is nil")
	}

	jwtSecret, err := uc.osw.GetEnv("JWT_SECRET_KEY")
	if err != nil {
		return "", fmt.Errorf("missing JWT secret: %w", err)
	}
	return jwtSecret, nil
}

func (uc *AuthUseCase) issuer() string {
	if uc.osw == nil {
		return "business"
	}

	issuer, err := uc.osw.GetEnv("APP_NAME")
	if err != nil {
		return "business"
	}

	issuer = strings.TrimSpace(issuer)
	if issuer == "" {
		return "business"
	}
	return issuer
}

func readDurationSecondsEnv(osw interface{ GetEnv(string) (string, error) }, key string, fallback time.Duration) time.Duration {
	if osw == nil {
		return fallback
	}

	raw, err := osw.GetEnv(key)
	if err != nil {
		return fallback
	}

	seconds, err := time.ParseDuration(strings.TrimSpace(raw) + "s")
	if err != nil || seconds <= 0 {
		return fallback
	}
	return seconds
}

func generateRefreshToken() (string, error) {
	const tokenSize = 32

	raw := make([]byte, tokenSize)
	if _, err := io.ReadFull(rand.Reader, raw); err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func digestRefreshToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}
