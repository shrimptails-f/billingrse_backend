package openai

import (
	"business/internal/library/logger"
	"business/internal/library/ratelimit"
	"business/internal/library/retry"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	openaisdk "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const (
	// ProviderName identifies this AI provider in persisted metadata.
	ProviderName = "openai"
	// DefaultModel is the chat model currently used by the OpenAI client.
	DefaultModel = "gpt-5-mini"
)

const (
	parsedEmailResponseSchemaName = "parsed_email_analysis_results"
)

// BuildParsedEmailPrompt builds the extraction prompt for one email body.
// The output contract is a billing header with nested line-items.
func BuildParsedEmailPrompt(subject, from string, receivedAt time.Time, body string) string {
	return fmt.Sprintf(`あなたはメール本文から請求関連の構造化情報を抽出するアシスタントです。

出力規約:
- JSONオブジェクトのみを返してください
- トップレベルのキーは parsedEmails のみを使用してください
- parsedEmails は配列にしてください
- parsedEmails の各要素は「請求ヘッダ」を表し、同一請求番号の複数商品は lineItems の子要素として返してください
- parsedEmails の各要素のキーは productNameRaw, productNameDisplay, vendorName, billingNumber, invoiceNumber, amount, currency, billingDate, paymentCycle, lineItems のみを使用してください
- lineItems は配列にしてください
- lineItems の各要素のキーは productNameRaw, productNameDisplay, amount, currency のみを使用してください
- 値が分からない場合は null を設定してください
- billingDate は RFC3339 形式の文字列、または null にしてください
- paymentCycle は one_time, recurring, null のいずれかにしてください
- amount は請求ヘッダの合計金額として数値で返してください
- lineItems.amount は明細行ごとの金額を数値で返してください
- 請求関連の情報が読み取れない場合は {"parsedEmails": []} を返してください
- 推測で値を補完しないでください

subject: %s
from: %s
receivedAt: %s
body:
%s
`, strings.TrimSpace(normalizePromptText(subject)), strings.TrimSpace(normalizePromptText(from)), receivedAt.UTC().Format(time.RFC3339), strings.TrimSpace(normalizePromptText(body)))
}

type Client struct {
	sdk     *openaisdk.Client
	limiter ratelimit.Limiter
	log     logger.Interface
}

func New(apiKey string, limiter ratelimit.Limiter, log logger.Interface) *Client {
	if log == nil {
		log = logger.NewNop()
	}
	log = log.With(logger.Component("openai_client"))

	client := openaisdk.NewClient(
		option.WithAPIKey(apiKey),
		// Keep retries in this package so every outbound attempt is gated by the limiter.
		option.WithMaxRetries(0),
	)
	return &Client{
		sdk:     &client,
		limiter: limiter,
		log:     log,
	}
}

// Chat executes a raw chat completion request and returns the assistant content as-is.
func (c *Client) Chat(ctx context.Context, prompt string) (string, error) {
	if ctx == nil {
		return "", logger.ErrNilContext
	}
	normalizedPrompt := normalizePromptText(prompt)
	promptWasNormalized := normalizedPrompt != prompt

	reqLog := c.log
	if withContext, err := c.log.WithContext(ctx); err == nil {
		reqLog = withContext
	}

	const (
		provider  = ProviderName
		operation = "chat_completion"
	)
	model := DefaultModel

	var resp *openaisdk.ChatCompletion
	err := retry.DoWithCondition(ctx, retry.DefaultBackoff, shouldRetryOpenAIError, func(ctx context.Context) error {
		if c.limiter == nil {
			return errors.New("openai rate limiter is not configured")
		}
		if err := c.limiter.Wait(ctx); err != nil {
			return err
		}

		r, err := c.sdk.Chat.Completions.New(ctx, buildChatCompletionParams(normalizedPrompt))
		if err != nil {
			return err
		}
		resp = r
		return nil
	})
	if err != nil {
		reqLog.Error("external_api_failed",
			logger.String("provider", provider),
			logger.String("operation", operation),
			logger.String("model", model),
			logger.Int("prompt_bytes", len(normalizedPrompt)),
			logger.Bool("prompt_utf8_valid", utf8.ValidString(normalizedPrompt)),
			logger.Bool("prompt_sanitized", promptWasNormalized),
			logger.Err(err),
		)
		return "", err
	}

	if resp == nil || len(resp.Choices) == 0 {
		err := errors.New("openai returned no choices")
		reqLog.Error("external_api_failed",
			logger.String("provider", provider),
			logger.String("operation", operation),
			logger.String("model", model),
			logger.String("reason", "empty_choices"),
			logger.Err(err),
		)
		return "", err
	}

	raw := strings.TrimSpace(resp.Choices[0].Message.Content)
	if raw == "" {
		err := errors.New("openai returned empty message content")
		reqLog.Error("external_api_failed",
			logger.String("provider", provider),
			logger.String("operation", operation),
			logger.String("model", model),
			logger.String("reason", "empty_content"),
			logger.Err(err),
		)
		return "", err
	}

	reqLog.Info("external_api_succeeded",
		logger.String("provider", provider),
		logger.String("operation", operation),
		logger.String("model", model),
		logger.Int("response_bytes", len(raw)),
	)

	return raw, nil
}

func buildChatCompletionParams(prompt string) openaisdk.ChatCompletionNewParams {
	return openaisdk.ChatCompletionNewParams{
		Model: DefaultModel,
		Messages: []openaisdk.ChatCompletionMessageParamUnion{
			openaisdk.UserMessage(prompt),
		},
		ResponseFormat: openaisdk.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openaisdk.ResponseFormatJSONSchemaParam{
				JSONSchema: openaisdk.ResponseFormatJSONSchemaJSONSchemaParam{
					Name:        parsedEmailResponseSchemaName,
					Description: openaisdk.String("Structured billing-related information extracted from a single email."),
					Strict:      openaisdk.Bool(true),
					Schema:      parsedEmailResponseSchema(),
				},
			},
		},
	}
}

func parsedEmailResponseSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"parsedEmails": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"properties": map[string]any{
						"productNameRaw":     nullableStringSchema("Representative full product/service name for this billing header, or null when unknown."),
						"productNameDisplay": nullableStringSchema("Representative short display name for this billing header, or null when unknown."),
						"vendorName":         nullableStringSchema("Candidate vendor name extracted from the email, or null when unknown."),
						"billingNumber":      nullableStringSchema("Billing number extracted from the email. Multiple products under the same number must be nested in lineItems."),
						"invoiceNumber":      nullableStringSchema("Qualified invoice number extracted from the email, or null when unknown."),
						"amount":             nullableNumberSchema("Billing total amount extracted from the email, or null when unknown."),
						"currency":           nullableStringSchema("ISO 4217 currency code, or null when unknown."),
						"billingDate":        nullableStringSchema("RFC3339 billing date string, or null when unknown."),
						"paymentCycle": map[string]any{
							"type":        []string{"string", "null"},
							"description": "Billing cycle. Use one_time, recurring.",
							"enum":        []any{"one_time", "recurring"},
						},
						"lineItems": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type":                 "object",
								"additionalProperties": false,
								"properties": map[string]any{
									"productNameRaw":     nullableStringSchema("Full product/service name for one billing line item, or null when unknown."),
									"productNameDisplay": nullableStringSchema("Short display name for one billing line item, or null when unknown."),
									"amount":             nullableNumberSchema("Line-item amount, or null when unknown."),
									"currency":           nullableStringSchema("Line-item currency in ISO 4217, or null when unknown."),
								},
								"required": []string{
									"productNameRaw",
									"productNameDisplay",
									"amount",
									"currency",
								},
							},
						},
					},
					"required": []string{
						"productNameRaw",
						"productNameDisplay",
						"vendorName",
						"billingNumber",
						"invoiceNumber",
						"amount",
						"currency",
						"billingDate",
						"paymentCycle",
						"lineItems",
					},
				},
			},
		},
		"required": []string{"parsedEmails"},
	}
}

func nullableStringSchema(description string) map[string]any {
	return map[string]any{
		"type":        []string{"string", "null"},
		"description": description,
	}
}

func nullableNumberSchema(description string) map[string]any {
	return map[string]any{
		"type":        []string{"number", "null"},
		"description": description,
	}
}

func normalizePromptText(text string) string {
	if text == "" {
		return ""
	}

	normalized := strings.ToValidUTF8(text, " ")
	normalized = strings.ReplaceAll(normalized, "\x00", " ")
	return normalized
}

// shouldRetryOpenAIError determines if an OpenAI API error should be retried.
// Returns true only for 429 (rate limit) and 5xx (server) errors.
func shouldRetryOpenAIError(err error) bool {
	var apiErr *openaisdk.Error
	if errors.As(err, &apiErr) {
		code := apiErr.StatusCode
		return code == 429 || (code >= 500 && code < 600)
	}
	// Non-openai errors are not retried
	return false
}
