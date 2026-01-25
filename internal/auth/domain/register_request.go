package domain

// RegisterRequest represents the registration request payload.
type RegisterRequest struct {
	Email    string
	Name     string
	Password string
}
