package seeders

import (
	model "business/tools/migrations/models"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// CreateUser はユーザーのサンプルデータを投入する。
func CreateUser(tx *gorm.DB) error {
	// パスワードをハッシュ化するヘルパー関数
	hashPassword := func(password string) string {
		hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("[WARN] Failed to hash password: %v", err)
			return ""
		}
		return string(hashed)
	}

	now := time.Now()
	users := []model.User{
		{
			Name:            "admin",
			Email:           "admin@example.com",
			Password:        hashPassword("password123"),
			EmailVerified:   true,
			EmailVerifiedAt: &now,
		},
		{
			Name:            "test_user",
			Email:           "test@example.com",
			Password:        hashPassword("testpass"),
			EmailVerified:   true,
			EmailVerifiedAt: &now,
		},
		{
			Name:            "test_user2",
			Email:           "user@example.com",
			Password:        hashPassword("userpass"),
			EmailVerified:   true,
			EmailVerifiedAt: &now,
		},
	}

	// ユーザーを作成
	for _, user := range users {
		err := tx.Create(&user).Error
		if err != nil {
			log.Printf("failed create user mail: %s", user.Email)
			return err
		}
	}

	return nil
}
