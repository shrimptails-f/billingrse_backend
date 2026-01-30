package openai

import (
	cd "business/internal/common/domain"
	"business/internal/library/logger"
	"business/internal/library/ratelimit"
	"business/internal/library/retry"
	"context"
	"encoding/json"
	"errors"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type Client struct {
	sdk     *openai.Client
	limiter ratelimit.Limiter
	log     logger.Interface
}

func New(apiKey string, limiter ratelimit.Limiter, log logger.Interface) *Client {
	if log == nil {
		log = logger.NewNop()
	}
	log = log.With(logger.String("component", "openai_client"))

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
	)
	return &Client{
		sdk:     &client,
		limiter: limiter,
		log:     log,
	}
}

// func generateSchema[T any]() interface{} {
// 	reflector := jsonschema.Reflector{
// 		AllowAdditionalProperties: false,
// 		DoNotReference:            true,
// 	}
// 	var v T
// 	return reflector.Reflect(v)
// }

// type AnalysisResults struct {
// 	Items []cd.AnalysisResult `json:"results" jsonschema_description:"分析結果の配列"`
// }

func (c *Client) Chat(ctx context.Context, prompt string) ([]cd.ParsedEmail, error) {
	// schema := GenerateSchema[AnalysisResults]()

	// schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
	// 	Name:        "email_analysis_result",
	// 	Description: openai.String("メールの構造化分析結果"),
	// 	Schema:      schema,
	// 	Strict:      openai.Bool(true),
	// }

	var resp *openai.ChatCompletion
	err := retry.DoWithCondition(ctx, retry.DefaultBackoff, shouldRetryOpenAIError, func(ctx context.Context) error {
		if c.limiter == nil {
			return errors.New("openai rate limiter is not configured")
		}
		if err := c.limiter.Wait(ctx); err != nil {
			return err
		}

		r, err := c.sdk.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model: openai.ChatModelGPT4_1Mini,
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage(prompt),
			},
			// 低コストバージョンを使いたいので指定できない。
			// ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			// 	OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
			// 		JSONSchema: schemaParam,
			// 	},
			// },
		})
		if err != nil {
			return err
		}
		resp = r
		return nil
	})
	if err != nil {
		return nil, err
	}
	raw := resp.Choices[0].Message.Content

	var results []cd.ParsedEmail
	if err := json.Unmarshal([]byte(raw), &results); err != nil {
		c.log.Error("JSON→構造体変換失敗",
			logger.Err(err),
			logger.String("raw_content", raw))
		return nil, err
	}

	return results, nil
}

// shouldRetryOpenAIError determines if an OpenAI API error should be retried.
// Returns true only for 429 (rate limit) and 5xx (server) errors.
func shouldRetryOpenAIError(err error) bool {
	var apiErr *openai.Error
	if errors.As(err, &apiErr) {
		code := apiErr.StatusCode
		return code == 429 || (code >= 500 && code < 600)
	}
	// Non-openai errors are not retried
	return false
}
