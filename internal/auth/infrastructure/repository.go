package infrastructure

import (
	"business/internal/library/logger"
	"time"

	"gorm.io/gorm"
)

// Repository provides database access for the auth domain
type Repository struct {
	db     *gorm.DB
	logger logger.Interface
}

// NewRepository creates a new Repository instance.
// If logger is nil, it defaults to logger.NewNop().
func NewRepository(db *gorm.DB, log logger.Interface) *Repository {
	if log == nil {
		log = logger.NewNop()
	}
	return &Repository{
		db:     db,
		logger: log.With(logger.String("component", "auth_repository")),
	}
}

// userRecord represents the database record structure for users table
type userRecord struct {
	ID              uint       `gorm:"column:id"`
	Name            string     `gorm:"column:name"`
	Email           string     `gorm:"column:email;unique"`
	Password        string     `gorm:"column:password"`
	EmailVerified   bool       `gorm:"column:email_verified"`
	EmailVerifiedAt *time.Time `gorm:"column:email_verified_at"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`
}

// TableName specifies the table name for userRecord
func (userRecord) TableName() string {
	return "users"
}

// emailVerificationTokenRecord represents the database record structure for email_verification_tokens table
type emailVerificationTokenRecord struct {
	ID         uint       `gorm:"column:id"`
	UserID     uint       `gorm:"column:user_id"`
	Token      string     `gorm:"column:token;unique"`
	ExpiresAt  time.Time  `gorm:"column:expires_at"`
	CreatedAt  time.Time  `gorm:"column:created_at"`
	ConsumedAt *time.Time `gorm:"column:consumed_at"`
}

// TableName specifies the table name for emailVerificationTokenRecord
func (emailVerificationTokenRecord) TableName() string {
	return "email_verification_tokens"
}
