package di

import (
	"business/internal/library/logger"
	"business/internal/library/openai"
	"business/internal/library/timewrapper"
	maapp "business/internal/mailanalysis/application"
	mainfra "business/internal/mailanalysis/infrastructure"

	"go.uber.org/dig"
	"gorm.io/gorm"
)

// ProvideMailAnalysisDependencies registers mailanalysis stage dependencies.
func ProvideMailAnalysisDependencies(container *dig.Container) {
	_ = container.Provide(func(db *gorm.DB, log *logger.Logger) *mainfra.GormParsedEmailRepositoryAdapter {
		return mainfra.NewGormParsedEmailRepositoryAdapter(db, log)
	})

	_ = container.Provide(func(oa *openai.Client, log *logger.Logger) *mainfra.OpenAIAnalyzerAdapter {
		return mainfra.NewOpenAIAnalyzerAdapter(oa, log)
	})

	_ = container.Provide(func(analyzer *mainfra.OpenAIAnalyzerAdapter, log *logger.Logger) *mainfra.DefaultAnalyzerFactory {
		return mainfra.NewDefaultAnalyzerFactory(analyzer, log)
	})

	_ = container.Provide(func(
		clock *timewrapper.Clock,
		factory *mainfra.DefaultAnalyzerFactory,
		repository *mainfra.GormParsedEmailRepositoryAdapter,
		log *logger.Logger,
	) maapp.UseCase {
		return maapp.NewUseCase(clock, factory, repository, log)
	})
}
