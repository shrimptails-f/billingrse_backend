package di

import (
	"business/internal/app/middleware"
	"business/internal/auth/application"
	"business/internal/auth/infrastructure"
	"business/internal/auth/infrastructure/mailer"
	"business/internal/library/logger"
	"business/internal/library/mysql"
	"business/internal/library/oswrapper"
	"business/internal/library/sendMailerClient"
	"log"
	"strconv"
	"strings"
	"time"

	"go.uber.org/dig"
)

// ProvideAuthDependencies registers authentication-related dependencies.
func ProvideAuthDependencies(container *dig.Container) {
	_ = container.Provide(func(conn *mysql.MySQL, log logger.Interface) *infrastructure.Repository {
		return infrastructure.NewRepository(conn.DB, log)
	})

	_ = container.Provide(func(osw *oswrapper.OsWrapper) sendMailerClient.Client {
		return sendMailerClient.New(osw)
	})

	_ = container.Provide(func(client sendMailerClient.Client, log logger.Interface) application.VerificationEmailSender {
		return mailer.NewSMTPVerificationEmailSender(client, log)
	})

	_ = container.Provide(func(
		repo *infrastructure.Repository,
		osw *oswrapper.OsWrapper,
		mailer application.VerificationEmailSender,
	) application.AuthUseCaseInterface {
		return application.NewAuthUseCase(repo, osw, parseTokenTTL(osw), mailer, nil)
	})

	_ = container.Provide(func(osw *oswrapper.OsWrapper, repo *infrastructure.Repository) *middleware.AuthMiddleware {
		return middleware.NewAuthMiddleware(osw, repo)
	})
}

func parseTokenTTL(osw oswrapper.OsWapperInterface) time.Duration {
	const defaultTTL = 24 * time.Hour

	expiresInRaw, err := osw.GetEnv("JWT_EXPIRES_IN")
	if err != nil {
		return defaultTTL
	}

	expiresIn := strings.TrimSpace(expiresInRaw)

	if expiresIn == "" {
		return defaultTTL
	}

	// Try duration format first (e.g., "30m", "2h")
	if duration, err := time.ParseDuration(expiresIn); err == nil {
		return duration
	}

	// Try as seconds (e.g., "3600")
	if seconds, err := strconv.ParseInt(expiresIn, 10, 64); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}

	// Invalid format, warn and fallback
	log.Printf("[WARN] Invalid JWT_EXPIRES_IN value '%s', falling back to %v", expiresIn, defaultTTL)
	return defaultTTL
}
