package infrastructure

import (
	cd "business/internal/common/domain"
	gmaillib "business/internal/library/gmail"
	"business/internal/library/logger"
	mfdomain "business/internal/mailfetch/domain"
	"context"
	"errors"
	"fmt"
	"strings"
)

type gmailClientBuilder interface {
	Build(ctx context.Context, connectionID, userID uint) (gmailMessageClient, error)
}

// GmailMailFetcherAdapter fetches Gmail messages for a single mail-account connection.
type GmailMailFetcherAdapter struct {
	conn    mfdomain.ConnectionRef
	builder gmailClientBuilder
	log     logger.Interface
}

// NewGmailMailFetcherAdapter creates a Gmail-backed mail fetcher.
func NewGmailMailFetcherAdapter(
	conn mfdomain.ConnectionRef,
	builder gmailClientBuilder,
	log logger.Interface,
) *GmailMailFetcherAdapter {
	if log == nil {
		log = logger.NewNop()
	}
	return &GmailMailFetcherAdapter{
		conn:    conn,
		builder: builder,
		log:     log.With(logger.Component("manual_mail_fetch_gmail_fetcher")),
	}
}

// Fetch loads message details for the configured connection and filters them by the requested period.
func (f *GmailMailFetcherAdapter) Fetch(ctx context.Context, cond mfdomain.FetchCondition) ([]cd.FetchedEmailDTO, []mfdomain.MessageFailure, error) {
	client, err := f.builder.Build(ctx, f.conn.ConnectionID, f.conn.UserID)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", mfdomain.ErrProviderSessionBuildFailed, err)
	}

	messageIDs, err := client.GetMessagesByLabelName(ctx, cond.LabelName, cond.Since)
	if err != nil {
		if errors.Is(err, gmaillib.ErrLabelNotFound) {
			return nil, nil, fmt.Errorf("%w: %s", mfdomain.ErrProviderLabelNotFound, cond.LabelName)
		}
		return nil, nil, fmt.Errorf("%w: %v", mfdomain.ErrProviderListFailed, err)
	}

	fetched := make([]cd.FetchedEmailDTO, 0, len(messageIDs))
	failures := make([]mfdomain.MessageFailure, 0)

	for _, messageID := range messageIDs {
		dto, detailErr := client.GetGmailDetail(ctx, messageID)
		if detailErr != nil {
			failures = append(failures, mfdomain.MessageFailure{
				ExternalMessageID: messageID,
				Stage:             mfdomain.FailureStageFetchDetail,
				Code:              mfdomain.FailureCodeFetchDetailFailed,
			})
			continue
		}

		dto.ID = strings.TrimSpace(dto.ID)
		if dto.ID == "" || dto.Date.IsZero() {
			failures = append(failures, mfdomain.MessageFailure{
				ExternalMessageID: fallbackExternalMessageID(dto.ID, messageID),
				Stage:             mfdomain.FailureStageNormalize,
				Code:              mfdomain.FailureCodeInvalidFetchedEmail,
			})
			continue
		}

		if dto.Date.Before(cond.Since) || !dto.Date.Before(cond.Until) {
			continue
		}

		fetched = append(fetched, dto)
	}

	return fetched, failures, nil
}

func fallbackExternalMessageID(preferred string, fallback string) string {
	if strings.TrimSpace(preferred) != "" {
		return strings.TrimSpace(preferred)
	}
	return strings.TrimSpace(fallback)
}
