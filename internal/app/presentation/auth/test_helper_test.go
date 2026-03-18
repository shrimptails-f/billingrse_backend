package auth

import (
	"business/internal/auth/application"
	"business/internal/auth/domain"
	"business/internal/library/logger"
	mocklibrary "business/test/mock/library"
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

func newTestLogger() logger.Interface {
	return mocklibrary.NewNopLogger()
}

func newCapturingTestLogger() *mocklibrary.CapturingLogger {
	return mocklibrary.NewCapturingLogger()
}

var authControllerEnvDefaults = map[string]string{
	"APP":    "local",
	"DOMAIN": "localhost",
}

// newTestController wires Controller with shared test mocks.
func newTestController(usecase application.AuthUseCaseInterface, log logger.Interface) *Controller {
	return NewController(usecase, log, mocklibrary.NewOsWrapperMock(authControllerEnvDefaults).WithEnv(nil))
}

func newTestControllerWithVars(usecase application.AuthUseCaseInterface, log logger.Interface, vars map[string]string) *Controller {
	return NewController(usecase, log, mocklibrary.NewOsWrapperMock(authControllerEnvDefaults).WithEnv(vars))
}
