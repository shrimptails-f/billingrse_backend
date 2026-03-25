package di

import (
	manualpresentation "business/internal/app/presentation/manualmailworkflow"
	billingapp "business/internal/billing/application"
	beapp "business/internal/billingeligibility/application"
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

	_ = container.Provide(func(usecase beapp.UseCase) *manualinfra.DirectBillingEligibilityAdapter {
		return manualinfra.NewDirectBillingEligibilityAdapter(usecase)
	})

	_ = container.Provide(func(usecase billingapp.UseCase) *manualinfra.DirectBillingAdapter {
		return manualinfra.NewDirectBillingAdapter(usecase)
	})

	_ = container.Provide(func(
		fetchStage *manualinfra.DirectManualMailFetchAdapter,
		analyzeStage *manualinfra.DirectMailAnalysisAdapter,
		vendorResolutionStage *manualinfra.DirectVendorResolutionAdapter,
		billingEligibilityStage *manualinfra.DirectBillingEligibilityAdapter,
		billingStage *manualinfra.DirectBillingAdapter,
		log *logger.Logger,
	) manualapp.UseCase {
		return manualapp.NewUseCase(fetchStage, analyzeStage, vendorResolutionStage, billingEligibilityStage, billingStage, log)
	})

	_ = container.Provide(func(
		runner manualapp.UseCase,
		log *logger.Logger,
	) *manualinfra.InProcessWorkflowDispatcher {
		return manualinfra.NewInProcessWorkflowDispatcher(runner, log)
	})

	_ = container.Provide(func(
		dispatcher *manualinfra.InProcessWorkflowDispatcher,
		log *logger.Logger,
	) manualapp.StartUseCase {
		return manualapp.NewStartUseCase(dispatcher, log)
	})

	_ = container.Provide(func(
		usecase manualapp.StartUseCase,
		log *logger.Logger,
	) *manualpresentation.Controller {
		return manualpresentation.NewController(usecase, log)
	})
}
