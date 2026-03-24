package di

import (
	"business/internal/library/crypto"
	"business/internal/library/gmail"
	"business/internal/library/gmailService"
	"business/internal/library/logger"
	macinfra "business/internal/mailaccountconnection/infrastructure"
	mfapp "business/internal/mailfetch/application"
	mfinfra "business/internal/mailfetch/infrastructure"

	"go.uber.org/dig"
	"gorm.io/gorm"
)

// ProvideMailFetchDependencies registers manualmailfetch stage dependencies.
func ProvideMailFetchDependencies(container *dig.Container) {
	_ = container.Provide(func(db *gorm.DB, log *logger.Logger) *mfinfra.GormEmailRepositoryAdapter {
		return mfinfra.NewGormEmailRepositoryAdapter(db, log)
	})

	_ = container.Provide(func(repo *macinfra.Repository, log *logger.Logger) *mfinfra.MailAccountConnectionReaderAdapter {
		return mfinfra.NewMailAccountConnectionReaderAdapter(repo, log)
	})

	_ = container.Provide(func(
		repo *macinfra.Repository,
		vault *crypto.Vault,
		oauthConfig *gmailService.OAuthConfigLoader,
		gmailServiceClient *gmailService.Client,
		gmailClient *gmail.Client,
		log *logger.Logger,
	) *mfinfra.GmailSessionBuilder {
		return mfinfra.NewGmailSessionBuilder(repo, vault, oauthConfig, gmailServiceClient, gmailClient, log)
	})

	_ = container.Provide(func(builder *mfinfra.GmailSessionBuilder, log *logger.Logger) *mfinfra.DefaultMailFetcherFactory {
		return mfinfra.NewDefaultMailFetcherFactory(builder, log)
	})

	_ = container.Provide(func(
		connectionRepo *mfinfra.MailAccountConnectionReaderAdapter,
		fetcherFactory *mfinfra.DefaultMailFetcherFactory,
		emailRepo *mfinfra.GormEmailRepositoryAdapter,
		log *logger.Logger,
	) mfapp.UseCase {
		return mfapp.NewUseCase(connectionRepo, fetcherFactory, emailRepo, log)
	})
}
