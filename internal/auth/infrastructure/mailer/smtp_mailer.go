package mailer

import (
	"business/internal/auth/domain"
	"business/internal/library/logger"
	"business/internal/library/sendMailer"
	"context"
	"fmt"
)

// SMTPVerificationEmailSender sends verification emails via SMTP
type SMTPVerificationEmailSender struct {
	client sendMailer.Client
	logger logger.Interface
}

// NewSMTPVerificationEmailSender creates a new SMTP email sender.
// If logger is nil, it defaults to logger.NewNop().
func NewSMTPVerificationEmailSender(client sendMailer.Client, log logger.Interface) *SMTPVerificationEmailSender {
	if client == nil {
		panic("mailer: sendMailer.Client is required")
	}
	if log == nil {
		log = logger.NewNop()
	}
	return &SMTPVerificationEmailSender{
		client: client,
		logger: log.With(logger.Component("auth_mailer")),
	}
}

// SendVerificationEmail sends a verification email to the user
func (s *SMTPVerificationEmailSender) SendVerificationEmail(ctx context.Context, user domain.User, verifyURL string) error {
	if ctx == nil {
		return logger.ErrNilContext
	}

	reqLog := s.logger
	if withContext, err := s.logger.WithContext(ctx); err == nil {
		reqLog = withContext
	}

	subject := "【重要】メールアドレスの確認をお願いします"
	body := fmt.Sprintf(`%s 様

ご登録いただきありがとうございます。

以下のリンクをクリックして、メールアドレスの確認を完了してください:

%s

このリンクの有効期限は、発行から3時間以内です。

本メールに心当たりがない場合は、このメールを無視してください。

よろしくお願いいたします。
`, user.Name.String(), verifyURL)

	if err := s.client.Send(ctx, sendMailer.Message{
		To:      user.Email.String(),
		Subject: subject,
		Body:    body,
	}); err != nil {
		reqLog.Error("external_api_failed",
			logger.String("provider", "smtp"),
			logger.String("operation", "send_verification_email"),
			logger.Uint("user_id", user.ID),
			logger.Err(err),
		)
		return fmt.Errorf("failed to send email: %w", err)
	}

	reqLog.Info("external_api_succeeded",
		logger.String("provider", "smtp"),
		logger.String("operation", "send_verification_email"),
		logger.Uint("user_id", user.ID),
	)

	return nil
}
