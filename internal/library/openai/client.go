package openai

import (
	"business/internal/library/logger"
	"business/internal/library/ratelimit"
	"business/internal/library/retry"
	"context"
	"errors"
	"strings"

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

		r, err := c.sdk.Chat.Completions.New(ctx, buildChatCompletionParams(prompt))
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
						"productNameRaw":     nullableStringSchema("Full product or service name as it appears in the email, or null when unknown."),
						"productNameDisplay": nullableStringSchema("Short display name. Use only the product name, or the set name for bundled products, or null when unknown."),
						"vendorName":         nullableStringSchema("Candidate vendor name extracted from the email, or null when unknown."),
						"billingNumber":      nullableStringSchema("Billing number extracted from the email, or null when unknown."),
						"invoiceNumber":      nullableStringSchema("Qualified invoice number extracted from the email, or null when unknown."),
						"amount":             nullableNumberSchema("Billing amount extracted from the email, or null when unknown."),
						"currency":           nullableStringSchema("ISO 4217 currency code, or null when unknown."),
						"billingDate":        nullableStringSchema("RFC3339 billing date string, or null when unknown."),
						"paymentCycle": map[string]any{
							"type":        []string{"string", "null"},
							"description": "Billing cycle. Use one_time, recurring, or null when unknown.",
							"enum":        []any{"one_time", "recurring", nil},
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
