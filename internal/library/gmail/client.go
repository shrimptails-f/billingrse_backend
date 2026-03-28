package gmail

import (
	cd "business/internal/common/domain"
	"business/internal/library/logger"
	"business/internal/library/ratelimit"
	"business/internal/library/retry"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"time"

	htmlcharset "golang.org/x/net/html/charset"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/googleapi"
)

var (
	// ErrLabelNotFound is returned when the requested Gmail label does not exist.
	ErrLabelNotFound = errors.New("gmail label not found")
)

type Client struct {
	svc     *gmail.Service
	limiter ratelimit.Limiter
	log     logger.Interface
}

func New(limiter ratelimit.Limiter, log logger.Interface) *Client {
	if log == nil {
		log = logger.NewNop()
	}

	return &Client{
		limiter: limiter,
		log:     log.With(logger.Component("gmail_client")),
	}
}

func (c *Client) SetClient(svc *gmail.Service) *Client {
	return &Client{
		svc:     svc,
		limiter: c.limiter,
		log:     c.log,
	}
}

func (c *Client) GetMessagesByLabelName(ctx context.Context, labelName string, startDate time.Time) ([]string, error) {
	if ctx == nil {
		return nil, logger.ErrNilContext
	}

	reqLog := c.log
	if withContext, err := c.log.WithContext(ctx); err == nil {
		reqLog = withContext
	}

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
		reqLog.Error("external_api_failed",
			logger.String("provider", "gmail"),
			logger.String("operation", "list_labels"),
			logger.Err(err),
		)
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
		return nil, fmt.Errorf("%w: %s", ErrLabelNotFound, labelName)
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
			reqLog.Error("external_api_failed",
				logger.String("provider", "gmail"),
				logger.String("operation", "list_messages_by_label"),
				logger.Err(err),
			)
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

	reqLog.Info("external_api_succeeded",
		logger.String("provider", "gmail"),
		logger.String("operation", "list_messages_by_label"),
		logger.Int("message_count", len(messageIds)),
	)

	return messageIds, nil
}

func (c *Client) GetGmailDetail(ctx context.Context, id string) (cd.FetchedEmailDTO, error) {
	if ctx == nil {
		return cd.FetchedEmailDTO{}, logger.ErrNilContext
	}

	reqLog := c.log
	if withContext, err := c.log.WithContext(ctx); err == nil {
		reqLog = withContext
	}

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
		reqLog.Error("external_api_failed",
			logger.String("provider", "gmail"),
			logger.String("operation", "get_message"),
			logger.String("gmail_message_id", id),
			logger.Err(err),
		)
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

	reqLog.Info("external_api_succeeded",
		logger.String("provider", "gmail"),
		logger.String("operation", "get_message"),
		logger.String("gmail_message_id", id),
	)

	return msg, nil
}

func getHeader(headers []*gmail.MessagePartHeader, name string) string {
	for _, h := range headers {
		if strings.EqualFold(h.Name, name) {
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
	if payload == nil {
		return ""
	}

	if (payload.MimeType == "text/plain" || payload.MimeType == "text/html") &&
		payload.Body != nil &&
		payload.Body.Data != "" {
		decoded, err := decodeMessagePartText(payload)
		if err == nil {
			return decoded
		}
	}
	for _, part := range payload.Parts {
		if body := extractBody(part); body != "" {
			return body
		}
	}
	return ""
}

func decodeMessagePartText(part *gmail.MessagePart) (string, error) {
	decoded, err := decodeGmailBodyData(part.Body.Data)
	if err != nil {
		return "", err
	}

	decoded, err = decodeTransferEncoding(decoded, getHeader(part.Headers, "Content-Transfer-Encoding"))
	if err != nil {
		return "", err
	}

	text, err := decodeBodyCharset(decoded, getHeader(part.Headers, "Content-Type"))
	if err != nil {
		text = string(decoded)
	}

	return normalizeMailText(text), nil
}

func decodeGmailBodyData(raw string) ([]byte, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	if decoded, err := base64.RawURLEncoding.DecodeString(trimmed); err == nil {
		return decoded, nil
	}
	if decoded, err := base64.URLEncoding.DecodeString(trimmed); err == nil {
		return decoded, nil
	}

	if remainder := len(trimmed) % 4; remainder != 0 {
		trimmed += strings.Repeat("=", 4-remainder)
	}
	return base64.URLEncoding.DecodeString(trimmed)
}

func decodeTransferEncoding(body []byte, rawEncoding string) ([]byte, error) {
	encoding := strings.ToLower(strings.TrimSpace(rawEncoding))

	switch encoding {
	case "", "7bit", "8bit", "binary":
		return body, nil
	case "quoted-printable":
		return io.ReadAll(quotedprintable.NewReader(bytes.NewReader(body)))
	case "base64":
		compacted := strings.Map(func(r rune) rune {
			switch r {
			case ' ', '\t', '\r', '\n':
				return -1
			default:
				return r
			}
		}, string(body))
		return base64.StdEncoding.DecodeString(compacted)
	default:
		return body, nil
	}
}

func decodeBodyCharset(body []byte, rawContentType string) (string, error) {
	_, params, err := mime.ParseMediaType(strings.TrimSpace(rawContentType))
	if err != nil {
		return "", err
	}

	charsetName := strings.TrimSpace(params["charset"])
	if charsetName == "" || strings.EqualFold(charsetName, "utf-8") || strings.EqualFold(charsetName, "us-ascii") {
		return string(body), nil
	}

	reader, err := htmlcharset.NewReaderLabel(charsetName, bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	decoded, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func normalizeMailText(text string) string {
	if text == "" {
		return ""
	}

	normalized := strings.ToValidUTF8(text, " ")
	normalized = strings.ReplaceAll(normalized, "\x00", " ")
	return normalized
}

func (c *Client) execute(ctx context.Context, fn func(context.Context) error) error {
	if ctx == nil {
		return logger.ErrNilContext
	}
	if c.svc == nil {
		return fmt.Errorf("gmail service is not configured")
	}
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
