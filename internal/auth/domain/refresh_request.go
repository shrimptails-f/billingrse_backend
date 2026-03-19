package domain

// RefreshRequest carries the refresh token used to mint a new token pair.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}
