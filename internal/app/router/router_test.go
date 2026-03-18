package v1_test

import (
	"context"
	"time"

	"business/internal/auth/domain"
)

type stubAuthUserProvider struct{}

func (s *stubAuthUserProvider) GetUserByID(ctx context.Context, id uint) (domain.User, error) {
	verifiedAt := time.Now()
	return domain.User{
		ID:              id,
		EmailVerifiedAt: &verifiedAt,
	}, nil
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
