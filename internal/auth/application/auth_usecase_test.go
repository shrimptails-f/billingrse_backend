package application

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"business/internal/auth/domain"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type mockAuthRepository struct {
	mock.Mock
}

func (m *mockAuthRepository) GetUserByEmail(ctx context.Context, email domain.EmailAddress) (domain.User, error) {
	args := m.Called(ctx, email)
	user, _ := args.Get(0).(domain.User)
	return user, args.Error(1)
}

func (m *mockAuthRepository) GetUserByID(ctx context.Context, id uint) (domain.User, error) {
	args := m.Called(ctx, id)
	user, _ := args.Get(0).(domain.User)
	return user, args.Error(1)
}

func (m *mockAuthRepository) CreateUser(ctx context.Context, user domain.User) (domain.User, error) {
	args := m.Called(ctx, user)
	createdUser, _ := args.Get(0).(domain.User)
	return createdUser, args.Error(1)
}

func (m *mockAuthRepository) GetActiveTokenForUser(ctx context.Context, userID uint, now time.Time) (domain.EmailVerificationToken, error) {
	args := m.Called(ctx, userID, now)
	token, _ := args.Get(0).(domain.EmailVerificationToken)
	return token, args.Error(1)
}

func (m *mockAuthRepository) CreateEmailVerificationToken(ctx context.Context, token domain.EmailVerificationToken) (domain.EmailVerificationToken, error) {
	args := m.Called(ctx, token)
	createdToken, _ := args.Get(0).(domain.EmailVerificationToken)
	return createdToken, args.Error(1)
}

func (m *mockAuthRepository) InvalidateActiveTokens(ctx context.Context, userID uint, consumedAt time.Time) error {
	args := m.Called(ctx, userID, consumedAt)
	return args.Error(0)
}

func (m *mockAuthRepository) GetEmailVerificationToken(ctx context.Context, token string) (domain.EmailVerificationToken, error) {
	args := m.Called(ctx, token)
	tokenObj, _ := args.Get(0).(domain.EmailVerificationToken)
	return tokenObj, args.Error(1)
}

func (m *mockAuthRepository) ConsumeTokenAndVerifyUser(ctx context.Context, tokenID uint, userID uint, consumedAt time.Time) (domain.User, error) {
	args := m.Called(ctx, tokenID, userID, consumedAt)
	user, _ := args.Get(0).(domain.User)
	return user, args.Error(1)
}

func (m *mockAuthRepository) GetLatestTokenForUser(ctx context.Context, userID uint) (domain.EmailVerificationToken, error) {
	args := m.Called(ctx, userID)
	token, _ := args.Get(0).(domain.EmailVerificationToken)
	return token, args.Error(1)
}

func (m *mockAuthRepository) DeleteTokenByID(ctx context.Context, tokenID uint) error {
	args := m.Called(ctx, tokenID)
	return args.Error(0)
}

func (m *mockAuthRepository) DeleteUserByID(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type mockVerificationEmailSender struct {
	mock.Mock
}

func (m *mockVerificationEmailSender) SendVerificationEmail(ctx context.Context, user domain.User, verifyURL string) error {
	args := m.Called(ctx, user, verifyURL)
	return args.Error(0)
}

type stubOsWrapper struct {
	env map[string]string
}

func (s *stubOsWrapper) ReadFile(path string) (string, error) {
	return "", errors.New("not implemented")
}

func (s *stubOsWrapper) GetEnv(key string) (string, error) {
	if s.env == nil {
		return "", fmt.Errorf("environment variable %s not set", key)
	}
	if value, ok := s.env[key]; ok && value != "" {
		return value, nil
	}
	return "", fmt.Errorf("environment variable %s not set", key)
}

func newStubOsWrapper(secret string) *stubOsWrapper {
	return &stubOsWrapper{
		env: map[string]string{
			"JWT_SECRET_KEY": secret,
		},
	}
}

func TestAuthUseCase_Success(t *testing.T) {
	t.Parallel()
	secret := "test-secret"
	stubOS := newStubOsWrapper(secret)
	repo := new(mockAuthRepository)

	hashedPassword, err := domain.NewPasswordHashFromPlaintext("password123")
	assert.NoError(t, err)

	repo.On("GetUserByEmail", mock.Anything, domain.EmailAddress("user@example.com")).Return(domain.User{
		ID:           1,
		Email:        domain.EmailAddress("user@example.com"),
		PasswordHash: hashedPassword,
	}, nil)

	mailer := new(mockVerificationEmailSender)
	uc := NewAuthUseCase(repo, stubOS, time.Hour, mailer, nil)

	token, err := uc.Login(context.Background(), domain.LoginRequest{
		Email:    "user@example.com",
		Password: "password123",
	})

	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	parsedToken, err := jwt.ParseWithClaims(token, &domain.AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	assert.NoError(t, err)
	assert.True(t, parsedToken.Valid)
	claims, ok := parsedToken.Claims.(*domain.AuthClaims)
	assert.True(t, ok)
	assert.Equal(t, uint(1), claims.UserID)

	repo.AssertExpectations(t)
}

func TestAuthUseCase_UserNotFound(t *testing.T) {
	t.Parallel()
	repo := new(mockAuthRepository)
	repo.On("GetUserByEmail", mock.Anything, domain.EmailAddress("missing@example.com")).Return(domain.User{}, gorm.ErrRecordNotFound)

	mailer := new(mockVerificationEmailSender)
	uc := NewAuthUseCase(repo, newStubOsWrapper("secret"), time.Hour, mailer, nil)
	token, err := uc.Login(context.Background(), domain.LoginRequest{
		Email:    "missing@example.com",
		Password: "password123",
	})

	assert.ErrorIs(t, err, ErrInvalidCredentials)
	assert.Empty(t, token)
}

func TestAuthUseCase_InvalidPassword(t *testing.T) {
	t.Parallel()
	repo := new(mockAuthRepository)
	hashedPassword, err := domain.NewPasswordHashFromPlaintext("password123")
	assert.NoError(t, err)

	repo.On("GetUserByEmail", mock.Anything, domain.EmailAddress("user@example.com")).Return(domain.User{
		ID:           1,
		Email:        domain.EmailAddress("user@example.com"),
		PasswordHash: hashedPassword,
	}, nil)

	mailer := new(mockVerificationEmailSender)
	uc := NewAuthUseCase(repo, newStubOsWrapper("secret"), time.Hour, mailer, nil)
	token, err := uc.Login(context.Background(), domain.LoginRequest{
		Email:    "user@example.com",
		Password: "wrong-password",
	})

	assert.ErrorIs(t, err, ErrInvalidCredentials)
	assert.Empty(t, token)
}

func TestAuthUseCase_RepositoryError(t *testing.T) {
	t.Parallel()
	repo := new(mockAuthRepository)
	repo.On("GetUserByEmail", mock.Anything, domain.EmailAddress("user@example.com")).Return(domain.User{}, errors.New("db error"))

	mailer := new(mockVerificationEmailSender)
	uc := NewAuthUseCase(repo, newStubOsWrapper("secret"), time.Hour, mailer, nil)
	token, err := uc.Login(context.Background(), domain.LoginRequest{
		Email:    "user@example.com",
		Password: "password123",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
	assert.Empty(t, token)
}

func TestAuthUseCase_RegisterSuccess(t *testing.T) {
	t.Parallel()
	fixedTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := new(mockAuthRepository)
	mailer := new(mockVerificationEmailSender)

	repo.On("GetUserByEmail", mock.Anything, domain.EmailAddress("new@example.com")).Return(domain.User{}, gorm.ErrRecordNotFound).Once()
	repo.On("CreateUser", mock.Anything, mock.MatchedBy(func(user domain.User) bool {
		return user.Email.String() == "new@example.com" && user.Name.String() == "New User" && user.PasswordHash.String() != ""
	})).Return(domain.User{
		ID:        1,
		Email:     domain.EmailAddress("new@example.com"),
		Name:      domain.UserName("New User"),
		CreatedAt: fixedTime,
	}, nil).Once()
	repo.On("GetActiveTokenForUser", mock.Anything, uint(1), fixedTime).Return(domain.EmailVerificationToken{}, gorm.ErrRecordNotFound).Once()
	repo.On("CreateEmailVerificationToken", mock.Anything, mock.MatchedBy(func(token domain.EmailVerificationToken) bool {
		return token.UserID == 1 && token.Token != ""
	})).Return(domain.EmailVerificationToken{ID: 1, UserID: 1, Token: "test-token"}, nil).Once()
	mailer.On("SendVerificationEmail", mock.Anything, mock.Anything, mock.MatchedBy(func(url string) bool {
		return url == "https://example.com/signup/verify?token=test-token"
	})).Return(nil).Once()

	stubOS := &stubOsWrapper{env: map[string]string{"FRONT_DOMAIN": "https://example.com"}}
	uc := NewAuthUseCase(repo, stubOS, time.Hour, mailer, func() time.Time { return fixedTime })

	user, err := uc.Register(context.Background(), domain.RegisterRequest{
		Email:    "new@example.com",
		Name:     "New User",
		Password: "password123",
	})

	assert.NoError(t, err)
	assert.Equal(t, uint(1), user.ID)
	assert.Equal(t, "new@example.com", user.Email.String())
	assert.Equal(t, "New User", user.Name.String())
	assert.False(t, user.IsEmailVerified())
	repo.AssertExpectations(t)
	mailer.AssertExpectations(t)
}

func TestAuthUseCase_RegisterEmailExists(t *testing.T) {
	t.Parallel()
	repo := new(mockAuthRepository)
	mailer := new(mockVerificationEmailSender)
	repo.On("GetUserByEmail", mock.Anything, domain.EmailAddress("dup@example.com")).Return(domain.User{ID: 1}, nil).Once()

	uc := NewAuthUseCase(repo, newStubOsWrapper("secret"), time.Hour, mailer, nil)
	user, err := uc.Register(context.Background(), domain.RegisterRequest{
		Email:    "dup@example.com",
		Name:     "Dup",
		Password: "password123",
	})

	assert.ErrorIs(t, err, ErrEmailAlreadyExists)
	assert.Equal(t, uint(0), user.ID)
	repo.AssertExpectations(t)
}

func TestAuthUseCase_RegisterLookupError(t *testing.T) {
	t.Parallel()
	repo := new(mockAuthRepository)
	mailer := new(mockVerificationEmailSender)
	repo.On("GetUserByEmail", mock.Anything, domain.EmailAddress("user@example.com")).Return(domain.User{}, errors.New("db failure")).Once()

	uc := NewAuthUseCase(repo, newStubOsWrapper("secret"), time.Hour, mailer, nil)
	user, err := uc.Register(context.Background(), domain.RegisterRequest{
		Email:    "user@example.com",
		Name:     "User",
		Password: "password123",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db failure")
	assert.Equal(t, uint(0), user.ID)
	repo.AssertExpectations(t)
}

func TestAuthUseCase_RegisterDuplicatedKey(t *testing.T) {
	t.Parallel()
	repo := new(mockAuthRepository)
	mailer := new(mockVerificationEmailSender)
	repo.On("GetUserByEmail", mock.Anything, domain.EmailAddress("user@example.com")).Return(domain.User{}, gorm.ErrRecordNotFound).Once()
	repo.On("CreateUser", mock.Anything, mock.Anything).Return(domain.User{}, gorm.ErrDuplicatedKey).Once()

	uc := NewAuthUseCase(repo, newStubOsWrapper("secret"), time.Hour, mailer, nil)
	user, err := uc.Register(context.Background(), domain.RegisterRequest{
		Email:    "user@example.com",
		Name:     "User",
		Password: "password123",
	})

	assert.ErrorIs(t, err, ErrEmailAlreadyExists)
	assert.Equal(t, uint(0), user.ID)
	repo.AssertExpectations(t)
}

func TestAuthUseCase_RegisterCreateError(t *testing.T) {
	t.Parallel()
	repo := new(mockAuthRepository)
	mailer := new(mockVerificationEmailSender)
	repo.On("GetUserByEmail", mock.Anything, domain.EmailAddress("user@example.com")).Return(domain.User{}, gorm.ErrRecordNotFound).Once()
	repo.On("CreateUser", mock.Anything, mock.Anything).Return(domain.User{}, errors.New("insert failed")).Once()

	uc := NewAuthUseCase(repo, newStubOsWrapper("secret"), time.Hour, mailer, nil)
	user, err := uc.Register(context.Background(), domain.RegisterRequest{
		Email:    "user@example.com",
		Name:     "User",
		Password: "password123",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insert failed")
	assert.Equal(t, uint(0), user.ID)
	repo.AssertExpectations(t)
}

func TestAuthUseCase_RegisterTokenCreationFailed(t *testing.T) {
	t.Parallel()
	fixedTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := new(mockAuthRepository)
	mailer := new(mockVerificationEmailSender)

	repo.On("GetUserByEmail", mock.Anything, domain.EmailAddress("new@example.com")).Return(domain.User{}, gorm.ErrRecordNotFound).Once()
	repo.On("CreateUser", mock.Anything, mock.Anything).Return(domain.User{
		ID:    1,
		Email: domain.EmailAddress("new@example.com"),
		Name:  domain.UserName("New User"),
	}, nil).Once()
	repo.On("GetActiveTokenForUser", mock.Anything, uint(1), fixedTime).Return(domain.EmailVerificationToken{}, gorm.ErrRecordNotFound).Once()
	repo.On("CreateEmailVerificationToken", mock.Anything, mock.Anything).Return(domain.EmailVerificationToken{}, errors.New("token creation failed")).Once()
	repo.On("DeleteUserByID", mock.Anything, uint(1)).Return(nil).Once()

	stubOS := &stubOsWrapper{env: map[string]string{"FRONT_DOMAIN": "https://example.com"}}
	uc := NewAuthUseCase(repo, stubOS, time.Hour, mailer, func() time.Time { return fixedTime })

	user, err := uc.Register(context.Background(), domain.RegisterRequest{
		Email:    "new@example.com",
		Name:     "New User",
		Password: "password123",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create verification token")
	assert.Equal(t, uint(0), user.ID)
	repo.AssertExpectations(t)
}

func TestAuthUseCase_RegisterMailSendFailed(t *testing.T) {
	t.Parallel()
	fixedTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := new(mockAuthRepository)
	mailer := new(mockVerificationEmailSender)

	repo.On("GetUserByEmail", mock.Anything, domain.EmailAddress("new@example.com")).Return(domain.User{}, gorm.ErrRecordNotFound).Once()
	repo.On("CreateUser", mock.Anything, mock.Anything).Return(domain.User{
		ID:    1,
		Email: domain.EmailAddress("new@example.com"),
		Name:  domain.UserName("New User"),
	}, nil).Once()
	repo.On("GetActiveTokenForUser", mock.Anything, uint(1), fixedTime).Return(domain.EmailVerificationToken{}, gorm.ErrRecordNotFound).Once()
	repo.On("CreateEmailVerificationToken", mock.Anything, mock.Anything).Return(domain.EmailVerificationToken{ID: 1, UserID: 1, Token: "test-token"}, nil).Once()
	mailer.On("SendVerificationEmail", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("smtp error")).Once()
	repo.On("DeleteTokenByID", mock.Anything, uint(1)).Return(nil).Once()
	repo.On("DeleteUserByID", mock.Anything, uint(1)).Return(nil).Once()

	stubOS := &stubOsWrapper{env: map[string]string{"FRONT_DOMAIN": "https://example.com"}}
	uc := NewAuthUseCase(repo, stubOS, time.Hour, mailer, func() time.Time { return fixedTime })

	user, err := uc.Register(context.Background(), domain.RegisterRequest{
		Email:    "new@example.com",
		Name:     "New User",
		Password: "password123",
	})

	assert.ErrorIs(t, err, ErrMailSendFailed)
	assert.Equal(t, uint(0), user.ID)
	repo.AssertExpectations(t)
	mailer.AssertExpectations(t)
}

func TestAuthUseCase_VerifyEmailSuccess(t *testing.T) {
	t.Parallel()
	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := new(mockAuthRepository)
	mailer := new(mockVerificationEmailSender)

	token := domain.EmailVerificationToken{
		ID:        1,
		UserID:    1,
		Token:     "valid-token",
		ExpiresAt: fixedTime.Add(1 * time.Hour),
		CreatedAt: fixedTime.Add(-1 * time.Hour),
	}

	repo.On("GetEmailVerificationToken", mock.Anything, "valid-token").Return(token, nil).Once()
	repo.On("ConsumeTokenAndVerifyUser", mock.Anything, uint(1), uint(1), fixedTime).Return(domain.User{
		ID:              1,
		Email:           domain.EmailAddress("user@example.com"),
		EmailVerifiedAt: &fixedTime,
	}, nil).Once()

	uc := NewAuthUseCase(repo, newStubOsWrapper("secret"), time.Hour, mailer, func() time.Time { return fixedTime })

	user, err := uc.VerifyEmail(context.Background(), domain.VerifyEmailRequest{Token: "valid-token"})

	assert.NoError(t, err)
	assert.Equal(t, uint(1), user.ID)
	assert.True(t, user.IsEmailVerified())
	repo.AssertExpectations(t)
}

func TestAuthUseCase_VerifyEmailInvalidToken(t *testing.T) {
	t.Parallel()
	repo := new(mockAuthRepository)
	mailer := new(mockVerificationEmailSender)

	repo.On("GetEmailVerificationToken", mock.Anything, "invalid-token").Return(domain.EmailVerificationToken{}, gorm.ErrRecordNotFound).Once()

	uc := NewAuthUseCase(repo, newStubOsWrapper("secret"), time.Hour, mailer, nil)

	user, err := uc.VerifyEmail(context.Background(), domain.VerifyEmailRequest{Token: "invalid-token"})

	assert.ErrorIs(t, err, ErrVerificationTokenInvalid)
	assert.Equal(t, uint(0), user.ID)
	repo.AssertExpectations(t)
}

func TestAuthUseCase_VerifyEmailExpired(t *testing.T) {
	t.Parallel()
	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := new(mockAuthRepository)
	mailer := new(mockVerificationEmailSender)

	token := domain.EmailVerificationToken{
		ID:        1,
		UserID:    1,
		Token:     "expired-token",
		ExpiresAt: fixedTime.Add(-1 * time.Hour),
		CreatedAt: fixedTime.Add(-4 * time.Hour),
	}

	repo.On("GetEmailVerificationToken", mock.Anything, "expired-token").Return(token, nil).Once()

	uc := NewAuthUseCase(repo, newStubOsWrapper("secret"), time.Hour, mailer, func() time.Time { return fixedTime })

	user, err := uc.VerifyEmail(context.Background(), domain.VerifyEmailRequest{Token: "expired-token"})

	assert.ErrorIs(t, err, ErrVerificationTokenExpired)
	assert.Equal(t, uint(0), user.ID)
	repo.AssertExpectations(t)
}

func TestAuthUseCase_VerifyEmailConsumed(t *testing.T) {
	t.Parallel()
	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	consumedTime := fixedTime.Add(-30 * time.Minute)
	repo := new(mockAuthRepository)
	mailer := new(mockVerificationEmailSender)

	token := domain.EmailVerificationToken{
		ID:         1,
		UserID:     1,
		Token:      "consumed-token",
		ExpiresAt:  fixedTime.Add(1 * time.Hour),
		CreatedAt:  fixedTime.Add(-1 * time.Hour),
		ConsumedAt: &consumedTime,
	}

	repo.On("GetEmailVerificationToken", mock.Anything, "consumed-token").Return(token, nil).Once()

	uc := NewAuthUseCase(repo, newStubOsWrapper("secret"), time.Hour, mailer, func() time.Time { return fixedTime })

	user, err := uc.VerifyEmail(context.Background(), domain.VerifyEmailRequest{Token: "consumed-token"})

	assert.ErrorIs(t, err, ErrVerificationTokenConsumed)
	assert.Equal(t, uint(0), user.ID)
	repo.AssertExpectations(t)
}

func TestAuthUseCase_ResendVerificationEmailSuccess(t *testing.T) {
	t.Parallel()
	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := new(mockAuthRepository)
	mailer := new(mockVerificationEmailSender)

	hashedPassword, err := domain.NewPasswordHashFromPlaintext("password123")
	assert.NoError(t, err)

	user := domain.User{
		ID:           1,
		Email:        domain.EmailAddress("user@example.com"),
		PasswordHash: hashedPassword,
	}

	oldToken := domain.EmailVerificationToken{
		ID:        1,
		UserID:    1,
		Token:     "old-token",
		ExpiresAt: fixedTime.Add(1 * time.Hour),
		CreatedAt: fixedTime.Add(-20 * time.Minute),
	}

	repo.On("GetUserByEmail", mock.Anything, domain.EmailAddress("user@example.com")).Return(user, nil).Once()
	repo.On("GetLatestTokenForUser", mock.Anything, uint(1)).Return(oldToken, nil).Once()
	repo.On("GetActiveTokenForUser", mock.Anything, uint(1), fixedTime).Return(domain.EmailVerificationToken{}, gorm.ErrRecordNotFound).Once()
	repo.On("CreateEmailVerificationToken", mock.Anything, mock.Anything).Return(domain.EmailVerificationToken{ID: 2, UserID: 1, Token: "new-token"}, nil).Once()
	mailer.On("SendVerificationEmail", mock.Anything, mock.Anything, mock.MatchedBy(func(url string) bool {
		return url == "https://example.com/auth/email/verify?token=new-token"
	})).Return(nil).Once()

	stubOS := &stubOsWrapper{env: map[string]string{"EMAIL_VERIFICATION_BASE_URL": "https://example.com"}}
	uc := NewAuthUseCase(repo, stubOS, time.Hour, mailer, func() time.Time { return fixedTime })

	err = uc.ResendVerificationEmail(context.Background(), domain.ResendVerificationRequest{
		Email:    "user@example.com",
		Password: "password123",
	})

	assert.NoError(t, err)
	repo.AssertExpectations(t)
	mailer.AssertExpectations(t)
}

func TestAuthUseCase_ResendVerificationEmailInvalidCredentials(t *testing.T) {
	t.Parallel()
	repo := new(mockAuthRepository)
	mailer := new(mockVerificationEmailSender)

	repo.On("GetUserByEmail", mock.Anything, domain.EmailAddress("missing@example.com")).Return(domain.User{}, gorm.ErrRecordNotFound).Once()

	uc := NewAuthUseCase(repo, newStubOsWrapper("secret"), time.Hour, mailer, nil)

	err := uc.ResendVerificationEmail(context.Background(), domain.ResendVerificationRequest{
		Email:    "missing@example.com",
		Password: "password123",
	})

	assert.ErrorIs(t, err, ErrInvalidCredentials)
	repo.AssertExpectations(t)
}

func TestAuthUseCase_ResendVerificationEmailWrongPassword(t *testing.T) {
	t.Parallel()
	repo := new(mockAuthRepository)
	mailer := new(mockVerificationEmailSender)

	hashedPassword, err := domain.NewPasswordHashFromPlaintext("password123")
	assert.NoError(t, err)

	user := domain.User{
		ID:           1,
		Email:        domain.EmailAddress("user@example.com"),
		PasswordHash: hashedPassword,
	}

	repo.On("GetUserByEmail", mock.Anything, domain.EmailAddress("user@example.com")).Return(user, nil).Once()

	uc := NewAuthUseCase(repo, newStubOsWrapper("secret"), time.Hour, mailer, nil)

	err = uc.ResendVerificationEmail(context.Background(), domain.ResendVerificationRequest{
		Email:    "user@example.com",
		Password: "wrong-password",
	})

	assert.ErrorIs(t, err, ErrInvalidCredentials)
	repo.AssertExpectations(t)
}

func TestAuthUseCase_ResendVerificationEmailAlreadyVerified(t *testing.T) {
	t.Parallel()
	repo := new(mockAuthRepository)
	mailer := new(mockVerificationEmailSender)

	hashedPassword, err := domain.NewPasswordHashFromPlaintext("password123")
	assert.NoError(t, err)

	verifiedAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	user := domain.User{
		ID:              1,
		Email:           domain.EmailAddress("user@example.com"),
		PasswordHash:    hashedPassword,
		EmailVerifiedAt: &verifiedAt,
	}

	repo.On("GetUserByEmail", mock.Anything, domain.EmailAddress("user@example.com")).Return(user, nil).Once()

	uc := NewAuthUseCase(repo, newStubOsWrapper("secret"), time.Hour, mailer, nil)

	err = uc.ResendVerificationEmail(context.Background(), domain.ResendVerificationRequest{
		Email:    "user@example.com",
		Password: "password123",
	})

	assert.ErrorIs(t, err, ErrEmailAlreadyVerified)
	repo.AssertExpectations(t)
}

func TestAuthUseCase_ResendVerificationEmailRateLimited(t *testing.T) {
	t.Parallel()
	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := new(mockAuthRepository)
	mailer := new(mockVerificationEmailSender)

	hashedPassword, err := domain.NewPasswordHashFromPlaintext("password123")
	assert.NoError(t, err)

	user := domain.User{
		ID:           1,
		Email:        domain.EmailAddress("user@example.com"),
		PasswordHash: hashedPassword,
	}

	recentToken := domain.EmailVerificationToken{
		ID:        1,
		UserID:    1,
		Token:     "recent-token",
		ExpiresAt: fixedTime.Add(1 * time.Hour),
		CreatedAt: fixedTime.Add(-10 * time.Minute), // Less than 15 minutes ago
	}

	repo.On("GetUserByEmail", mock.Anything, domain.EmailAddress("user@example.com")).Return(user, nil).Once()
	repo.On("GetLatestTokenForUser", mock.Anything, uint(1)).Return(recentToken, nil).Once()

	uc := NewAuthUseCase(repo, newStubOsWrapper("secret"), time.Hour, mailer, func() time.Time { return fixedTime })

	err = uc.ResendVerificationEmail(context.Background(), domain.ResendVerificationRequest{
		Email:    "user@example.com",
		Password: "password123",
	})

	assert.ErrorIs(t, err, ErrResendRateLimited)
	repo.AssertExpectations(t)
}

func TestAuthUseCase_ResendVerificationEmailMailSendFailed(t *testing.T) {
	t.Parallel()
	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := new(mockAuthRepository)
	mailer := new(mockVerificationEmailSender)

	hashedPassword, err := domain.NewPasswordHashFromPlaintext("password123")
	assert.NoError(t, err)

	user := domain.User{
		ID:           1,
		Email:        domain.EmailAddress("user@example.com"),
		PasswordHash: hashedPassword,
	}

	oldToken := domain.EmailVerificationToken{
		ID:        1,
		UserID:    1,
		Token:     "old-token",
		ExpiresAt: fixedTime.Add(1 * time.Hour),
		CreatedAt: fixedTime.Add(-20 * time.Minute),
	}

	repo.On("GetUserByEmail", mock.Anything, domain.EmailAddress("user@example.com")).Return(user, nil).Once()
	repo.On("GetLatestTokenForUser", mock.Anything, uint(1)).Return(oldToken, nil).Once()
	repo.On("GetActiveTokenForUser", mock.Anything, uint(1), fixedTime).Return(domain.EmailVerificationToken{}, gorm.ErrRecordNotFound).Once()
	repo.On("CreateEmailVerificationToken", mock.Anything, mock.Anything).Return(domain.EmailVerificationToken{ID: 2, UserID: 1, Token: "new-token"}, nil).Once()
	mailer.On("SendVerificationEmail", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("smtp error")).Once()
	repo.On("DeleteTokenByID", mock.Anything, uint(2)).Return(nil).Once()

	stubOS := &stubOsWrapper{env: map[string]string{"EMAIL_VERIFICATION_BASE_URL": "https://example.com"}}
	uc := NewAuthUseCase(repo, stubOS, time.Hour, mailer, func() time.Time { return fixedTime })

	err = uc.ResendVerificationEmail(context.Background(), domain.ResendVerificationRequest{
		Email:    "user@example.com",
		Password: "password123",
	})

	assert.ErrorIs(t, err, ErrMailSendFailed)
	repo.AssertExpectations(t)
	mailer.AssertExpectations(t)
}
