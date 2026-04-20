package oswrapper

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"business/internal/library/secret"
)

// OsWrapper は両方の機能を持つ具象構造体です
type OsWrapper struct {
	appSecretClient secret.Client
}

var appSecretEnvKeys = map[string]struct{}{
	"JWT_SECRET_KEY":            {},
	"OPENAI_API_KEY":            {},
	"EMAIL_TOKEN_KEY_V1":        {},
	"EMAIL_TOKEN_SALT":          {},
	"REDIS_PASSWORD":            {},
	"MYSQL_USER":                {},
	"MYSQL_PASSWORD":            {},
	"EMAIL_GMAIL_CLIENT_ID":     {},
	"EMAIL_GMAIL_CLIENT_SECRET": {},
}

// New は OsWrapper のインスタンスを返します
func New(appSecretClient secret.Client) (*OsWrapper, error) {
	wrapper := &OsWrapper{
		appSecretClient: appSecretClient,
	}

	if !shouldUseSecretManager() {
		return wrapper, nil
	}

	return wrapper, nil
}

// ReadFile はファイルを読み込み文字列として返します
func (o *OsWrapper) ReadFile(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("パス解決失敗: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("ファイル読み込み失敗: %w", err)
	}

	return string(data), nil
}

// GetEnv は環境変数を取得します。空文字の場合はエラーを返します。
func (o *OsWrapper) GetEnv(key string) (string, error) {
	if shouldUseSecretManager() {
		if _, ok := appSecretEnvKeys[key]; ok {
			if o.appSecretClient == nil {
				return "", errors.New("アプリ共通のシークレットクライアントがnilです")
			}
			val, err := o.appSecretClient.GetValue(context.Background(), key)
			if err != nil {
				return "", fmt.Errorf("シークレット %s の取得に失敗しました: %w", key, err)
			}
			if strings.TrimSpace(val) == "" {
				return "", fmt.Errorf("AWS SecretManagerに %s が設定されていません", key)
			}
			return val, nil
		}
	}

	value := os.Getenv(key)
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("環境変数 %s が設定されていません", key)
	}

	return value, nil
}

func shouldUseSecretManager() bool {
	stage := strings.TrimSpace(os.Getenv("APP"))
	return stage != "" && stage != "local" && stage != "ci"
}
