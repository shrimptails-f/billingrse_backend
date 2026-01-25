package sendMailerClient

import (
	"business/internal/library/oswrapper"
	"context"
	"fmt"
	"net/smtp"
)

// Message represents the minimum information necessary to deliver an email.
type Message struct {
	To      string
	Subject string
	Body    string
}

type smtpClient struct {
	osw      oswrapper.OsWapperInterface
	sendMail func(addr string, a smtp.Auth, from string, to []string, msg []byte) error
}

// New builds a production-ready SMTP client that fetches configuration
// from the provided os wrapper.
func New(osw oswrapper.OsWapperInterface) Client {
	return &smtpClient{
		osw:      osw,
		sendMail: smtp.SendMail,
	}
}

// Send composes and delivers the email based on environment configuration.
func (c *smtpClient) Send(ctx context.Context, msg Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	smtpHost, err := c.osw.GetEnv("SMTP_HOST")
	if err != nil {
		return fmt.Errorf("failed to load SMTP_HOST: %w", err)
	}
	smtpPort, err := c.osw.GetEnv("SMTP_PORT")
	if err != nil {
		return fmt.Errorf("failed to load SMTP_PORT: %w", err)
	}
	from, err := c.osw.GetEnv("EMAIL_FROM_ADDRESS")
	if err != nil {
		return fmt.Errorf("failed to load EMAIL_FROM_ADDRESS: %w", err)
	}

	payload := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n"+
		"\r\n"+
		"%s", from, msg.To, msg.Subject, msg.Body)

	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)
	return c.sendMail(addr, nil, from, []string{msg.To}, []byte(payload))
}
