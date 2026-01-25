//go:build integration
// +build integration

package sendMailerClient

import (
	mocktools "business/test/mock/tools"
	"context"
	"errors"
	"net/smtp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSMTPClient_Send_UsesEnvConfig(t *testing.T) {
	osw := mocktools.NewOsWrapperMock(map[string]string{
		"SMTP_HOST":          "mailhog",
		"SMTP_PORT":          "1025",
		"EMAIL_FROM_ADDRESS": "custom@example.com",
	})

	var captured struct {
		addr string
		from string
		to   []string
		msg  []byte
	}

	client := &smtpClient{
		osw: osw,
		sendMail: func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
			captured.addr = addr
			captured.from = from
			captured.to = append([]string(nil), to...)
			captured.msg = append([]byte(nil), msg...)
			return nil
		},
	}

	err := client.Send(context.Background(), Message{
		To:      "alice@example.com",
		Subject: "subject",
		Body:    "メール本文",
	})
	require.NoError(t, err)
	require.Equal(t, "smtp.example.com:2525", captured.addr)
	require.Equal(t, "custom@example.com", captured.from)
	require.Equal(t, []string{"alice@example.com"}, captured.to)
	require.Contains(t, string(captured.msg), "Subject: subject")
	require.Contains(t, string(captured.msg), "メール本文")
}

func TestSMTPClient_Send_ReturnsSendMailError(t *testing.T) {
	expectedErr := errors.New("smtp failure")
	osw := mocktools.NewOsWrapperMock(nil)

	client := &smtpClient{
		osw: osw,
		sendMail: func(string, smtp.Auth, string, []string, []byte) error {
			return expectedErr
		},
	}

	err := client.Send(context.Background(), Message{
		To:      "bob@example.com",
		Subject: "test",
		Body:    "body",
	})
	require.ErrorIs(t, err, expectedErr)
}

func TestSMTPClient_Send_ContextCanceled(t *testing.T) {
	osw := mocktools.NewOsWrapperMock(nil)
	client := &smtpClient{
		osw: osw,
		sendMail: func(string, smtp.Auth, string, []string, []byte) error {
			t.Fatal("sendMail should not be called")
			return nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.Send(ctx, Message{
		To:      "bob@example.com",
		Subject: "test",
		Body:    "body",
	})
	require.ErrorIs(t, err, context.Canceled)
}
