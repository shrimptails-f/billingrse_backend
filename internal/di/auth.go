package di

import (
	"business/internal/app/middleware"
	"business/internal/auth/application"
	"business/internal/auth/infrastructure"
	"business/internal/auth/infrastructure/mailer"
	"business/internal/library/logger"
	"business/internal/library/mysql"
	"business/internal/library/oswrapper"
	"business/internal/library/sendMailer"
	"business/internal/library/timewrapper"

	"go.uber.org/dig"
)

// ProvideAuthDependencies registers authentication-related dependencies.
func ProvideAuthDependencies(container *dig.Container) {
	_ = container.Provide(func(conn *mysql.MySQL, log logger.Interface) *infrastructure.Repository {
		return infrastructure.NewRepository(conn.DB, log)
	})

	_ = container.Provide(func(osw *oswrapper.OsWrapper) sendMailer.Client {
		return sendMailer.New(osw)
	})

	_ = container.Provide(func(client sendMailer.Client, log logger.Interface) application.VerificationEmailSender {
		return mailer.NewSMTPVerificationEmailSender(client, log)
	})

	_ = container.Provide(func(
		repo *infrastructure.Repository,
		osw *oswrapper.OsWrapper,
		mailer application.VerificationEmailSender,
		clock timewrapper.ClockInterface,
	) application.AuthUseCaseInterface {
		return application.NewAuthUseCase(repo, osw, mailer, clock)
	})

	_ = container.Provide(func(osw *oswrapper.OsWrapper, repo *infrastructure.Repository, log logger.Interface) *middleware.AuthMiddleware {
		return middleware.NewAuthMiddleware(osw, repo, log)
	})
}
