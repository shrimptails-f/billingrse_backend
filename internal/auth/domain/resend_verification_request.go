package domain

// ResendVerificationRequest represents the request to resend verification email
type ResendVerificationRequest struct {
	Email    string
	Password string
}
