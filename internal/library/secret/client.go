package secret

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

const (
	defaultVersionStage = "AWSCURRENT"
	defaultSecretName   = "billingrse_dev"
	defaultRegion       = "ap-northeast-1"
)

type secretClient struct {
}

func New(ctx context.Context) (Client, error) {
	if ctx == nil {
		return nil, errors.New("ctx is required")
	}

	return &secretClient{}, nil
}

func getSecrets(ctx context.Context) (map[string]string, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(defaultRegion))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	svc := secretsmanager.NewFromConfig(cfg)
	result, err := svc.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(defaultSecretName),
		VersionStage: aws.String(defaultVersionStage),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %s: %w", defaultSecretName, err)
	}

	raw := ""
	if result.SecretString != nil {
		raw = *result.SecretString
	} else if len(result.SecretBinary) > 0 {
		decoded, decErr := base64.StdEncoding.DecodeString(string(result.SecretBinary))
		if decErr != nil {
			return nil, fmt.Errorf("failed to decode secret binary: %w", decErr)
		}
		raw = string(decoded)
	}

	values, err := parseSecret(raw)
	if err != nil {
		return nil, err
	}

	return values, nil
}

// GetValue は指定されたキーの値を返します。
// このクライアントは 1 つのシークレット（初期化時に指定）だけを扱います。
func (c *secretClient) GetValue(ctx context.Context, key string) (string, error) {
	if key == "" {
		return "", errors.New("key is required")
	}

	secrets, err := getSecrets(ctx)
	if err != nil {
		return "", err
	}
	v, ok := secrets[key]
	if !ok {
		return "", fmt.Errorf("%s: not found in secret", key)
	}
	return v, nil
}

func parseSecret(raw string) (map[string]string, error) {
	if raw == "" {
		return nil, errors.New("secret string is empty")
	}

	var obj map[string]string
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal secret JSON: %w", err)
	}
	if len(obj) == 0 {
		return nil, errors.New("secret JSON is empty")
	}
	return obj, nil
}
