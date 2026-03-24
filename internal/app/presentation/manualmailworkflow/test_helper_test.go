package manualmailworkflow

import (
	"business/internal/library/logger"
	manualapp "business/internal/manualmailworkflow/application"
	mocklibrary "business/test/mock/library"
	"context"

	"github.com/stretchr/testify/mock"
)

type mockUseCase struct {
	mock.Mock
}

func (m *mockUseCase) Execute(ctx context.Context, cmd manualapp.Command) (manualapp.Result, error) {
	args := m.Called(ctx, cmd)
	result, _ := args.Get(0).(manualapp.Result)
	return result, args.Error(1)
}

func newTestLogger() logger.Interface {
	return mocklibrary.NewNopLogger()
}

func newTestController(usecase manualapp.UseCase) *Controller {
	return NewController(usecase, newTestLogger())
}
