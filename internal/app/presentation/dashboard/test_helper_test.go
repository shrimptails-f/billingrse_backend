package dashboard

import (
	dashboardqueryapp "business/internal/dashboardquery/application"
	"business/internal/library/logger"
	mocklibrary "business/test/mock/library"
	"context"

	"github.com/stretchr/testify/mock"
)

type mockSummaryUseCase struct {
	mock.Mock
}

func (m *mockSummaryUseCase) Get(ctx context.Context, query dashboardqueryapp.SummaryQuery) (dashboardqueryapp.SummaryResult, error) {
	args := m.Called(ctx, query)
	result, _ := args.Get(0).(dashboardqueryapp.SummaryResult)
	return result, args.Error(1)
}

func newTestLogger() logger.Interface {
	return mocklibrary.NewNopLogger()
}

func newTestController(usecase dashboardqueryapp.SummaryUseCaseInterface) *Controller {
	return NewController(usecase, newTestLogger())
}
