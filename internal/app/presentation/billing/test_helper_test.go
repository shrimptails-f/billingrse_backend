package billing

import (
	billingqueryapp "business/internal/billingquery/application"
	"business/internal/library/logger"
	mocklibrary "business/test/mock/library"
	"context"

	"github.com/stretchr/testify/mock"
)

type mockUseCase struct {
	mock.Mock
}

func (m *mockUseCase) List(ctx context.Context, query billingqueryapp.ListQuery) (billingqueryapp.ListResult, error) {
	args := m.Called(ctx, query)
	result, _ := args.Get(0).(billingqueryapp.ListResult)
	return result, args.Error(1)
}

type mockMonthDetailUseCase struct {
	mock.Mock
}

func (m *mockMonthDetailUseCase) Get(ctx context.Context, query billingqueryapp.MonthDetailQuery) (billingqueryapp.MonthDetailResult, error) {
	args := m.Called(ctx, query)
	result, _ := args.Get(0).(billingqueryapp.MonthDetailResult)
	return result, args.Error(1)
}

type mockMonthlyTrendUseCase struct {
	mock.Mock
}

func (m *mockMonthlyTrendUseCase) Get(ctx context.Context, query billingqueryapp.MonthlyTrendQuery) (billingqueryapp.MonthlyTrendResult, error) {
	args := m.Called(ctx, query)
	result, _ := args.Get(0).(billingqueryapp.MonthlyTrendResult)
	return result, args.Error(1)
}

func newTestLogger() logger.Interface {
	return mocklibrary.NewNopLogger()
}

func newTestController(
	usecase billingqueryapp.ListUseCaseInterface,
	monthlyTrendUseCase billingqueryapp.MonthlyTrendUseCaseInterface,
	monthDetailUseCase billingqueryapp.MonthDetailUseCaseInterface,
) *Controller {
	return NewController(usecase, monthlyTrendUseCase, monthDetailUseCase, newTestLogger())
}
