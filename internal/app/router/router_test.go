package v1_test

import (
	"context"
	"errors"
	"fmt"

	"business/internal/auth/domain"
)

type stubAuthUserProvider struct{}

func (s *stubAuthUserProvider) GetUserByID(ctx context.Context, id uint) (domain.User, error) {
	return domain.User{
		ID:            id,
		EmailVerified: true,
	}, nil
}

type stubOsWrapper struct{}

func (s *stubOsWrapper) ReadFile(path string) (string, error) {
	return "", errors.New("not implemented")
}

func (s *stubOsWrapper) GetEnv(key string) (string, error) {
	if key == "JWT_SECRET_KEY" {
		return "test-secret", nil
	}
	return "", fmt.Errorf("environment variable %s not set", key)
}

type stubAuthUseCase struct{}

func (s *stubAuthUseCase) Login(ctx context.Context, req domain.LoginRequest) (string, error) {
	return "dummy-token", nil
}

func (s *stubAuthUseCase) Register(ctx context.Context, req domain.RegisterRequest) (domain.User, error) {
	return domain.User{}, nil
}

func (s *stubAuthUseCase) VerifyEmail(ctx context.Context, req domain.VerifyEmailRequest) (domain.User, error) {
	return domain.User{}, nil
}

func (s *stubAuthUseCase) ResendVerificationEmail(ctx context.Context, req domain.ResendVerificationRequest) error {
	return nil
}

type stubAgentUsecase struct{}

type stubEmailCredentialUsecase struct{}
