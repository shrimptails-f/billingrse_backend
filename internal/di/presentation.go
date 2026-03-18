package di

import (
	authpresentation "business/internal/app/presentation/auth"
	authapp "business/internal/auth/application"
	"business/internal/library/logger"
	"business/internal/library/oswrapper"

	"go.uber.org/dig"
)

// ProvidePresentationDependencies プレゼンテーション層の依存注入設定
func ProvidePresentationDependencies(container *dig.Container) {

	// Auth controller dependencies.
	_ = container.Provide(func(
		usecase authapp.AuthUseCaseInterface,
		log logger.Interface,
		osw oswrapper.OsWapperInterface,
	) *authpresentation.Controller {
		return authpresentation.NewController(usecase, log, osw)
	})

}
