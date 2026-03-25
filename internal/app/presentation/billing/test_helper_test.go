package billing

import (
	"business/internal/billing/application"
	"business/internal/library/logger"
	mocklibrary "business/test/mock/library"
	"context"

	"github.com/stretchr/testify/mock"
)

type mockUseCase struct {
	mock.Mock
}

func (m *mockUseCase) List(ctx context.Context, query application.ListQuery) (application.ListResult, error) {
	args := m.Called(ctx, query)
	result, _ := args.Get(0).(application.ListResult)
	return result, args.Error(1)
}

func newTestLogger() logger.Interface {
	return mocklibrary.NewNopLogger()
}

func newTestController(usecase application.ListUseCase) *Controller {
	return NewController(usecase, newTestLogger())
}
