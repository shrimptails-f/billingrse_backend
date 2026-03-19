package application

import (
	"business/internal/auth/domain"
	"context"
)

// Login authenticates a user with email and password, and returns a JWT token.
// Returns ErrInvalidCredentials if the email doesn't exist or password is incorrect.
func (uc *AuthUseCase) Login(ctx context.Context, req domain.LoginRequest) (string, error) {
	user, err := uc.authenticateUser(ctx, req)
	if err != nil {
		return "", err
	}

	return uc.issueAccessToken(user.ID, uc.clock.Now())
}
