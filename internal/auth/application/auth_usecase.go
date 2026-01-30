package application

import (
	"business/internal/auth/domain"
	"business/internal/library/oswrapper"
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
	// DeleteTokenByID deletes a token by ID
	DeleteTokenByID(ctx context.Context, tokenID uint) error
}

// VerificationEmailSender defines the interface for sending verification emails
type VerificationEmailSender interface {
	// SendVerificationEmail sends a verification email to the user
	SendVerificationEmail(ctx context.Context, user domain.User, verifyURL string) error
}

// AuthUseCaseInterface defines the interface for login business logic
type AuthUseCaseInterface interface {
	// Login authenticates a user and returns a JWT token
	Login(ctx context.Context, req domain.LoginRequest) (string, error)
	// Register creates a new user account.
	Register(ctx context.Context, req domain.RegisterRequest) (domain.User, error)
	// VerifyEmail verifies a user's email address using a token
	VerifyEmail(ctx context.Context, req domain.VerifyEmailRequest) (domain.User, error)
	// ResendVerificationEmail resends the verification email
	ResendVerificationEmail(ctx context.Context, req domain.ResendVerificationRequest) error
}

// AuthUseCase implements the login business logic
type AuthUseCase struct {
	repo     AuthRepository
	osw      oswrapper.OsWapperInterface
	tokenTTL time.Duration
	mailer   VerificationEmailSender
	clock    func() time.Time
}

// NewAuthUseCase creates a new AuthUseCase instance
func NewAuthUseCase(repo AuthRepository, osw oswrapper.OsWapperInterface, tokenTTL time.Duration, mailer VerificationEmailSender, clock func() time.Time) AuthUseCaseInterface {
	if clock == nil {
		clock = time.Now
	}
	return &AuthUseCase{
		repo:     repo,
		osw:      osw,
		tokenTTL: tokenTTL,
		mailer:   mailer,
		clock:    clock,
	}
}
