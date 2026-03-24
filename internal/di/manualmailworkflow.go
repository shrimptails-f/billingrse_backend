package di

import (
	manualpresentation "business/internal/app/presentation/manualmailworkflow"
	"business/internal/library/logger"
	maapp "business/internal/mailanalysis/application"
	mfapp "business/internal/mailfetch/application"
	manualapp "business/internal/manualmailworkflow/application"
	manualinfra "business/internal/manualmailworkflow/infrastructure"
	vrapp "business/internal/vendorresolution/application"

	"go.uber.org/dig"
)

// ProvideManualMailWorkflowDependencies registers manual mail workflow dependencies.
func ProvideManualMailWorkflowDependencies(container *dig.Container) {
	_ = container.Provide(func(usecase mfapp.UseCase) *manualinfra.DirectManualMailFetchAdapter {
		return manualinfra.NewDirectManualMailFetchAdapter(usecase)
	})

	_ = container.Provide(func(usecase maapp.UseCase) *manualinfra.DirectMailAnalysisAdapter {
		return manualinfra.NewDirectMailAnalysisAdapter(usecase)
	})

	_ = container.Provide(func(usecase vrapp.UseCase) *manualinfra.DirectVendorResolutionAdapter {
		return manualinfra.NewDirectVendorResolutionAdapter(usecase)
	})

	_ = container.Provide(func(
		fetchStage *manualinfra.DirectManualMailFetchAdapter,
		analyzeStage *manualinfra.DirectMailAnalysisAdapter,
		vendorResolutionStage *manualinfra.DirectVendorResolutionAdapter,
		log *logger.Logger,
	) manualapp.UseCase {
		return manualapp.NewUseCase(fetchStage, analyzeStage, vendorResolutionStage, log)
	})

	_ = container.Provide(func(
		usecase manualapp.UseCase,
		log *logger.Logger,
	) *manualpresentation.Controller {
		return manualpresentation.NewController(usecase, log)
	})
}
