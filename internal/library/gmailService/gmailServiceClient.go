package gmailService

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/api/option"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

type Client struct {
}

// GメールのAPIコールをする前の処理をまとめた構造体です
// 認可はURLを手動で開く必要があります。
func New() *Client {
	return &Client{}
}

func (c *Client) CreateServiceWithToken(ctx context.Context, credentialsPath string, token *oauth2.Token) (*gmail.Service, error) {
	credBytes, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("クレデンシャル読み込み失敗: %w", err)
	}

	config, err := google.ConfigFromJSON(credBytes, gmail.GmailReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("OAuth2構成失敗: %w", err)
	}

	return c.CreateServiceWithTokenSource(ctx, config.TokenSource(ctx, token))
}

// CreateServiceWithTokenSource builds a Gmail service using the provided token source.
func (c *Client) CreateServiceWithTokenSource(ctx context.Context, tokenSource oauth2.TokenSource) (*gmail.Service, error) {
	svc, err := gmail.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, fmt.Errorf("gmailサービス初期化失敗: %w", err)
	}

	return svc, nil
}
