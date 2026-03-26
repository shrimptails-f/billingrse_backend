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
	"sync"
)

const maxGmailDetailFetchWorkers = 8

type gmailClientBuilder interface {
	Build(ctx context.Context, connectionID, userID uint) (gmailMessageClient, error)
}

type gmailDetailFetchResult struct {
	messageID string
	dto       cd.FetchedEmailDTO
	err       error
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
	detailResults := fetchGmailDetailsConcurrently(ctx, client, messageIDs)

	for _, detailResult := range detailResults {
		if detailResult.err != nil {
			failures = append(failures, mfdomain.MessageFailure{
				ExternalMessageID: detailResult.messageID,
				Stage:             mfdomain.FailureStageFetchDetail,
				Code:              mfdomain.FailureCodeFetchDetailFailed,
				Message:           fetchDetailFailureMessage(detailResult.messageID),
			})
			continue
		}

		dto := detailResult.dto
		dto.ID = strings.TrimSpace(dto.ID)
		if dto.ID == "" || dto.Date.IsZero() {
			failures = append(failures, mfdomain.MessageFailure{
				ExternalMessageID: fallbackExternalMessageID(dto.ID, detailResult.messageID),
				Stage:             mfdomain.FailureStageNormalize,
				Code:              mfdomain.FailureCodeInvalidFetchedEmail,
				Message:           normalizeFetchedEmailFailureMessage(fallbackExternalMessageID(dto.ID, detailResult.messageID), dto.Date.IsZero()),
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

func fetchGmailDetailsConcurrently(ctx context.Context, client gmailMessageClient, messageIDs []string) []gmailDetailFetchResult {
	if len(messageIDs) == 0 {
		return nil
	}

	results := make([]gmailDetailFetchResult, len(messageIDs))
	workerCount := len(messageIDs)
	if workerCount > maxGmailDetailFetchWorkers {
		workerCount = maxGmailDetailFetchWorkers
	}

	jobs := make(chan int)
	var wg sync.WaitGroup
	wg.Add(workerCount)
	for range workerCount {
		go func() {
			defer wg.Done()
			for idx := range jobs {
				messageID := messageIDs[idx]
				dto, err := client.GetGmailDetail(ctx, messageID)
				results[idx] = gmailDetailFetchResult{
					messageID: messageID,
					dto:       dto,
					err:       err,
				}
			}
		}()
	}

	for idx := range messageIDs {
		jobs <- idx
	}
	close(jobs)
	wg.Wait()

	return results
}

func fallbackExternalMessageID(preferred string, fallback string) string {
	if strings.TrimSpace(preferred) != "" {
		return strings.TrimSpace(preferred)
	}
	return strings.TrimSpace(fallback)
}

func fetchDetailFailureMessage(externalMessageID string) string {
	externalMessageID = strings.TrimSpace(externalMessageID)
	if externalMessageID == "" {
		externalMessageID = "unknown"
	}
	return "Gmail本文の取得に失敗しました。メールID=" + externalMessageID
}

func normalizeFetchedEmailFailureMessage(externalMessageID string, missingDate bool) string {
	externalMessageID = strings.TrimSpace(externalMessageID)
	if externalMessageID == "" {
		externalMessageID = "unknown"
	}
	if missingDate {
		return "取得メール(" + externalMessageID + ")の受信日時が不正でした。"
	}
	return "取得メール(" + externalMessageID + ")のIDが不正でした。"
}
