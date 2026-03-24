package infrastructure

import (
	commondomain "business/internal/common/domain"
	"business/internal/library/logger"
	maapp "business/internal/mailanalysis/application"
	madomain "business/internal/mailanalysis/domain"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const promptVersion = "emailanalysis_v1"

type openAIClient interface {
	Chat(ctx context.Context, prompt string) (string, error)
}

// OpenAIAnalyzerAdapter calls OpenAI and maps the raw JSON response into ParsedEmails.
type OpenAIAnalyzerAdapter struct {
	client openAIClient
	log    logger.Interface
}

// NewOpenAIAnalyzerAdapter creates an OpenAI-backed analyzer adapter.
func NewOpenAIAnalyzerAdapter(client openAIClient, log logger.Interface) *OpenAIAnalyzerAdapter {
	if log == nil {
		log = logger.NewNop()
	}

	return &OpenAIAnalyzerAdapter{
		client: client,
		log:    log.With(logger.Component("email_analysis_openai_analyzer")),
	}
}

// Analyze executes OpenAI analysis for a single email.
func (a *OpenAIAnalyzerAdapter) Analyze(ctx context.Context, email maapp.EmailForAnalysisTarget) (madomain.AnalysisOutput, error) {
	if ctx == nil {
		return madomain.AnalysisOutput{}, logger.ErrNilContext
	}
	if err := email.Validate(); err != nil {
		return madomain.AnalysisOutput{}, err
	}
	if a.client == nil {
		return madomain.AnalysisOutput{}, fmt.Errorf("openai client is not configured")
	}

	raw, err := a.client.Chat(ctx, buildPrompt(email))
	if err != nil {
		return madomain.AnalysisOutput{}, err
	}

	drafts, err := parseAnalysisResponse(raw)
	if err != nil {
		return madomain.AnalysisOutput{}, fmt.Errorf("%w: %v", madomain.ErrAnalysisResponseInvalid, err)
	}

	return madomain.AnalysisOutput{
		ParsedEmails:  drafts,
		PromptVersion: promptVersion,
	}.Normalize(), nil
}

type parsedEmailResponse struct {
	ProductNameRaw     *string  `json:"productNameRaw"`
	ProductNameDisplay *string  `json:"productNameDisplay"`
	VendorName         *string  `json:"vendorName"`
	BillingNumber      *string  `json:"billingNumber"`
	InvoiceNumber      *string  `json:"invoiceNumber"`
	Amount             *float64 `json:"amount"`
	Currency           *string  `json:"currency"`
	BillingDate        *string  `json:"billingDate"`
	PaymentCycle       *string  `json:"paymentCycle"`
}

type parsedEmailResponseEnvelope struct {
	ParsedEmails []parsedEmailResponse `json:"parsedEmails"`
}

func buildPrompt(email maapp.EmailForAnalysisTarget) string {
	return fmt.Sprintf(`あなたはメール本文から請求関連の構造化情報を抽出するアシスタントです。

出力規約:
- JSONオブジェクトのみを返してください
- トップレベルのキーは parsedEmails のみを使用してください
- parsedEmails は配列にしてください
- parsedEmails の各要素のキーは productNameRaw, productNameDisplay, vendorName, billingNumber, invoiceNumber, amount, currency, billingDate, paymentCycle のみを使用してください
- 値が分からない場合は null を設定してください
- billingDate は RFC3339 形式の文字列、または null にしてください
- paymentCycle は one_time, recurring, null のいずれかにしてください
- productNameRaw はメールに書かれている商品名/サービス名の全文、分からない場合は null にしてください
- productNameDisplay は表示用の短い商品名です。単品なら商品名だけ、セット商品ならセット名、短縮できなければ productNameRaw と同じで構いません。分からない場合は null にしてください
- amount は数値、複数請求があれば parsedEmails に複数要素を入れてください
- 請求関連の情報が読み取れない場合は {"parsedEmails": []} を返してください
- 推測で値を補完しないでください

subject: %s
from: %s
receivedAt: %s
body:
%s
`, email.Subject, email.From, email.ReceivedAt.UTC().Format(time.RFC3339), email.Body)
}

func parseAnalysisResponse(raw string) ([]commondomain.ParsedEmail, error) {
	payload := strings.TrimSpace(raw)
	if strings.TrimSpace(payload) == "" {
		return nil, fmt.Errorf("response body is empty")
	}

	var response parsedEmailResponseEnvelope
	if err := json.Unmarshal([]byte(payload), &response); err != nil {
		return nil, err
	}

	parsedEmails := make([]commondomain.ParsedEmail, 0, len(response.ParsedEmails))
	for _, item := range response.ParsedEmails {
		billingDate, err := parseBillingDate(item.BillingDate)
		if err != nil {
			return nil, err
		}

		parsedEmail := commondomain.ParsedEmail{
			ProductNameRaw:     item.ProductNameRaw,
			ProductNameDisplay: item.ProductNameDisplay,
			VendorName:         item.VendorName,
			BillingNumber:      item.BillingNumber,
			InvoiceNumber:      item.InvoiceNumber,
			Amount:             item.Amount,
			Currency:           item.Currency,
			BillingDate:        billingDate,
			PaymentCycle:       item.PaymentCycle,
		}.Normalize()
		if parsedEmail.IsEmpty() {
			continue
		}
		parsedEmails = append(parsedEmails, parsedEmail)
	}

	return parsedEmails, nil
}

func parseBillingDate(raw *string) (*time.Time, error) {
	if raw == nil {
		return nil, nil
	}

	value := strings.TrimSpace(*raw)
	if value == "" {
		return nil, nil
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err == nil {
		utc := parsed.UTC()
		return &utc, nil
	}

	return nil, fmt.Errorf("invalid billingDate: %s", value)
}
