package infrastructure

import (
	maapp "business/internal/mailanalysis/application"
	"business/internal/mailanalysis/domain"
	"context"
	"errors"
	"testing"
	"time"
)

type mockOpenAIClient struct {
	chat func(ctx context.Context, prompt string) (string, error)
}

func (m *mockOpenAIClient) Chat(ctx context.Context, prompt string) (string, error) {
	return m.chat(ctx, prompt)
}

func TestOpenAIAnalyzerAdapter_Analyze_Success(t *testing.T) {
	t.Parallel()

	adapter := NewOpenAIAnalyzerAdapter(&mockOpenAIClient{
		chat: func(ctx context.Context, prompt string) (string, error) {
			return "{\n  \"parsedEmails\": [\n    {\n      \"productNameRaw\": \" Example Product Full Name \",\n      \"productNameDisplay\": \" Example Product \",\n      \"vendorName\": \" Example Vendor \",\n      \"billingNumber\": \" INV-001 \",\n      \"invoiceNumber\": null,\n      \"amount\": 12.345,\n      \"currency\": \" jpy \",\n      \"billingDate\": \"2026-03-24T00:00:00Z\",\n      \"paymentCycle\": \"one time\"\n    }\n  ]\n}", nil
		},
	}, nil)

	output, err := adapter.Analyze(context.Background(), maapp.EmailForAnalysisTarget{
		EmailID:           1,
		ExternalMessageID: "msg-1",
		Subject:           "subject",
		From:              "from@example.com",
		ReceivedAt:        time.Date(2026, 3, 24, 10, 0, 0, 0, time.UTC),
		Body:              "body",
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	if output.PromptVersion != promptVersion {
		t.Fatalf("unexpected metadata: %+v", output)
	}
	if len(output.ParsedEmails) != 1 {
		t.Fatalf("unexpected parsed emails: %+v", output.ParsedEmails)
	}

	parsed := output.ParsedEmails[0]
	if got := *parsed.ProductNameRaw; got != "Example Product Full Name" {
		t.Fatalf("unexpected raw product: %q", got)
	}
	if got := *parsed.ProductNameDisplay; got != "Example Product" {
		t.Fatalf("unexpected display product: %q", got)
	}
	if got := *parsed.VendorName; got != "Example Vendor" {
		t.Fatalf("unexpected vendor: %q", got)
	}
	if got := *parsed.BillingNumber; got != "INV-001" {
		t.Fatalf("unexpected billing number: %q", got)
	}
	if got := *parsed.Currency; got != "JPY" {
		t.Fatalf("unexpected currency: %q", got)
	}
	if got := *parsed.PaymentCycle; got != "one_time" {
		t.Fatalf("unexpected payment cycle: %q", got)
	}
	if parsed.BillingDate == nil || parsed.BillingDate.Format("2006-01-02") != "2026-03-24" {
		t.Fatalf("unexpected billing date: %+v", parsed.BillingDate)
	}
	if !parsed.ExtractedAt.IsZero() {
		t.Fatalf("expected zero extracted_at before save, got %+v", parsed.ExtractedAt)
	}
}

func TestOpenAIAnalyzerAdapter_Analyze_InvalidResponse(t *testing.T) {
	t.Parallel()

	adapter := NewOpenAIAnalyzerAdapter(&mockOpenAIClient{
		chat: func(ctx context.Context, prompt string) (string, error) {
			return `[{"billingDate":"2026-03-24"}]`, nil
		},
	}, nil)

	_, err := adapter.Analyze(context.Background(), maapp.EmailForAnalysisTarget{
		EmailID:           1,
		ExternalMessageID: "msg-1",
		Body:              "body",
	})
	if !errors.Is(err, domain.ErrAnalysisResponseInvalid) {
		t.Fatalf("expected ErrAnalysisResponseInvalid, got %v", err)
	}
}

func TestOpenAIAnalyzerAdapter_Analyze_InvalidResponseNonRFC3339BillingDate(t *testing.T) {
	t.Parallel()

	adapter := NewOpenAIAnalyzerAdapter(&mockOpenAIClient{
		chat: func(ctx context.Context, prompt string) (string, error) {
			return `{"parsedEmails":[{"billingDate":"not-a-date"}]}`, nil
		},
	}, nil)

	_, err := adapter.Analyze(context.Background(), maapp.EmailForAnalysisTarget{
		EmailID:           1,
		ExternalMessageID: "msg-1",
		Body:              "body",
	})
	if !errors.Is(err, domain.ErrAnalysisResponseInvalid) {
		t.Fatalf("expected ErrAnalysisResponseInvalid, got %v", err)
	}
}

func TestOpenAIAnalyzerAdapter_Analyze_UnknownFieldIsIgnored(t *testing.T) {
	t.Parallel()

	adapter := NewOpenAIAnalyzerAdapter(&mockOpenAIClient{
		chat: func(ctx context.Context, prompt string) (string, error) {
			return `{"parsedEmails":[{"product_name_raw":"Example Product"}]}`, nil
		},
	}, nil)

	output, err := adapter.Analyze(context.Background(), maapp.EmailForAnalysisTarget{
		EmailID:           1,
		ExternalMessageID: "msg-1",
		Body:              "body",
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(output.ParsedEmails) != 0 {
		t.Fatalf("expected empty parsed emails, got %+v", output.ParsedEmails)
	}
}

func TestOpenAIAnalyzerAdapter_Analyze_InvalidResponseCodeFence(t *testing.T) {
	t.Parallel()

	adapter := NewOpenAIAnalyzerAdapter(&mockOpenAIClient{
		chat: func(ctx context.Context, prompt string) (string, error) {
			return "```json\n{\"parsedEmails\":[{\"productNameRaw\":\"Example Product Full Name\"}]}\n```", nil
		},
	}, nil)

	_, err := adapter.Analyze(context.Background(), maapp.EmailForAnalysisTarget{
		EmailID:           1,
		ExternalMessageID: "msg-1",
		Body:              "body",
	})
	if !errors.Is(err, domain.ErrAnalysisResponseInvalid) {
		t.Fatalf("expected ErrAnalysisResponseInvalid, got %v", err)
	}
}

func TestOpenAIAnalyzerAdapter_Analyze_InvalidResponsePrefixedText(t *testing.T) {
	t.Parallel()

	adapter := NewOpenAIAnalyzerAdapter(&mockOpenAIClient{
		chat: func(ctx context.Context, prompt string) (string, error) {
			return "Here is the result:\n{\"parsedEmails\":[{\"productNameRaw\":\"Example Product Full Name\"}]}", nil
		},
	}, nil)

	_, err := adapter.Analyze(context.Background(), maapp.EmailForAnalysisTarget{
		EmailID:           1,
		ExternalMessageID: "msg-1",
		Body:              "body",
	})
	if !errors.Is(err, domain.ErrAnalysisResponseInvalid) {
		t.Fatalf("expected ErrAnalysisResponseInvalid, got %v", err)
	}
}

func TestOpenAIAnalyzerAdapter_Analyze_AllNullDraftsBecomeEmpty(t *testing.T) {
	t.Parallel()

	adapter := NewOpenAIAnalyzerAdapter(&mockOpenAIClient{
		chat: func(ctx context.Context, prompt string) (string, error) {
			return `{"parsedEmails":[{"productNameRaw":null,"productNameDisplay":null,"vendorName":null,"billingNumber":null,"invoiceNumber":null,"amount":null,"currency":null,"billingDate":null,"paymentCycle":null}]}`, nil
		},
	}, nil)

	output, err := adapter.Analyze(context.Background(), maapp.EmailForAnalysisTarget{
		EmailID:           1,
		ExternalMessageID: "msg-1",
		Body:              "body",
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(output.ParsedEmails) != 0 {
		t.Fatalf("expected empty parsed emails, got %+v", output.ParsedEmails)
	}
}
