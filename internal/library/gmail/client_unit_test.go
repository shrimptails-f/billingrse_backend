package gmail

import (
	"encoding/base64"
	"testing"
	"unicode/utf8"

	"golang.org/x/text/encoding/japanese"
	gmailapi "google.golang.org/api/gmail/v1"
)

func TestExtractBody_DecodesRawURLBase64WithoutPadding(t *testing.T) {
	t.Parallel()

	payload := &gmailapi.MessagePart{
		MimeType: "text/plain",
		Headers: []*gmailapi.MessagePartHeader{
			{Name: "Content-Type", Value: "text/plain; charset=utf-8"},
		},
		Body: &gmailapi.MessagePartBody{
			Data: base64.RawURLEncoding.EncodeToString([]byte("Hello, billing world!")),
		},
	}

	if got := extractBody(payload); got != "Hello, billing world!" {
		t.Fatalf("unexpected body: %q", got)
	}
}

func TestExtractBody_DecodesShiftJISBody(t *testing.T) {
	t.Parallel()

	encoded, err := japanese.ShiftJIS.NewEncoder().Bytes([]byte("請求書"))
	if err != nil {
		t.Fatalf("failed to encode test body: %v", err)
	}

	payload := &gmailapi.MessagePart{
		MimeType: "text/plain",
		Headers: []*gmailapi.MessagePartHeader{
			{Name: "Content-Type", Value: "text/plain; charset=shift_jis"},
		},
		Body: &gmailapi.MessagePartBody{
			Data: base64.RawURLEncoding.EncodeToString(encoded),
		},
	}

	if got := extractBody(payload); got != "請求書" {
		t.Fatalf("unexpected decoded body: %q", got)
	}
}

func TestNormalizeMailText_RemovesInvalidUTF8AndNullBytes(t *testing.T) {
	t.Parallel()

	raw := string([]byte{'a', 0xff, 'b', 0x00, 'c'})
	got := normalizeMailText(raw)

	if got != "a b c" {
		t.Fatalf("unexpected normalized text: %q", got)
	}
	if !utf8.ValidString(got) {
		t.Fatalf("normalized text must be valid utf-8: %q", got)
	}
}
