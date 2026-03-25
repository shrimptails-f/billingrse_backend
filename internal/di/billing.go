package di

import (
	billingapp "business/internal/billing/application"
	billinginfra "business/internal/billing/infrastructure"
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
	) *billinginfra.GormBillingRepository {
		return billinginfra.NewGormBillingRepository(db, clock, log)
	})

	_ = container.Provide(func(
		repository *billinginfra.GormBillingRepository,
		log *logger.Logger,
	) billingapp.UseCase {
		return billingapp.NewUseCase(repository, log)
	})
}
