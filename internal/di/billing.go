package di

import (
	billingpresentation "business/internal/app/presentation/billing"
	billingapp "business/internal/billing/application"
	billinginfra "business/internal/billing/infrastructure"
	billingqueryapp "business/internal/billingquery/application"
	billingqueryinfra "business/internal/billingquery/infrastructure"
	"business/internal/library/logger"
	"business/internal/library/timewrapper"

	"go.uber.org/dig"
	"gorm.io/gorm"
)

// ProvideBillingDependencies registers billing dependencies.
func ProvideBillingDependencies(container *dig.Container) {
	_ = container.Provide(func(
		db *gorm.DB,
		clock *timewrapper.Clock,
		log *logger.Logger,
	) *billinginfra.BillingRepository {
		return billinginfra.NewBillingRepository(db, clock, log)
	})

	_ = container.Provide(func(
		db *gorm.DB,
		clock *timewrapper.Clock,
		log *logger.Logger,
	) *billingqueryinfra.BillingQueryRepository {
		return billingqueryinfra.NewBillingQueryRepository(db, clock, log)
	})

	_ = container.Provide(func(
		repository *billinginfra.BillingRepository,
		log *logger.Logger,
	) billingapp.UseCase {
		return billingapp.NewUseCase(repository, log)
	})

	_ = container.Provide(func(
		repository *billingqueryinfra.BillingQueryRepository,
		log *logger.Logger,
	) *billingqueryapp.ListUseCase {
		return billingqueryapp.NewListUseCase(repository, log)
	})

	_ = container.Provide(func(
		repository *billingqueryinfra.BillingQueryRepository,
		clock *timewrapper.Clock,
		log *logger.Logger,
	) *billingqueryapp.MonthlyTrendUseCase {
		return billingqueryapp.NewMonthlyTrendUseCase(repository, clock, log)
	})

	_ = container.Provide(func(
		repository *billingqueryinfra.BillingQueryRepository,
		log *logger.Logger,
	) *billingqueryapp.MonthDetailUseCase {
		return billingqueryapp.NewMonthDetailUseCase(repository, log)
	})

	_ = container.Provide(func(
		usecase *billingqueryapp.ListUseCase,
		monthlyTrendUseCase *billingqueryapp.MonthlyTrendUseCase,
		monthDetailUseCase *billingqueryapp.MonthDetailUseCase,
		log *logger.Logger,
	) *billingpresentation.Controller {
		return billingpresentation.NewController(usecase, monthlyTrendUseCase, monthDetailUseCase, log)
	})
}
