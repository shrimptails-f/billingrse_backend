package di

import (
	"business/internal/app/middleware"
	"business/internal/auth/application"
	"business/internal/auth/infrastructure"
	"business/internal/auth/infrastructure/mailer"
	"business/internal/library/crypto"
	"business/internal/library/logger"
	"business/internal/library/mysql"
	"business/internal/library/oswrapper"
	"business/internal/library/sendMailer"
	"business/internal/library/timewrapper"
	"context"
	"fmt"
	"strings"

	"go.uber.org/dig"
)

// ProvideAuthDependencies registers authentication-related dependencies.
func ProvideAuthDependencies(container *dig.Container) {
	_ = container.Provide(func(conn *mysql.MySQL, log *logger.Logger) *infrastructure.Repository {
		return infrastructure.NewRepository(conn.DB, log)
	})

	_ = container.Provide(func(osw *oswrapper.OsWrapper, log *logger.Logger) (*mailer.SMTPVerificationEmailSender, error) {
		app, err := osw.GetEnv("APP")
		if err != nil {
			return nil, fmt.Errorf("failed to get env APP: %w", err)
		}

		var client sendMailer.Client
		environment := strings.TrimSpace(app)
		if environment == "local" || environment == "ci" {
			client = sendMailer.New(osw)
		} else {
			client, err = sendMailer.NewSES(context.Background(), osw)
			if err != nil {
				return nil, err
			}
		}

		return mailer.NewSMTPVerificationEmailSender(client, log), nil
	})

	_ = container.Provide(func(
		repo *infrastructure.Repository,
		osw *oswrapper.OsWrapper,
		mailer *mailer.SMTPVerificationEmailSender,
		clock *timewrapper.Clock,
		vault *crypto.Vault,
	) *application.AuthUseCase {
		return application.NewAuthUseCase(repo, osw, mailer, clock, vault)
	})

	_ = container.Provide(func(osw *oswrapper.OsWrapper, repo *infrastructure.Repository, log *logger.Logger) *middleware.AuthMiddleware {
		return middleware.NewAuthMiddleware(osw, repo, log)
	})
}
