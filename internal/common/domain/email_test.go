package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNewEmailFromFetchedDTO(t *testing.T) {
	t.Parallel()

	t.Run("creates email from dto", func(t *testing.T) {
		t.Parallel()

		dto := FetchedEmailDTO{
			ID:      "msg-123",
			Subject: "subject",
			From:    "sender@example.com",
			To:      []string{"to@example.com"},
			Date:    time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC),
		}

		email, err := NewEmailFromFetchedDTO(1, dto)
		if err != nil {
			t.Fatalf("NewEmailFromFetchedDTO returned error: %v", err)
		}

		if email.UserID != 1 {
			t.Fatalf("want user id 1, got %d", email.UserID)
		}
		if email.ExternalMessageID != "msg-123" {
			t.Fatalf("want external message id msg-123, got %s", email.ExternalMessageID)
		}
		if email.Subject != "subject" {
			t.Fatalf("want subject subject, got %s", email.Subject)
		}
		if email.From != "sender@example.com" {
			t.Fatalf("want from sender@example.com, got %s", email.From)
		}
		if len(email.To) != 1 || email.To[0] != "to@example.com" {
			t.Fatalf("unexpected to list: %v", email.To)
		}
		if !email.ReceivedAt.Equal(dto.Date) {
			t.Fatalf("want receivedAt %v, got %v", dto.Date, email.ReceivedAt)
		}
	})

	t.Run("returns error when user id is empty", func(t *testing.T) {
		t.Parallel()

		dto := FetchedEmailDTO{ID: "msg-123"}
		_, err := NewEmailFromFetchedDTO(0, dto)
		if !errors.Is(err, ErrEmailUserIDEmpty) {
			t.Fatalf("expected ErrEmailUserIDEmpty, got %v", err)
		}
	})

	t.Run("returns error when external id is empty", func(t *testing.T) {
		t.Parallel()

		dto := FetchedEmailDTO{ID: "  "}
		_, err := NewEmailFromFetchedDTO(1, dto)
		if !errors.Is(err, ErrEmailExternalMessageIDEmpty) {
			t.Fatalf("expected ErrEmailExternalMessageIDEmpty, got %v", err)
		}
	})
}

func TestEmailAppendParsedEmail(t *testing.T) {
	t.Parallel()

	email := Email{UserID: 1, ExternalMessageID: "msg-123"}
	email.AppendParsedEmail(ParsedEmail{})
	if !email.HasParsedEmail() {
		t.Fatalf("expected parsed email to be attached")
	}
	if len(email.ParsedEmails) != 1 {
		t.Fatalf("expected 1 parsed email, got %d", len(email.ParsedEmails))
	}

	email.AppendParsedEmail(ParsedEmail{})
	if len(email.ParsedEmails) != 2 {
		t.Fatalf("expected 2 parsed emails, got %d", len(email.ParsedEmails))
	}
}

func TestParsedEmailNormalize(t *testing.T) {
	t.Parallel()

	billingDate := time.Date(2026, 3, 24, 12, 0, 0, 0, time.FixedZone("JST", 9*60*60))
	extractedAt := time.Date(2026, 3, 24, 13, 0, 0, 0, time.FixedZone("JST", 9*60*60))

	parsed := ParsedEmail{
		ProductNameRaw:     stringPtr(" Example Product Full Name "),
		ProductNameDisplay: stringPtr(" Example Product "),
		VendorName:         stringPtr(" Example Vendor "),
		BillingNumber:      stringPtr(" INV-001 "),
		InvoiceNumber:      stringPtr(" inv-001 "),
		Amount:             float64Ptr(123.456),
		Currency:           stringPtr(" jpy "),
		BillingDate:        &billingDate,
		PaymentCycle:       stringPtr(" One Time "),
		ExtractedAt:        extractedAt,
	}.Normalize()

	if got := *parsed.ProductNameRaw; got != "Example Product Full Name" {
		t.Fatalf("unexpected raw product name: %q", got)
	}
	if got := *parsed.ProductNameDisplay; got != "Example Product" {
		t.Fatalf("unexpected display product name: %q", got)
	}
	if got := *parsed.VendorName; got != "Example Vendor" {
		t.Fatalf("unexpected vendor name: %q", got)
	}
	if got := *parsed.BillingNumber; got != "INV-001" {
		t.Fatalf("unexpected billing number: %q", got)
	}
	if got := *parsed.InvoiceNumber; got != "inv-001" {
		t.Fatalf("unexpected invoice number: %q", got)
	}
	if got := *parsed.Currency; got != "JPY" {
		t.Fatalf("unexpected currency: %q", got)
	}
	if got := *parsed.PaymentCycle; got != "one_time" {
		t.Fatalf("unexpected payment cycle: %q", got)
	}
	if parsed.BillingDate == nil || parsed.BillingDate.Location() != time.UTC {
		t.Fatalf("expected UTC billing date, got %+v", parsed.BillingDate)
	}
	if parsed.ExtractedAt.Location() != time.UTC {
		t.Fatalf("expected UTC extracted_at, got %+v", parsed.ExtractedAt)
	}
}

func TestParsedEmailNormalize_BlankOptionalValuesBecomeNil(t *testing.T) {
	t.Parallel()

	parsed := ParsedEmail{
		ProductNameRaw:     stringPtr(" "),
		ProductNameDisplay: stringPtr(" "),
		VendorName:         stringPtr(" "),
		BillingNumber:      stringPtr(" "),
		InvoiceNumber:      stringPtr(" "),
		Currency:           stringPtr(" "),
		PaymentCycle:       stringPtr(" "),
	}.Normalize()

	if !parsed.IsEmpty() {
		t.Fatalf("expected parsed email to become empty, got %+v", parsed)
	}
}

func TestParsedEmailWithExtractedAt(t *testing.T) {
	t.Parallel()

	extractedAt := time.Date(2026, 3, 24, 13, 0, 0, 0, time.FixedZone("JST", 9*60*60))
	parsed := ParsedEmail{
		ProductNameDisplay: stringPtr("Product"),
	}.WithExtractedAt(extractedAt)

	if got := *parsed.ProductNameDisplay; got != "Product" {
		t.Fatalf("unexpected product name: %q", got)
	}
	if parsed.ExtractedAt.Location() != time.UTC {
		t.Fatalf("expected UTC extracted_at, got %+v", parsed.ExtractedAt)
	}
}

func stringPtr(value string) *string {
	return &value
}

func float64Ptr(value float64) *float64 {
	return &value
}
