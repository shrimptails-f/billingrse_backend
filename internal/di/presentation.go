package di

import (
	"business/internal/app/presentation"
	authapp "business/internal/auth/application"
	"business/internal/library/logger"
	"business/internal/library/oswrapper"

	"go.uber.org/dig"
)

// ProvidePresentationDependencies プレゼンテーション層の依存注入設定
func ProvidePresentationDependencies(container *dig.Container) {

	// AuthControllerの依存注入
	_ = container.Provide(func(
		usecase authapp.AuthUseCaseInterface,
		log logger.Interface,
		osw oswrapper.OsWapperInterface,
	) *presentation.AuthController {
		return presentation.NewAuthController(usecase, log.With(logger.String("component", "auth_controller")), osw)
	})

}
