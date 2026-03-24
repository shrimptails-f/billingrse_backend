package di

import (
	"business/internal/library/logger"
	vrapp "business/internal/vendorresolution/application"
	vrinfra "business/internal/vendorresolution/infrastructure"

	"go.uber.org/dig"
	"gorm.io/gorm"
)

// ProvideVendorResolutionDependencies は vendorresolution 関連の依存を登録する。
func ProvideVendorResolutionDependencies(container *dig.Container) {
	_ = container.Provide(func(db *gorm.DB, log *logger.Logger) *vrinfra.VendorResolutionRepository {
		return vrinfra.NewVendorResolutionRepository(db, log)
	})

	_ = container.Provide(func(db *gorm.DB, log *logger.Logger) *vrinfra.VendorRegistrationRepository {
		return vrinfra.NewVendorRegistrationRepository(db, log)
	})

	_ = container.Provide(func(resolutionRepository *vrinfra.VendorResolutionRepository, registrationRepository *vrinfra.VendorRegistrationRepository, log *logger.Logger) vrapp.UseCase {
		return vrapp.NewUseCase(resolutionRepository, registrationRepository, log)
	})
}
