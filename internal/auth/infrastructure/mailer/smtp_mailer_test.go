package mailer

import (
	"business/internal/auth/domain"
	"business/internal/library/logger"
	"business/internal/library/sendMailerClient"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSMTPVerificationEmailSender_SendVerificationEmail_ComposesMessage(t *testing.T) {
	client := &spySMTPClient{}
	sender := NewSMTPVerificationEmailSender(client, logger.NewNop())

	user := domain.User{Name: "山田太郎", Email: "taro@example.com"}
	verifyURL := "https://example.com/verify"

	err := sender.SendVerificationEmail(context.Background(), user, verifyURL)
	require.NoError(t, err)

	require.Equal(t, user.Email, client.msg.To)
	require.Contains(t, client.msg.Subject, "メールアドレスの確認")
	require.Contains(t, client.msg.Body, user.Name)
	require.Contains(t, client.msg.Body, verifyURL)
}

func TestSMTPVerificationEmailSender_SendVerificationEmail_ClientError(t *testing.T) {
	client := &spySMTPClient{
		err: errors.New("smtp failure"),
	}
	sender := NewSMTPVerificationEmailSender(client, logger.NewNop())

	err := sender.SendVerificationEmail(context.Background(), domain.User{Email: "err@example.com"}, "https://example.com/verify")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to send email")
	require.Contains(t, err.Error(), "smtp failure")
}

type spySMTPClient struct {
	msg sendMailerClient.Message
	err error
}

func (s *spySMTPClient) Send(ctx context.Context, msg sendMailerClient.Message) error {
	s.msg = msg
	return s.err
}
