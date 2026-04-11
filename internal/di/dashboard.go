package di

import (
	dashboardpresentation "business/internal/app/presentation/dashboard"
	dashboardqueryapp "business/internal/dashboardquery/application"
	dashboardqueryinfra "business/internal/dashboardquery/infrastructure"
	"business/internal/library/logger"
	"business/internal/library/timewrapper"

	"go.uber.org/dig"
	"gorm.io/gorm"
)

// ProvideDashboardDependencies registers dashboard summary dependencies.
func ProvideDashboardDependencies(container *dig.Container) {
	_ = container.Provide(func(
		db *gorm.DB,
		log *logger.Logger,
	) *dashboardqueryinfra.DashboardSummaryRepository {
		return dashboardqueryinfra.NewDashboardSummaryRepository(db, log)
	})

	_ = container.Provide(func(
		repository *dashboardqueryinfra.DashboardSummaryRepository,
		clock *timewrapper.Clock,
		log *logger.Logger,
	) *dashboardqueryapp.SummaryUseCase {
		return dashboardqueryapp.NewSummaryUseCase(repository, clock, log)
	})

	_ = container.Provide(func(
		usecase *dashboardqueryapp.SummaryUseCase,
		log *logger.Logger,
	) *dashboardpresentation.Controller {
		return dashboardpresentation.NewController(usecase, log)
	})
}
