package di

import (
	macpresentation "business/internal/app/presentation/mailaccountconnection"
	"business/internal/library/crypto"
	"business/internal/library/gmailService"
	"business/internal/library/logger"
	"business/internal/library/mysql"
	"business/internal/library/oswrapper"
	"business/internal/library/timewrapper"
	"business/internal/mailaccountconnection/application"
	"business/internal/mailaccountconnection/infrastructure"

	"go.uber.org/dig"
)

// ProvideMailAccountConnectionDependencies registers mail account connection related dependencies.
func ProvideMailAccountConnectionDependencies(container *dig.Container) {
	// Repository
	_ = container.Provide(func(conn *mysql.MySQL, log *logger.Logger) *infrastructure.Repository {
		return infrastructure.NewRepository(conn.DB, log)
	})

	// OAuthConfigProvider (wraps existing OAuthConfigLoader)
	_ = container.Provide(func(osw *oswrapper.OsWrapper) *gmailService.OAuthConfigLoader {
		return gmailService.NewOAuthConfigLoader(osw)
	})

	// OAuthTokenExchanger
	_ = container.Provide(func() *infrastructure.OAuthTokenExchanger {
		return infrastructure.NewOAuthTokenExchanger()
	})

	// GmailProfileFetcher
	_ = container.Provide(func(gs *gmailService.Client, log *logger.Logger) *infrastructure.GmailProfileFetcher {
		return infrastructure.NewGmailProfileFetcher(gs, log)
	})

	// UseCase
	_ = container.Provide(func(
		repo *infrastructure.Repository,
		oauthCfg *gmailService.OAuthConfigLoader,
		exchanger *infrastructure.OAuthTokenExchanger,
		profiler *infrastructure.GmailProfileFetcher,
		vault *crypto.Vault,
		clock timewrapper.ClockInterface,
		log *logger.Logger,
	) *application.UseCase {
		return application.NewUseCase(repo, oauthCfg, exchanger, profiler, vault, clock, log)
	})

	// Controller
	_ = container.Provide(func(
		usecase *application.UseCase,
		log *logger.Logger,
	) *macpresentation.Controller {
		return macpresentation.NewController(usecase, log)
	})
}
