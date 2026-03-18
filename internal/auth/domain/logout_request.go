package domain

// LogoutRequest carries the refresh token that should be revoked.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}
