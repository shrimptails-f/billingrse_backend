package seeders

import (
	model "business/tools/migrations/models"
	"errors"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	adminUserEmail    = "admin@example.com"
	testUserEmail     = "test@example.com"
	testUser2Email    = "user@example.com"
	defaultSeedTZName = "UTC"
)

// CreateUser はユーザーのサンプルデータを投入する。
func CreateUser(tx *gorm.DB) error {
	now := seedNow()
	users := []model.User{
		{
			Name:            "admin",
			Email:           adminUserEmail,
			Password:        hashPassword("password123"),
			EmailVerified:   true,
			EmailVerifiedAt: timePtr(now),
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			Name:            "test_user",
			Email:           testUserEmail,
			Password:        hashPassword("testpass"),
			EmailVerified:   true,
			EmailVerifiedAt: timePtr(now),
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			Name:            "test_user2",
			Email:           testUser2Email,
			Password:        hashPassword("userpass"),
			EmailVerified:   true,
			EmailVerifiedAt: timePtr(now),
			CreatedAt:       now,
			UpdatedAt:       now,
		},
	}

	for _, user := range users {
		if err := upsertUserByEmail(tx, user); err != nil {
			log.Printf("failed create user mail: %s", user.Email)
			return err
		}
	}

	return nil
}

func hashPassword(password string) string {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[WARN] Failed to hash password: %v", err)
		return ""
	}
	return string(hashed)
}

func upsertUserByEmail(tx *gorm.DB, user model.User) error {
	var existing model.User
	err := tx.Where("email = ?", user.Email).First(&existing).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return tx.Create(&user).Error
	case err != nil:
		return fmt.Errorf("failed to find user by email %s: %w", user.Email, err)
	default:
		return tx.Model(&existing).Updates(map[string]interface{}{
			"name":              user.Name,
			"password":          user.Password,
			"email_verified":    user.EmailVerified,
			"email_verified_at": user.EmailVerifiedAt,
			"updated_at":        user.UpdatedAt,
		}).Error
	}
}

func seedNow() time.Time {
	location, err := time.LoadLocation(defaultSeedTZName)
	if err != nil {
		return time.Date(2026, 3, 25, 9, 0, 0, 0, time.UTC)
	}

	return time.Date(2026, 3, 25, 9, 0, 0, 0, location).UTC()
}

func timePtr(value time.Time) *time.Time {
	return &value
}
