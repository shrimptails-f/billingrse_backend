package gmailService

import (
	"context"
	"testing"

	mocklibrary "business/test/mock/library"

	"github.com/stretchr/testify/require"
)

func TestOAuthConfigLoader_UsesInlineClientIDAndSecret(t *testing.T) {
	t.Parallel()

	osw := mocklibrary.NewOsWrapperMock(map[string]string{
		"EMAIL_GMAIL_CLIENT_ID":     "test-client-id",
		"EMAIL_GMAIL_CLIENT_SECRET": "test-client-secret",
		"EMAIL_GMAIL_REDIRECT_URL":  "http://localhost:8080/callback",
	})

	loader := NewOAuthConfigLoader(osw)
	cfg, err := loader.GetGmailOAuthConfig(context.Background())
	require.NoError(t, err)
	require.Equal(t, "test-client-id", cfg.ClientID)
	require.Equal(t, "test-client-secret", cfg.ClientSecret)
	require.Equal(t, "http://localhost:8080/callback", cfg.RedirectURL)
}

func TestOAuthConfigLoader_MissingClientIDReturnsError(t *testing.T) {
	t.Parallel()

	osw := mocklibrary.NewOsWrapperMock(map[string]string{
		"EMAIL_GMAIL_CLIENT_SECRET": "inline-secret",
		"EMAIL_GMAIL_REDIRECT_URL":  "http://localhost:8080/callback",
	})

	loader := NewOAuthConfigLoader(osw)
	_, err := loader.GetGmailOAuthConfig(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "EMAIL_GMAIL_CLIENT_ID")
}

func TestOAuthConfigLoader_MissingClientSecretReturnsError(t *testing.T) {
	t.Parallel()

	osw := mocklibrary.NewOsWrapperMock(map[string]string{
		"EMAIL_GMAIL_CLIENT_ID":    "inline-client-id",
		"EMAIL_GMAIL_REDIRECT_URL": "http://localhost:8080/callback",
	})

	loader := NewOAuthConfigLoader(osw)
	_, err := loader.GetGmailOAuthConfig(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "EMAIL_GMAIL_CLIENT_SECRET")
}
