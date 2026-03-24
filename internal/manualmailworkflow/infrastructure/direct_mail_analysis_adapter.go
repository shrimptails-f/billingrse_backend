package infrastructure

import (
	maapp "business/internal/mailanalysis/application"
	manualapp "business/internal/manualmailworkflow/application"
	"context"
	"errors"
)

// DirectMailAnalysisAdapter invokes the mailanalysis use case directly.
type DirectMailAnalysisAdapter struct {
	usecase maapp.UseCase
}

// NewDirectMailAnalysisAdapter creates a direct mailanalysis adapter.
func NewDirectMailAnalysisAdapter(usecase maapp.UseCase) *DirectMailAnalysisAdapter {
	return &DirectMailAnalysisAdapter{usecase: usecase}
}

// Execute runs the mailanalysis stage and converts its result into workflow-owned types.
func (a *DirectMailAnalysisAdapter) Execute(ctx context.Context, cmd manualapp.AnalyzeCommand) (manualapp.AnalyzeResult, error) {
	if a.usecase == nil {
		return manualapp.AnalyzeResult{}, errors.New("mailanalysis usecase is not configured")
	}

	emails := make([]maapp.EmailForAnalysisTarget, 0, len(cmd.Emails))
	for _, email := range cmd.Emails {
		emails = append(emails, maapp.EmailForAnalysisTarget{
			EmailID:           email.EmailID,
			ExternalMessageID: email.ExternalMessageID,
			Subject:           email.Subject,
			From:              email.From,
			To:                append([]string{}, email.To...),
			ReceivedAt:        email.ReceivedAt,
			Body:              email.Body,
		})
	}

	result, err := a.usecase.Execute(ctx, maapp.Command{
		UserID: cmd.UserID,
		Emails: emails,
	})
	if err != nil {
		return manualapp.AnalyzeResult{}, err
	}

	parsedEmailIDs := make([]uint, 0, len(result.ParsedEmailIDs))
	parsedEmailIDs = append(parsedEmailIDs, result.ParsedEmailIDs...)

	failures := make([]manualapp.AnalysisFailure, 0, len(result.Failures))
	for _, failure := range result.Failures {
		failures = append(failures, manualapp.AnalysisFailure{
			EmailID:           failure.EmailID,
			ExternalMessageID: failure.ExternalMessageID,
			Stage:             failure.Stage,
			Code:              failure.Code,
		})
	}

	return manualapp.AnalyzeResult{
		ParsedEmailIDs:     parsedEmailIDs,
		AnalyzedEmailCount: result.AnalyzedEmailCount,
		ParsedEmailCount:   result.ParsedEmailCount,
		Failures:           failures,
	}, nil
}
