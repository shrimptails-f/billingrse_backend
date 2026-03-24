package infrastructure

import (
	"business/internal/library/logger"
	maapp "business/internal/mailanalysis/application"
	"context"
	"errors"
)

// DefaultAnalyzerFactory always resolves the current OpenAI analyzer implementation.
type DefaultAnalyzerFactory struct {
	analyzer *OpenAIAnalyzerAdapter
	log      logger.Interface
}

// NewDefaultAnalyzerFactory creates the default analyzer factory.
func NewDefaultAnalyzerFactory(analyzer *OpenAIAnalyzerAdapter, log logger.Interface) *DefaultAnalyzerFactory {
	if log == nil {
		log = logger.NewNop()
	}

	return &DefaultAnalyzerFactory{
		analyzer: analyzer,
		log:      log.With(logger.Component("email_analysis_analyzer_factory")),
	}
}

// Create returns the configured analyzer implementation.
func (f *DefaultAnalyzerFactory) Create(ctx context.Context, spec maapp.AnalyzerSpec) (maapp.Analyzer, error) {
	if ctx == nil {
		return nil, logger.ErrNilContext
	}
	if spec.UserID == 0 {
		return nil, errors.New("user_id is required")
	}
	if f.analyzer == nil {
		return nil, errors.New("openai analyzer is not configured")
	}
	return f.analyzer, nil
}
