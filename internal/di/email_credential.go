package di

import (
	macpresentation "business/internal/app/presentation/mailaccountconnection"
	"business/internal/emailcredential/application"
	"business/internal/emailcredential/infrastructure"
	"business/internal/library/gmailService"
	"business/internal/library/logger"
	"business/internal/library/mysql"
	"business/internal/library/oswrapper"
	"business/internal/library/timewrapper"

	"go.uber.org/dig"
)

// ProvideEmailCredentialDependencies registers email credential related dependencies.
func ProvideEmailCredentialDependencies(container *dig.Container) {
	// Repository
	_ = container.Provide(func(conn *mysql.MySQL, log logger.Interface) *infrastructure.Repository {
		return infrastructure.NewRepository(conn.DB, log)
	})

	// OAuthConfigProvider (wraps existing OAuthConfigLoader)
	_ = container.Provide(func(osw *oswrapper.OsWrapper) application.OAuthConfigProvider {
		return gmailService.NewOAuthConfigLoader(osw)
	})

	// OAuthTokenExchanger
	_ = container.Provide(func() application.OAuthTokenExchanger {
		return infrastructure.NewOAuthTokenExchanger()
	})

	// GmailProfileFetcher
	_ = container.Provide(func(log logger.Interface) application.GmailProfileFetcher {
		return infrastructure.NewGmailProfileFetcher(log)
	})

	// UseCase
	_ = container.Provide(func(
		repo *infrastructure.Repository,
		oauthCfg application.OAuthConfigProvider,
		exchanger application.OAuthTokenExchanger,
		profiler application.GmailProfileFetcher,
		osw oswrapper.OsWapperInterface,
		clock timewrapper.ClockInterface,
		log logger.Interface,
	) application.UseCaseInterface {
		return application.NewUseCase(repo, oauthCfg, exchanger, profiler, osw, clock, log)
	})

	// Controller
	_ = container.Provide(func(
		usecase application.UseCaseInterface,
		log logger.Interface,
	) *macpresentation.Controller {
		return macpresentation.NewController(usecase, log)
	})
}
