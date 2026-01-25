package gmailService

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type oauthStubOsWrapper struct {
	env map[string]string
}

func (s *oauthStubOsWrapper) ReadFile(string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (s *oauthStubOsWrapper) GetEnv(key string) (string, error) {
	if s.env == nil {
		return "", fmt.Errorf("environment variable %s not set", key)
	}
	if value, ok := s.env[key]; ok && value != "" {
		return value, nil
	}
	return "", fmt.Errorf("environment variable %s not set", key)
}

func TestOAuthConfigLoader_UsesInlineClientIDAndSecret(t *testing.T) {
	t.Parallel()

	osw := &oauthStubOsWrapper{
		env: map[string]string{
			"EMAIL_GMAIL_CLIENT_ID":     "test-client-id",
			"EMAIL_GMAIL_CLIENT_SECRET": "test-client-secret",
			"EMAIL_GMAIL_REDIRECT_URL":  "http://localhost:8080/callback",
		},
	}

	loader := NewOAuthConfigLoader(osw)
	cfg, err := loader.GetGmailOAuthConfig(context.Background())
	require.NoError(t, err)
	require.Equal(t, "test-client-id", cfg.ClientID)
	require.Equal(t, "test-client-secret", cfg.ClientSecret)
	require.Equal(t, "http://localhost:8080/callback", cfg.RedirectURL)
}

func TestOAuthConfigLoader_MissingClientIDReturnsError(t *testing.T) {
	t.Parallel()

	osw := &oauthStubOsWrapper{
		env: map[string]string{
			"EMAIL_GMAIL_CLIENT_SECRET": "inline-secret",
			"EMAIL_GMAIL_REDIRECT_URL":  "http://localhost:8080/callback",
		},
	}

	loader := NewOAuthConfigLoader(osw)
	_, err := loader.GetGmailOAuthConfig(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "EMAIL_GMAIL_CLIENT_ID")
}

func TestOAuthConfigLoader_MissingClientSecretReturnsError(t *testing.T) {
	t.Parallel()

	osw := &oauthStubOsWrapper{
		env: map[string]string{
			"EMAIL_GMAIL_CLIENT_ID":    "inline-client-id",
			"EMAIL_GMAIL_REDIRECT_URL": "http://localhost:8080/callback",
		},
	}

	loader := NewOAuthConfigLoader(osw)
	_, err := loader.GetGmailOAuthConfig(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "EMAIL_GMAIL_CLIENT_SECRET")
}
