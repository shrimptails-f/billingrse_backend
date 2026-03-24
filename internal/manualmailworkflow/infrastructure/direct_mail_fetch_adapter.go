package infrastructure

import (
	mfapp "business/internal/mailfetch/application"
	mfdomain "business/internal/mailfetch/domain"
	manualapp "business/internal/manualmailworkflow/application"
	"context"
	"errors"
)

// DirectManualMailFetchAdapter は mailfetch usecase を直接呼び出す。
type DirectManualMailFetchAdapter struct {
	usecase mfapp.UseCase
}

// NewDirectManualMailFetchAdapter は direct な mailfetch adapter を生成する。
func NewDirectManualMailFetchAdapter(usecase mfapp.UseCase) *DirectManualMailFetchAdapter {
	return &DirectManualMailFetchAdapter{usecase: usecase}
}

// Execute は mailfetch stage を実行し、workflow 側の型へ変換する。
func (a *DirectManualMailFetchAdapter) Execute(ctx context.Context, cmd manualapp.FetchCommand) (manualapp.FetchResult, error) {
	if a.usecase == nil {
		return manualapp.FetchResult{}, errors.New("mailfetch usecase is not configured")
	}

	result, err := a.usecase.Execute(ctx, mfapp.Command{
		UserID:       cmd.UserID,
		ConnectionID: cmd.ConnectionID,
		Condition: mfdomain.FetchCondition{
			LabelName: cmd.Condition.LabelName,
			Since:     cmd.Condition.Since,
			Until:     cmd.Condition.Until,
		},
	})
	if err != nil {
		return manualapp.FetchResult{}, err
	}

	createdEmailIDs := make([]uint, 0, len(result.CreatedEmailIDs))
	createdEmailIDs = append(createdEmailIDs, result.CreatedEmailIDs...)

	createdEmails := make([]manualapp.CreatedEmail, 0, len(result.CreatedEmails))
	for _, createdEmail := range result.CreatedEmails {
		createdEmails = append(createdEmails, manualapp.CreatedEmail{
			EmailID:           createdEmail.EmailID,
			ExternalMessageID: createdEmail.ExternalMessageID,
			Subject:           createdEmail.Subject,
			From:              createdEmail.From,
			To:                append([]string{}, createdEmail.To...),
			ReceivedAt:        createdEmail.Date,
			Body:              createdEmail.Body,
			BodyDigest:        createdEmail.BodyDigest,
		})
	}

	existingEmailIDs := make([]uint, 0, len(result.ExistingEmailIDs))
	existingEmailIDs = append(existingEmailIDs, result.ExistingEmailIDs...)

	failures := make([]manualapp.FetchFailure, 0, len(result.Failures))
	for _, failure := range result.Failures {
		failures = append(failures, manualapp.FetchFailure{
			ExternalMessageID: failure.ExternalMessageID,
			Stage:             failure.Stage,
			Code:              failure.Code,
		})
	}

	return manualapp.FetchResult{
		Provider:            result.Provider,
		AccountIdentifier:   result.AccountIdentifier,
		MatchedMessageCount: result.MatchedMessageCount,
		CreatedEmailIDs:     createdEmailIDs,
		CreatedEmails:       createdEmails,
		ExistingEmailIDs:    existingEmailIDs,
		Failures:            failures,
	}, nil
}
