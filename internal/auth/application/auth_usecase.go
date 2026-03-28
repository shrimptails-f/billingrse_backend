package application

import (
	"business/internal/auth/domain"
	"business/internal/library/oswrapper"
	"business/internal/library/timewrapper"
	"context"
	"errors"
	"time"
)

// ErrInvalidCredentials is returned when email or password is incorrect
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrInvalidInput is returned when request payload values are invalid.
var ErrInvalidInput = errors.New("invalid input")

// ErrEmailAlreadyExists is returned when attempting to register with an existing email.
var ErrEmailAlreadyExists = errors.New("email already exists")

// ErrFailedToCheckExists is failed login
var ErrFailedToCheckExists = errors.New("failed to check existing user")

// ErrMailSendFailed is returned when email sending fails
var ErrMailSendFailed = errors.New("mail send failed")

// ErrVerificationTokenInvalid is returned when the verification token is invalid
var ErrVerificationTokenInvalid = errors.New("verification token invalid")

// ErrVerificationTokenExpired is returned when the verification token has expired
var ErrVerificationTokenExpired = errors.New("verification token expired")

// ErrVerificationTokenConsumed is returned when the verification token has already been consumed
var ErrVerificationTokenConsumed = errors.New("verification token already consumed")

// ErrEmailAlreadyVerified is returned when the email is already verified
var ErrEmailAlreadyVerified = errors.New("email already verified")

// ErrResendRateLimited is returned when resend is rate limited
var ErrResendRateLimited = errors.New("resend rate limited")

// Aliases for backward compatibility with controller naming
var (
	ErrInvalidToken      = ErrVerificationTokenInvalid
	ErrTokenExpired      = ErrVerificationTokenExpired
	ErrTokenAlreadyUsed  = ErrVerificationTokenConsumed
	ErrAlreadyVerified   = ErrEmailAlreadyVerified
	ErrRateLimitExceeded = ErrResendRateLimited
)

// AuthRepository defines the interface for user data access
type AuthRepository interface {
	// GetUserByEmail retrieves a user by email address
	GetUserByEmail(ctx context.Context, email domain.EmailAddress) (domain.User, error)
	// GetUserByID retrieves a user by ID
	GetUserByID(ctx context.Context, id uint) (domain.User, error)
	// CreateUser inserts a new user record.
	CreateUser(ctx context.Context, user domain.User) (domain.User, error)
	// DeleteUserByID deletes a user by ID
	DeleteUserByID(ctx context.Context, id uint) error
	// GetActiveTokenForUser retrieves an active (unconsumed and not expired) token for a user
	GetActiveTokenForUser(ctx context.Context, userID uint, now time.Time) (domain.EmailVerificationToken, error)
	// CreateEmailVerificationToken creates or updates an email verification token (upsert)
	CreateEmailVerificationToken(ctx context.Context, token domain.EmailVerificationToken) (domain.EmailVerificationToken, error)
	// InvalidateActiveTokens invalidates all active tokens for a user
	InvalidateActiveTokens(ctx context.Context, userID uint, consumedAt time.Time) error
	// GetEmailVerificationToken retrieves a verification token by token string
	GetEmailVerificationToken(ctx context.Context, token string) (domain.EmailVerificationToken, error)
	// ConsumeTokenAndVerifyUser consumes the token and marks the user as verified
	ConsumeTokenAndVerifyUser(ctx context.Context, tokenID uint, userID uint, consumedAt time.Time) (domain.User, error)
	// GetLatestTokenForUser retrieves the latest token for a user
	GetLatestTokenForUser(ctx context.Context, userID uint) (domain.EmailVerificationToken, error)
	// UpdateVerificationEmailResendWindow records the fixed-window resend state after a successful resend.
	UpdateVerificationEmailResendWindow(ctx context.Context, tokenID uint, windowStartedAt time.Time, resendCount int) error
	// DeleteTokenByID deletes a token by ID
	DeleteTokenByID(ctx context.Context, tokenID uint) error
	// CreateRefreshToken stores a refresh token record.
	CreateRefreshToken(ctx context.Context, token domain.RefreshToken) (domain.RefreshToken, error)
	// FindActiveRefreshTokenByDigest retrieves an active refresh token by digest.
	FindActiveRefreshTokenByDigest(ctx context.Context, digest string, now time.Time) (domain.RefreshToken, error)
	// RotateRefreshToken revokes the current refresh token and stores a replacement.
	RotateRefreshToken(ctx context.Context, currentID uint, next domain.RefreshToken, now time.Time) (domain.RefreshToken, error)
	// RevokeRefreshTokenByDigest revokes the refresh token matching the digest.
	RevokeRefreshTokenByDigest(ctx context.Context, digest string, now time.Time) error
}

// VerificationEmailSender defines the interface for sending verification emails
type VerificationEmailSender interface {
	// SendVerificationEmail sends a verification email to the user
	SendVerificationEmail(ctx context.Context, user domain.User, verifyURL string) error
}

// PasswordHasher generates bcrypt hashes from plaintext passwords.
type PasswordHasher interface {
	GenerateHashPassword(password string) (string, error)
}

// AuthUseCaseInterface defines the interface for login business logic
type AuthUseCaseInterface interface {
	// Login authenticates a user and returns a short-lived access token.
	Login(ctx context.Context, req domain.LoginRequest) (string, error)
	// LoginTokens authenticates a user and returns an access token plus a refresh token.
	LoginTokens(ctx context.Context, req domain.LoginRequest) (domain.AuthTokens, error)
	// Register creates a new user account.
	Register(ctx context.Context, req domain.RegisterRequest) (domain.User, error)
	// VerifyEmail verifies a user's email address using a token
	VerifyEmail(ctx context.Context, req domain.VerifyEmailRequest) (domain.User, error)
	// ResendVerificationEmail resends the verification email
	ResendVerificationEmail(ctx context.Context, req domain.ResendVerificationRequest) error
	// Refresh rotates the refresh token and returns a new access token plus refresh token.
	Refresh(ctx context.Context, req domain.RefreshRequest) (domain.AuthTokens, error)
	// Logout revokes the refresh token for the current session.
	Logout(ctx context.Context, req domain.LogoutRequest) error
}

// AuthUseCase implements the login business logic
type AuthUseCase struct {
	repo            AuthRepository
	osw             oswrapper.OsWapperInterface
	tokenTTL        time.Duration
	refreshTokenTTL time.Duration
	mailer          VerificationEmailSender
	clock           timewrapper.ClockInterface
	hasher          PasswordHasher
}

// NewAuthUseCase creates a new AuthUseCase instance.
func NewAuthUseCase(repo AuthRepository, osw oswrapper.OsWapperInterface, mailer VerificationEmailSender, clock timewrapper.ClockInterface, hasher PasswordHasher) *AuthUseCase {
	if clock == nil {
		clock = timewrapper.NewClock()
	}
	return &AuthUseCase{
		repo:            repo,
		osw:             osw,
		tokenTTL:        readDurationSecondsEnv(osw, "AUTH_ACCESS_TOKEN_TTL_SECONDS", defaultAccessTokenTTL),
		refreshTokenTTL: readDurationSecondsEnv(osw, "AUTH_REFRESH_TOKEN_TTL_SECONDS", defaultRefreshTokenTTL),
		mailer:          mailer,
		clock:           clock,
		hasher:          hasher,
	}
}
