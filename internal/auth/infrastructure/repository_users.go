package infrastructure

import (
	"business/internal/auth/domain"
	"business/internal/library/logger"
	"context"
	"errors"
	"fmt"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

// GetUserByEmail retrieves a user by email address from the users table.
// Returns gorm.ErrRecordNotFound as-is when no user is found.
// Other errors are wrapped with context.
func (r *Repository) GetUserByEmail(ctx context.Context, email domain.EmailAddress) (domain.User, error) {
	var record userRecord

	// Fallback to Background if ctx is nil
	if ctx == nil {
		ctx = context.Background()
	}

	err := r.db.
		WithContext(ctx).
		Select("id, name, email, password, email_verified, email_verified_at, created_at, updated_at").
		Where("email = ?", email.String()).
		First(&record).
		Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.User{}, gorm.ErrRecordNotFound
		}
		r.logger.Error("failed to get user by email", logger.String("email", email.String()), logger.Err(err))
		return domain.User{}, fmt.Errorf("failed to get user by email: %w", err)
	}

	name, err := domain.NewUserName(record.Name)
	if err != nil {
		return domain.User{}, fmt.Errorf("invalid user name: %w", err)
	}
	emailAddress, err := domain.NewEmailAddress(record.Email)
	if err != nil {
		return domain.User{}, fmt.Errorf("invalid email address: %w", err)
	}

	domainUser := domain.User{
		ID:              record.ID,
		Name:            name,
		Email:           emailAddress,
		PasswordHash:    record.Password,
		EmailVerified:   record.EmailVerified,
		EmailVerifiedAt: record.EmailVerifiedAt,
		CreatedAt:       record.CreatedAt,
		UpdatedAt:       record.UpdatedAt,
	}

	return domainUser, nil
}

// CreateUser inserts a new user into the users table.
// Returns gorm.ErrDuplicatedKey if the email already exists.
func (r *Repository) CreateUser(ctx context.Context, user domain.User) (domain.User, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	record := userRecord{
		Name:            user.Name.String(),
		Email:           user.Email.String(),
		Password:        user.PasswordHash,
		EmailVerified:   user.EmailVerified,
		EmailVerifiedAt: user.EmailVerifiedAt,
	}

	err := r.db.WithContext(ctx).Create(&record).Error
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return domain.User{}, gorm.ErrDuplicatedKey
		}
		r.logger.Error("failed to create user", logger.String("email", user.Email.String()), logger.Err(err))
		return domain.User{}, fmt.Errorf("failed to create user: %w", err)
	}

	name, err := domain.NewUserName(record.Name)
	if err != nil {
		return domain.User{}, fmt.Errorf("invalid user name: %w", err)
	}
	emailAddress, err := domain.NewEmailAddress(record.Email)
	if err != nil {
		return domain.User{}, fmt.Errorf("invalid email address: %w", err)
	}

	return domain.User{
		ID:              record.ID,
		Name:            name,
		Email:           emailAddress,
		PasswordHash:    record.Password,
		EmailVerified:   record.EmailVerified,
		EmailVerifiedAt: record.EmailVerifiedAt,
		CreatedAt:       record.CreatedAt,
		UpdatedAt:       record.UpdatedAt,
	}, nil
}

// GetUserByID retrieves a user by ID from the users table.
// Returns gorm.ErrRecordNotFound as-is when no user is found.
func (r *Repository) GetUserByID(ctx context.Context, id uint) (domain.User, error) {
	var record userRecord

	if ctx == nil {
		ctx = context.Background()
	}

	err := r.db.
		WithContext(ctx).
		Select("id, name, email, password, email_verified, email_verified_at, created_at, updated_at").
		Where("id = ?", id).
		First(&record).
		Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.User{}, gorm.ErrRecordNotFound
		}
		r.logger.Error("failed to get user by id", logger.Uint("user_id", id), logger.Err(err))
		return domain.User{}, fmt.Errorf("failed to get user by id: %w", err)
	}

	name, err := domain.NewUserName(record.Name)
	if err != nil {
		return domain.User{}, fmt.Errorf("invalid user name: %w", err)
	}
	emailAddress, err := domain.NewEmailAddress(record.Email)
	if err != nil {
		return domain.User{}, fmt.Errorf("invalid email address: %w", err)
	}

	domainUser := domain.User{
		ID:              record.ID,
		Name:            name,
		Email:           emailAddress,
		PasswordHash:    record.Password,
		EmailVerified:   record.EmailVerified,
		EmailVerifiedAt: record.EmailVerifiedAt,
		CreatedAt:       record.CreatedAt,
		UpdatedAt:       record.UpdatedAt,
	}

	return domainUser, nil
}

// DeleteUserByID deletes a user by ID from the users table.
// Returns gorm.ErrRecordNotFound as-is when no user is found.
func (r *Repository) DeleteUserByID(ctx context.Context, id uint) error {
	if ctx == nil {
		ctx = context.Background()
	}

	result := r.db.WithContext(ctx).Delete(&userRecord{}, id)

	if result.Error != nil {
		r.logger.Error("failed to delete user", logger.Uint("user_id", id), logger.Err(result.Error))
		return fmt.Errorf("failed to delete user: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}
