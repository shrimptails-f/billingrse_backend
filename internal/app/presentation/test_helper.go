package presentation

import (
	"business/internal/auth/domain"
	"context"

	"github.com/stretchr/testify/mock"
)

type mockAuthUseCase struct {
	mock.Mock
}

func (m *mockAuthUseCase) Login(ctx context.Context, req domain.LoginRequest) (string, error) {
	args := m.Called(ctx, req)
	return args.String(0), args.Error(1)
}

func (m *mockAuthUseCase) Register(ctx context.Context, req domain.RegisterRequest) (domain.User, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(domain.User), args.Error(1)
}

func (m *mockAuthUseCase) VerifyEmail(ctx context.Context, req domain.VerifyEmailRequest) (domain.User, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(domain.User), args.Error(1)
}

func (m *mockAuthUseCase) ResendVerificationEmail(ctx context.Context, req domain.ResendVerificationRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}
