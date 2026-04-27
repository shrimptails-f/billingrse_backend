package sendMailer

import (
	"business/internal/library/oswrapper"
	"context"
	"fmt"
	"net/smtp"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	sestypes "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

// Message はメール送信に必要な最小情報を表します。
type Message struct {
	To      string
	Subject string
	Body    string
}

type SmtpClient struct {
	osw      oswrapper.OsWapperInterface
	sendMail func(addr string, a smtp.Auth, from string, to []string, msg []byte) error
}

type sesSendEmailAPI interface {
	SendEmail(ctx context.Context, params *sesv2.SendEmailInput, optFns ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error)
}

type SesClient struct {
	osw    oswrapper.OsWapperInterface
	client sesSendEmailAPI
}

// New は受け取った os wrapper から設定を読み取り、
// 本番利用可能な SMTP クライアントを生成します。
func New(osw oswrapper.OsWapperInterface) *SmtpClient {
	return &SmtpClient{
		osw:      osw,
		sendMail: smtp.SendMail,
	}
}

// NewSES は受け取った os wrapper から設定を読み取り、
// 実行環境の AWS 認証情報を使う SES クライアントを生成します。
func NewSES(ctx context.Context, osw oswrapper.OsWapperInterface) (*SesClient, error) {
	region, err := osw.GetEnv("AWS_REGION")
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS_REGION: %w", err)
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load aws config: %w", err)
	}

	return &SesClient{
		osw:    osw,
		client: sesv2.NewFromConfig(cfg),
	}, nil
}

// Send は環境変数の設定値に基づいてメールを組み立てて送信します。
func (c *SmtpClient) Send(ctx context.Context, msg Message) error {
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

// Send は Amazon SES 経由でメールを組み立てて送信します。
func (c *SesClient) Send(ctx context.Context, msg Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	from, err := c.osw.GetEnv("EMAIL_FROM_ADDRESS")
	if err != nil {
		return fmt.Errorf("failed to load EMAIL_FROM_ADDRESS: %w", err)
	}

	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(from),
		Destination: &sestypes.Destination{
			ToAddresses: []string{msg.To},
		},
		Content: &sestypes.EmailContent{
			Simple: &sestypes.Message{
				Subject: &sestypes.Content{
					Data:    aws.String(msg.Subject),
					Charset: aws.String("UTF-8"),
				},
				Body: &sestypes.Body{
					Text: &sestypes.Content{
						Data:    aws.String(msg.Body),
						Charset: aws.String("UTF-8"),
					},
				},
			},
		},
	}

	if configSetName, err := c.osw.GetEnv("SES_CONFIGURATION_SET_NAME"); err == nil && configSetName != "" {
		input.ConfigurationSetName = aws.String(configSetName)
	}

	if _, err := c.client.SendEmail(ctx, input); err != nil {
		return fmt.Errorf("failed to send email with ses: %w", err)
	}

	return nil
}
