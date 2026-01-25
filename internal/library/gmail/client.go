package gmail

import (
	cd "business/internal/common/domain"
	"business/internal/library/ratelimit"
	"business/internal/library/retry"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/googleapi"
)

type Client struct {
	svc     *gmail.Service
	limiter ratelimit.Limiter
}

func New(limiter ratelimit.Limiter) *Client {
	return &Client{
		limiter: limiter,
	}
}

func (c *Client) SetClient(svc *gmail.Service) *Client {
	return &Client{
		svc:     svc,
		limiter: c.limiter,
	}
}

func (c *Client) GetMessagesByLabelName(ctx context.Context, labelName string, startDate time.Time) ([]string, error) {
	user := "me"

	// ラベルID取得
	var labelResp *gmail.ListLabelsResponse
	err := c.execute(ctx, func(ctx context.Context) error {
		resp, err := c.svc.Users.Labels.List(user).Context(ctx).Do()
		if err != nil {
			return err
		}
		labelResp = resp
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("ラベル取得に失敗しました。: %v", err)
	}
	var labelID string
	for _, label := range labelResp.Labels {
		if label.Name == labelName {
			labelID = label.Id
			break
		}
	}
	if labelID == "" {
		return nil, fmt.Errorf("ラベル '%s' が見つかりませんでした", labelName)
	}

	// 検索条件（入力されたタイムゾーンに合わせて 0 時に揃え、UTC へ変換）
	truncated := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	query := fmt.Sprintf("after:%d", truncated.In(time.UTC).Unix())

	// ページングしながら取得
	var messageIds []string
	pageToken := ""

	for {
		req := c.svc.Users.Messages.List(user).Context(ctx).
			LabelIds(labelID).
			Q(query).
			MaxResults(100)
		if pageToken != "" {
			req.PageToken(pageToken)
		}

		var resp *gmail.ListMessagesResponse
		err := c.execute(ctx, func(ctx context.Context) error {
			result, err := req.Context(ctx).Do()
			if err != nil {
				return err
			}
			resp = result
			return nil
		})
		if err != nil {
			return nil, err
		}

		for _, m := range resp.Messages {
			messageIds = append(messageIds, m.Id)
		}
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken

	}

	return messageIds, nil
}

func (c *Client) GetGmailDetail(ctx context.Context, id string) (cd.FetchedEmailDTO, error) {
	user := "me"

	var full *gmail.Message
	err := c.execute(ctx, func(ctx context.Context) error {
		resp, err := c.svc.Users.Messages.Get(user, id).Format("full").Context(ctx).Do()
		if err != nil {
			return err
		}
		full = resp
		return nil
	})
	if err != nil {
		return cd.FetchedEmailDTO{}, fmt.Errorf("gメール取得処理でエラーが発生しました。 %v", err)
	}

	msg := cd.FetchedEmailDTO{
		ID:      full.Id,
		Subject: getHeader(full.Payload.Headers, "Subject"),
		From:    getHeader(full.Payload.Headers, "From"),
		To:      parseHeaderMulti(getHeader(full.Payload.Headers, "To")),
		Date:    parseDate(getHeader(full.Payload.Headers, "Date")),
		Body:    stripHTMLTags(extractBody(full.Payload)), // HTMLタグを削除する。
	}
	return msg, nil
}

func getHeader(headers []*gmail.MessagePartHeader, name string) string {
	for _, h := range headers {
		if h.Name == name {
			return h.Value
		}
	}
	return ""
}

func parseHeaderMulti(raw string) []string {
	if raw == "" {
		return nil
	}
	return strings.Split(raw, ",")
}

func parseDate(raw string) time.Time {
	t, err := mail.ParseDate(raw)
	if err != nil {
		return time.Time{}
	}
	return t
}

func extractBody(payload *gmail.MessagePart) string {
	if (payload.MimeType == "text/plain" || payload.MimeType == "text/html") &&
		payload.Body != nil &&
		payload.Body.Data != "" {

		decoded, err := base64.URLEncoding.DecodeString(payload.Body.Data)

		if err == nil {
			return string(decoded)
		}
	}
	for _, part := range payload.Parts {
		if body := extractBody(part); body != "" {
			return body
		}
	}
	return ""
}

func (c *Client) execute(ctx context.Context, fn func(context.Context) error) error {
	if c.limiter == nil {
		return fmt.Errorf("gmail rate limiter is not configured")
	}
	return retry.DoWithCondition(ctx, retry.DefaultBackoff, shouldRetryGmailError, func(ctx context.Context) error {
		if err := c.limiter.Wait(ctx); err != nil {
			return err
		}
		return fn(ctx)
	})
}

// shouldRetryGmailError determines if a Gmail API error should be retried.
// Returns true only for 429 (rate limit) and 5xx (server) errors.
func shouldRetryGmailError(err error) bool {
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		code := apiErr.Code
		return code == 429 || (code >= 500 && code < 600)
	}
	// Non-googleapi errors are not retried
	return false
}
