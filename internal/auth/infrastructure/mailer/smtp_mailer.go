package mailer

import (
	"business/internal/auth/domain"
	"business/internal/library/logger"
	"business/internal/library/sendMailerClient"
	"context"
	"fmt"
)

// SMTPVerificationEmailSender sends verification emails via SMTP
type SMTPVerificationEmailSender struct {
	client sendMailerClient.Client
	logger logger.Interface
}

// NewSMTPVerificationEmailSender creates a new SMTP email sender.
// If logger is nil, it defaults to logger.NewNop().
func NewSMTPVerificationEmailSender(client sendMailerClient.Client, log logger.Interface) *SMTPVerificationEmailSender {
	if client == nil {
		panic("mailer: sendMailerClient.Client is required")
	}
	if log == nil {
		log = logger.NewNop()
	}
	return &SMTPVerificationEmailSender{
		client: client,
		logger: log.With(logger.String("component", "auth_mailer")),
	}
}

// SendVerificationEmail sends a verification email to the user
func (s *SMTPVerificationEmailSender) SendVerificationEmail(ctx context.Context, user domain.User, verifyURL string) error {
	subject := "【重要】メールアドレスの確認をお願いします"
	body := fmt.Sprintf(`%s 様

ご登録いただきありがとうございます。

以下のリンクをクリックして、メールアドレスの確認を完了してください:

%s

このリンクの有効期限は、発行から3時間以内です。

本メールに心当たりがない場合は、このメールを無視してください。

よろしくお願いいたします。
`, user.Name.String(), verifyURL)

	if err := s.client.Send(ctx, sendMailerClient.Message{
		To:      user.Email.String(),
		Subject: subject,
		Body:    body,
	}); err != nil {
		s.logger.Error("failed to send verification email",
			logger.String("email", user.Email.String()),
			logger.Uint("user_id", user.ID),
			logger.Err(err),
		)
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
