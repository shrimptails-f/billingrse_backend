package gmail

import (
	cd "business/internal/common/domain"
	"context"
	"time"

	"google.golang.org/api/gmail/v1"
)

type ClientInterface interface {
	GetMessagesByLabelName(ctx context.Context, labelName string, startDate time.Time) ([]string, error)
	GetGmailDetail(ctx context.Context, id string) (cd.FetchedEmailDTO, error)
	SetClient(svc *gmail.Service) *Client
}
