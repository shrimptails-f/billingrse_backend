package mailaccountconnection

import (
	"business/internal/emailcredential/application"
	"business/internal/emailcredential/domain"
	"business/internal/library/logger"
	mocklibrary "business/test/mock/library"
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

type mockUseCase struct {
	mock.Mock
}

func (m *mockUseCase) Authorize(ctx context.Context, userID uint) (application.AuthorizeResult, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(application.AuthorizeResult), args.Error(1)
}

func (m *mockUseCase) Callback(ctx context.Context, userID uint, code, state string) error {
	args := m.Called(ctx, userID, code, state)
	return args.Error(0)
}

func (m *mockUseCase) ListConnections(ctx context.Context, userID uint) ([]domain.ConnectionView, error) {
	args := m.Called(ctx, userID)
	connections, _ := args.Get(0).([]domain.ConnectionView)
	return connections, args.Error(1)
}

func (m *mockUseCase) Disconnect(ctx context.Context, userID uint, connectionID uint) error {
	args := m.Called(ctx, userID, connectionID)
	return args.Error(0)
}

func newTestLogger() logger.Interface {
	return mocklibrary.NewNopLogger()
}

func newTestController(usecase application.UseCaseInterface) *Controller {
	return NewController(usecase, newTestLogger())
}

func fixedExpiresAt() time.Time {
	return time.Date(2026, 3, 19, 12, 10, 0, 0, time.UTC)
}
