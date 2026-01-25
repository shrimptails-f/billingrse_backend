package gmailService

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestCreateServiceWithToken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Valid OAuth client credentials JSON
	validCredentials := `{
        "installed": {
            "client_id": "test-client-id.apps.googleusercontent.com",
			"project_id": "test-project",
			"auth_uri": "https://accounts.google.com/o/oauth2/auth",
			"token_uri": "https://oauth2.googleapis.com/token",
			"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
			"client_secret": "test-secret",
			"redirect_uris": ["http://localhost"]
		}
	}`

	invalidCredentials := `{ "invalid": "json structure" }`

	t.Run("success with valid credentials and token", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		credPath := filepath.Join(tmpDir, "credentials.json")
		err := os.WriteFile(credPath, []byte(validCredentials), 0600)
		require.NoError(t, err)

		token := &oauth2.Token{
			AccessToken:  "test-access-token",
			RefreshToken: "test-refresh-token",
			Expiry:       time.Now().Add(1 * time.Hour),
		}

		client := New()
		svc, err := client.CreateServiceWithToken(ctx, credPath, token)

		// Note: This will succeed in creating a service object even with fake credentials
		// because the Gmail API client doesn't validate credentials until actual API calls
		require.NoError(t, err)
		assert.NotNil(t, svc)
	})

	t.Run("error when credentials file does not exist", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		nonExistentPath := filepath.Join(tmpDir, "nonexistent.json")
		token := &oauth2.Token{
			AccessToken:  "test-access-token",
			RefreshToken: "test-refresh-token",
		}

		client := New()
		svc, err := client.CreateServiceWithToken(ctx, nonExistentPath, token)

		require.Error(t, err)
		assert.Nil(t, svc)
		assert.Contains(t, err.Error(), "クレデンシャル読み込み失敗")
	})

	t.Run("error when credentials file is invalid JSON", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		invalidPath := filepath.Join(tmpDir, "invalid.json")
		err := os.WriteFile(invalidPath, []byte(invalidCredentials), 0600)
		require.NoError(t, err)

		token := &oauth2.Token{
			AccessToken:  "test-access-token",
			RefreshToken: "test-refresh-token",
		}

		client := New()
		svc, err := client.CreateServiceWithToken(ctx, invalidPath, token)

		require.Error(t, err)
		assert.Nil(t, svc)
		assert.Contains(t, err.Error(), "OAuth2構成失敗")
	})

	t.Run("success with token without expiry", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		credPath := filepath.Join(tmpDir, "credentials.json")
		err := os.WriteFile(credPath, []byte(validCredentials), 0600)
		require.NoError(t, err)

		token := &oauth2.Token{
			AccessToken:  "test-access-token",
			RefreshToken: "test-refresh-token",
			// No Expiry set
		}

		client := New()
		svc, err := client.CreateServiceWithToken(ctx, credPath, token)

		require.NoError(t, err)
		assert.NotNil(t, svc)
	})
}
