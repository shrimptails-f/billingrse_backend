package infrastructure

import (
	commondomain "business/internal/common/domain"
	"business/internal/library/logger"
	openailib "business/internal/library/openai"
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
	ProductNameRaw     *string                       `json:"productNameRaw"`
	ProductNameDisplay *string                       `json:"productNameDisplay"`
	VendorName         *string                       `json:"vendorName"`
	BillingNumber      *string                       `json:"billingNumber"`
	InvoiceNumber      *string                       `json:"invoiceNumber"`
	Amount             *float64                      `json:"amount"`
	Currency           *string                       `json:"currency"`
	BillingDate        *string                       `json:"billingDate"`
	PaymentCycle       *string                       `json:"paymentCycle"`
	LineItems          []parsedEmailLineItemResponse `json:"lineItems"`
}

type parsedEmailLineItemResponse struct {
	ProductNameRaw     *string  `json:"productNameRaw"`
	ProductNameDisplay *string  `json:"productNameDisplay"`
	Amount             *float64 `json:"amount"`
	Currency           *string  `json:"currency"`
}

type parsedEmailResponseEnvelope struct {
	ParsedEmails []parsedEmailResponse `json:"parsedEmails"`
}

func buildPrompt(email maapp.EmailForAnalysisTarget) string {
	return openailib.BuildParsedEmailPrompt(email.Subject, email.From, email.ReceivedAt, email.Body)
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

		lineItems := make([]commondomain.ParsedEmailLineItem, 0, len(item.LineItems))
		for _, lineItem := range item.LineItems {
			normalizedLineItem := commondomain.ParsedEmailLineItem{
				ProductNameRaw:     lineItem.ProductNameRaw,
				ProductNameDisplay: lineItem.ProductNameDisplay,
				Amount:             lineItem.Amount,
				Currency:           lineItem.Currency,
			}.Normalize()
			if normalizedLineItem.IsEmpty() {
				continue
			}
			lineItems = append(lineItems, normalizedLineItem)
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
			LineItems:          lineItems,
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
