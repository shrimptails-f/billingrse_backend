package di

import (
	manualpresentation "business/internal/app/presentation/manualmailworkflow"
	billingapp "business/internal/billing/application"
	beapp "business/internal/billingeligibility/application"
	"business/internal/library/logger"
	"business/internal/library/timewrapper"
	maapp "business/internal/mailanalysis/application"
	mfapp "business/internal/mailfetch/application"
	manualapp "business/internal/manualmailworkflow/application"
	manualinfra "business/internal/manualmailworkflow/infrastructure"
	vrapp "business/internal/vendorresolution/application"

	"go.uber.org/dig"
	"gorm.io/gorm"
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
		db *gorm.DB,
		clock *timewrapper.Clock,
		log *logger.Logger,
	) *manualinfra.GormWorkflowStatusRepository {
		return manualinfra.NewGormWorkflowStatusRepository(db, clock, log)
	})

	_ = container.Provide(func(
		fetchStage *manualinfra.DirectManualMailFetchAdapter,
		analyzeStage *manualinfra.DirectMailAnalysisAdapter,
		vendorResolutionStage *manualinfra.DirectVendorResolutionAdapter,
		billingEligibilityStage *manualinfra.DirectBillingEligibilityAdapter,
		billingStage *manualinfra.DirectBillingAdapter,
		repository *manualinfra.GormWorkflowStatusRepository,
		clock *timewrapper.Clock,
		log *logger.Logger,
	) manualapp.UseCase {
		return manualapp.NewUseCase(fetchStage, analyzeStage, vendorResolutionStage, billingEligibilityStage, billingStage, repository, clock, log)
	})

	_ = container.Provide(func(
		runner manualapp.UseCase,
		log *logger.Logger,
	) *manualinfra.InProcessWorkflowDispatcher {
		return manualinfra.NewInProcessWorkflowDispatcher(runner, log)
	})

	_ = container.Provide(func(
		dispatcher *manualinfra.InProcessWorkflowDispatcher,
		repository *manualinfra.GormWorkflowStatusRepository,
		clock *timewrapper.Clock,
		log *logger.Logger,
	) manualapp.StartUseCase {
		return manualapp.NewStartUseCase(dispatcher, repository, clock, log)
	})

	_ = container.Provide(func(
		repository *manualinfra.GormWorkflowStatusRepository,
		log *logger.Logger,
	) manualapp.ListUseCase {
		return manualapp.NewListUseCase(repository, log)
	})

	_ = container.Provide(func(
		startUseCase manualapp.StartUseCase,
		listUseCase manualapp.ListUseCase,
		log *logger.Logger,
	) *manualpresentation.Controller {
		return manualpresentation.NewController(startUseCase, listUseCase, log)
	})
}
