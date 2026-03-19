package domain

import "time"

// EmailCredential represents a stored email credential (connected mail account).
type EmailCredential struct {
	ID                 uint
	UserID             uint
	Type               string
	GmailAddress       string
	KeyVersion         int16
	AccessToken        string
	AccessTokenDigest  string
	RefreshToken       string
	RefreshTokenDigest string
	TokenExpiry        *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
