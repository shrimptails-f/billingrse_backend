package infrastructure

import (
	maapp "business/internal/mailanalysis/application"
	manualapp "business/internal/manualmailworkflow/application"
	"context"
	"errors"
)

// DirectMailAnalysisAdapter は mailanalysis usecase を直接呼び出す。
type DirectMailAnalysisAdapter struct {
	usecase maapp.UseCase
}

// NewDirectMailAnalysisAdapter は direct な mailanalysis adapter を生成する。
func NewDirectMailAnalysisAdapter(usecase maapp.UseCase) *DirectMailAnalysisAdapter {
	return &DirectMailAnalysisAdapter{usecase: usecase}
}

// Execute は mailanalysis stage を実行し、workflow 側の型へ変換する。
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
			BodyDigest:        email.BodyDigest,
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

	parsedEmails := make([]manualapp.ParsedEmail, 0, len(result.ParsedEmails))
	for _, parsedEmail := range result.ParsedEmails {
		parsedEmails = append(parsedEmails, manualapp.ParsedEmail{
			ParsedEmailID:     parsedEmail.ParsedEmailID,
			EmailID:           parsedEmail.EmailID,
			ExternalMessageID: parsedEmail.ExternalMessageID,
			Subject:           parsedEmail.Subject,
			From:              parsedEmail.From,
			To:                append([]string{}, parsedEmail.To...),
			BodyDigest:        parsedEmail.BodyDigest,
			Data:              parsedEmail.ParsedEmail,
		})
	}

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
		ParsedEmails:       parsedEmails,
		AnalyzedEmailCount: result.AnalyzedEmailCount,
		ParsedEmailCount:   result.ParsedEmailCount,
		Failures:           failures,
	}, nil
}
