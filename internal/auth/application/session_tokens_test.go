package application

import (
	"context"
	"testing"
	"time"

	"business/internal/auth/domain"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

func TestAuthUseCase_LoginTokensSuccess(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	fixedTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := new(mockAuthRepository)
	mailer := new(mockVerificationEmailSender)
	hashed, err := testVault().GenerateHashPassword("password123")
	assert.NoError(t, err)
	hashedPassword := domain.NewPasswordHashFromHash(hashed)

	repo.On("GetUserByEmail", mock.Anything, domain.EmailAddress("user@example.com")).Return(domain.User{
		ID:           1,
		Email:        domain.EmailAddress("user@example.com"),
		PasswordHash: hashedPassword,
	}, nil).Once()
	repo.On("CreateRefreshToken", mock.Anything, mock.MatchedBy(func(token domain.RefreshToken) bool {
		return token.UserID == 1 &&
			token.Token != "" &&
			len(token.TokenDigest) == 64 &&
			token.ExpiresAt.Equal(fixedTime.Add(defaultRefreshTokenTTL)) &&
			token.CreatedAt.Equal(fixedTime)
	})).Return(domain.RefreshToken{ID: 2, UserID: 1, Token: "refresh-token", TokenDigest: "digest"}, nil).Once()

	uc := NewAuthUseCase(repo, newStubOsWrapper(secret), mailer, &stubClock{now: fixedTime}, testVault())

	tokens, err := uc.LoginTokens(context.Background(), domain.LoginRequest{
		Email:    "user@example.com",
		Password: "password123",
	})

	assert.NoError(t, err)
	assert.Equal(t, "Bearer", tokens.TokenType)
	assert.Equal(t, int64(defaultAccessTokenTTL/time.Second), tokens.ExpiresIn)
	assert.Equal(t, int64(defaultRefreshTokenTTL/time.Second), tokens.RefreshTokenExpiresIn)
	assert.Equal(t, "refresh-token", tokens.RefreshToken)
	assert.NotEmpty(t, tokens.AccessToken)

	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	parsedToken, err := parser.ParseWithClaims(tokens.AccessToken, &domain.AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	assert.NoError(t, err)
	assert.True(t, parsedToken.Valid)
	claims, ok := parsedToken.Claims.(*domain.AuthClaims)
	assert.True(t, ok)
	assert.Equal(t, uint(1), claims.UserID)
	assert.True(t, claims.IssuedAt.Time.Equal(fixedTime))
	assert.True(t, claims.ExpiresAt.Time.Equal(fixedTime.Add(defaultAccessTokenTTL)))
	repo.AssertExpectations(t)
}

func TestAuthUseCase_RefreshSuccess(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	fixedTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := new(mockAuthRepository)
	mailer := new(mockVerificationEmailSender)
	refreshToken := "refresh-token"
	refreshDigest := digestRefreshToken(refreshToken)

	repo.On("FindActiveRefreshTokenByDigest", mock.Anything, refreshDigest, fixedTime).Return(domain.RefreshToken{
		ID:          10,
		UserID:      1,
		TokenDigest: refreshDigest,
		ExpiresAt:   fixedTime.Add(defaultRefreshTokenTTL),
		CreatedAt:   fixedTime,
	}, nil).Once()
	repo.On("RotateRefreshToken", mock.Anything, uint(10), mock.MatchedBy(func(token domain.RefreshToken) bool {
		return token.UserID == 1 &&
			token.Token != "" &&
			len(token.TokenDigest) == 64 &&
			token.ExpiresAt.Equal(fixedTime.Add(defaultRefreshTokenTTL)) &&
			token.CreatedAt.Equal(fixedTime)
	}), fixedTime).Return(domain.RefreshToken{
		ID:          11,
		UserID:      1,
		Token:       "rotated-refresh-token",
		TokenDigest: "rotated-digest",
		ExpiresAt:   fixedTime.Add(defaultRefreshTokenTTL),
		CreatedAt:   fixedTime,
	}, nil).Once()

	uc := NewAuthUseCase(repo, newStubOsWrapper(secret), mailer, &stubClock{now: fixedTime}, testVault())

	tokens, err := uc.Refresh(context.Background(), domain.RefreshRequest{RefreshToken: refreshToken})

	assert.NoError(t, err)
	assert.Equal(t, "Bearer", tokens.TokenType)
	assert.Equal(t, "rotated-refresh-token", tokens.RefreshToken)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.Equal(t, int64(defaultAccessTokenTTL/time.Second), tokens.ExpiresIn)
	assert.Equal(t, int64(defaultRefreshTokenTTL/time.Second), tokens.RefreshTokenExpiresIn)

	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	parsedToken, err := parser.ParseWithClaims(tokens.AccessToken, &domain.AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	assert.NoError(t, err)
	assert.True(t, parsedToken.Valid)
	claims, ok := parsedToken.Claims.(*domain.AuthClaims)
	assert.True(t, ok)
	assert.Equal(t, uint(1), claims.UserID)
	assert.True(t, claims.IssuedAt.Time.Equal(fixedTime))
	assert.True(t, claims.ExpiresAt.Time.Equal(fixedTime.Add(defaultAccessTokenTTL)))
	repo.AssertExpectations(t)
}

func TestAuthUseCase_RefreshInvalidToken(t *testing.T) {
	t.Parallel()

	repo := new(mockAuthRepository)
	mailer := new(mockVerificationEmailSender)
	repo.On("FindActiveRefreshTokenByDigest", mock.Anything, digestRefreshToken("refresh-token"), mock.Anything).
		Return(domain.RefreshToken{}, gorm.ErrRecordNotFound).
		Once()

	uc := NewAuthUseCase(repo, newStubOsWrapper("test-secret"), mailer, &stubClock{now: time.Now().UTC()}, testVault())
	tokens, err := uc.Refresh(context.Background(), domain.RefreshRequest{RefreshToken: "refresh-token"})

	assert.ErrorIs(t, err, ErrRefreshTokenInvalid)
	assert.Empty(t, tokens.AccessToken)
}

func TestAuthUseCase_LogoutSuccess(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := new(mockAuthRepository)
	mailer := new(mockVerificationEmailSender)
	repo.On("RevokeRefreshTokenByDigest", mock.Anything, digestRefreshToken("refresh-token"), fixedTime).Return(nil).Once()

	uc := NewAuthUseCase(repo, newStubOsWrapper("test-secret"), mailer, &stubClock{now: fixedTime}, testVault())
	err := uc.Logout(context.Background(), domain.LogoutRequest{RefreshToken: "refresh-token"})

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestAuthUseCase_LogoutNoTokenIsNoop(t *testing.T) {
	t.Parallel()

	repo := new(mockAuthRepository)
	mailer := new(mockVerificationEmailSender)
	uc := NewAuthUseCase(repo, newStubOsWrapper("test-secret"), mailer, nil, testVault())

	err := uc.Logout(context.Background(), domain.LogoutRequest{})

	assert.NoError(t, err)
	repo.AssertNotCalled(t, "RevokeRefreshTokenByDigest", mock.Anything, mock.Anything, mock.Anything)
}
