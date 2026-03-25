package seeders

import (
	model "business/tools/migrations/models"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

const gmailConnectionType = "gmail"

type seedConnection struct {
	UserEmail          string
	GmailAddress       string
	AccessToken        string
	AccessTokenDigest  string
	RefreshToken       string
	RefreshTokenDigest string
	TokenExpiry        *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// CreateMailAccountConnections は手動 workflow サンプルが参照するメール連携データを投入する。
func CreateMailAccountConnections(tx *gorm.DB) error {
	tokenExpiry := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	connections := []seedConnection{
		{
			UserEmail:          adminUserEmail,
			GmailAddress:       "admin.billing@gmail.com",
			AccessToken:        "seed-admin-access-token-primary",
			AccessTokenDigest:  "seed-admin-access-token-primary-digest",
			RefreshToken:       "seed-admin-refresh-token-primary",
			RefreshTokenDigest: "seed-admin-refresh-token-primary-digest",
			TokenExpiry:        timePtr(tokenExpiry),
			CreatedAt:          time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC),
			UpdatedAt:          time.Date(2026, 3, 25, 8, 0, 0, 0, time.UTC),
		},
		{
			UserEmail:          adminUserEmail,
			GmailAddress:       "admin.subscriptions@gmail.com",
			AccessToken:        "seed-admin-access-token-secondary",
			AccessTokenDigest:  "seed-admin-access-token-secondary-digest",
			RefreshToken:       "seed-admin-refresh-token-secondary",
			RefreshTokenDigest: "seed-admin-refresh-token-secondary-digest",
			TokenExpiry:        timePtr(tokenExpiry),
			CreatedAt:          time.Date(2026, 3, 21, 11, 30, 0, 0, time.UTC),
			UpdatedAt:          time.Date(2026, 3, 25, 8, 5, 0, 0, time.UTC),
		},
		{
			UserEmail:          testUserEmail,
			GmailAddress:       "test.billing@gmail.com",
			AccessToken:        "seed-test-access-token-primary",
			AccessTokenDigest:  "seed-test-access-token-primary-digest",
			RefreshToken:       "seed-test-refresh-token-primary",
			RefreshTokenDigest: "seed-test-refresh-token-primary-digest",
			TokenExpiry:        timePtr(tokenExpiry),
			CreatedAt:          time.Date(2026, 3, 22, 12, 0, 0, 0, time.UTC),
			UpdatedAt:          time.Date(2026, 3, 25, 8, 10, 0, 0, time.UTC),
		},
	}

	for _, connection := range connections {
		user, err := findUserByEmail(tx, connection.UserEmail)
		if err != nil {
			return err
		}

		record := model.EmailCredential{
			UserID:             user.ID,
			Type:               gmailConnectionType,
			GmailAddress:       strings.ToLower(strings.TrimSpace(connection.GmailAddress)),
			KeyVersion:         1,
			AccessToken:        connection.AccessToken,
			AccessTokenDigest:  connection.AccessTokenDigest,
			RefreshToken:       connection.RefreshToken,
			RefreshTokenDigest: connection.RefreshTokenDigest,
			TokenExpiry:        connection.TokenExpiry,
			CreatedAt:          connection.CreatedAt,
			UpdatedAt:          connection.UpdatedAt,
		}
		if err := upsertCredential(tx, record); err != nil {
			return err
		}
	}

	return nil
}

func findUserByEmail(tx *gorm.DB, email string) (model.User, error) {
	var user model.User
	if err := tx.Where("email = ?", strings.TrimSpace(email)).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.User{}, fmt.Errorf("seed user not found: %s", email)
		}
		return model.User{}, fmt.Errorf("failed to find seed user by email %s: %w", email, err)
	}

	return user, nil
}

func upsertCredential(tx *gorm.DB, credential model.EmailCredential) error {
	var existing model.EmailCredential
	err := tx.
		Where("user_id = ? AND type = ? AND gmail_address = ? AND o_auth_state IS NULL",
			credential.UserID,
			credential.Type,
			credential.GmailAddress,
		).
		First(&existing).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return tx.Create(&credential).Error
	case err != nil:
		return fmt.Errorf("failed to find email credential for user_id=%d gmail=%s: %w", credential.UserID, credential.GmailAddress, err)
	default:
		return tx.Model(&existing).Updates(map[string]interface{}{
			"key_version":          credential.KeyVersion,
			"access_token":         credential.AccessToken,
			"access_token_digest":  credential.AccessTokenDigest,
			"refresh_token":        credential.RefreshToken,
			"refresh_token_digest": credential.RefreshTokenDigest,
			"token_expiry":         credential.TokenExpiry,
			"updated_at":           credential.UpdatedAt,
		}).Error
	}
}

func findCredentialByUserAndGmail(tx *gorm.DB, userID uint, gmailAddress string) (model.EmailCredential, error) {
	var credential model.EmailCredential
	err := tx.
		Where("user_id = ? AND type = ? AND gmail_address = ? AND o_auth_state IS NULL",
			userID,
			gmailConnectionType,
			strings.ToLower(strings.TrimSpace(gmailAddress)),
		).
		First(&credential).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.EmailCredential{}, fmt.Errorf("seed email credential not found: user_id=%d gmail=%s", userID, gmailAddress)
		}
		return model.EmailCredential{}, fmt.Errorf("failed to find seed email credential: %w", err)
	}

	return credential, nil
}
