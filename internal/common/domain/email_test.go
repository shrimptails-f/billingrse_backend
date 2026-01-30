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
