package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	// ErrEmailUserIDEmpty is returned when the user ID is missing.
	ErrEmailUserIDEmpty = errors.New("email user id is empty")
	// ErrEmailExternalMessageIDEmpty is returned when the external message ID is missing.
	ErrEmailExternalMessageIDEmpty = errors.New("email external message id is empty")
)

// FetchedEmailDTO は取得直後のメールを表す共通DTOです
type FetchedEmailDTO struct {
	ID      string    `json:"id"`
	Subject string    `json:"subject"`
	From    string    `json:"from"`
	To      []string  `json:"to"`
	Date    time.Time `json:"date"`
	Body    string    `json:"body"`
}

// ExtractSenderName は From フィールドから送信者名を抽出します
func (b FetchedEmailDTO) ExtractSenderName() string {
	if idx := strings.Index(b.From, "<"); idx > 0 {
		return strings.TrimSpace(b.From[:idx])
	}
	return b.From
}

// ExtractEmailAddress は From フィールドからメールアドレスを抽出します
func (b FetchedEmailDTO) ExtractEmailAddress() string {
	start := strings.Index(b.From, "<")
	end := strings.Index(b.From, ">")
	if start >= 0 && end > start {
		return b.From[start+1 : end]
	}
	return b.From
}

// Email represents a fetched email message in the domain.
// It keeps raw data without semantic interpretation.
type Email struct {
	ID                uint
	UserID            uint
	ExternalMessageID string
	Subject           string
	From              string
	To                []string
	ReceivedAt        time.Time
	ParsedEmails      []ParsedEmail
}

// NewEmailFromFetchedDTO builds an Email from a fetched DTO.
func NewEmailFromFetchedDTO(userID uint, dto FetchedEmailDTO) (Email, error) {
	email := Email{
		UserID:            userID,
		ExternalMessageID: strings.TrimSpace(dto.ID),
		Subject:           dto.Subject,
		From:              dto.From,
		To:                dto.To,
		ReceivedAt:        dto.Date,
	}
	if err := email.Validate(); err != nil {
		return Email{}, err
	}
	return email, nil
}

// Validate enforces basic invariants for Email.
func (e Email) Validate() error {
	if e.UserID == 0 {
		return ErrEmailUserIDEmpty
	}
	if strings.TrimSpace(e.ExternalMessageID) == "" {
		return ErrEmailExternalMessageIDEmpty
	}
	return nil
}

// HasParsedEmail reports whether parsed data is attached.
func (e Email) HasParsedEmail() bool {
	return len(e.ParsedEmails) > 0
}

// AppendParsedEmail appends parsed data to the email.
func (e *Email) AppendParsedEmail(parsed ParsedEmail) {
	e.ParsedEmails = append(e.ParsedEmails, parsed)
}

// ParsedEmail represents the analyzed, structured data derived from an Email.
// It does not represent a final billing decision.
type ParsedEmail struct {
	VendorName    *string    `json:"vendorName"`
	BillingNumber *string    `json:"billingNumber"`
	InvoiceNumber *string    `json:"invoiceNumber"`
	Amount        *float64   `json:"amount"`
	Currency      *string    `json:"currency"`
	BillingDate   *time.Time `json:"billingDate"`
	PaymentCycle  *string    `json:"paymentCycle"`
	ExtractedAt   time.Time  `json:"extractedAt"`
}
