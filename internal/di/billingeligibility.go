package di

import (
	beapp "business/internal/billingeligibility/application"
	"business/internal/library/logger"

	"go.uber.org/dig"
)

// ProvideBillingEligibilityDependencies registers billingeligibility dependencies.
func ProvideBillingEligibilityDependencies(container *dig.Container) {
	_ = container.Provide(func(log *logger.Logger) beapp.UseCase {
		return beapp.NewUseCase(log)
	})
}
