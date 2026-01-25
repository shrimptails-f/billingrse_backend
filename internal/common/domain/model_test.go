package domain

import (
	"crypto/aes"
	"crypto/cipher"
	"testing"
)

func TestAgentDecryptMethods(t *testing.T) {
	t.Parallel()

	key := []byte("0123456789abcdef0123456789abcdef") // 32 bytes
	tokenCipher := mustEncryptWithKey(t, key, []byte("agent-token"))
	refreshCipher := mustEncryptWithKey(t, key, []byte("refresh-token"))

	t.Run("DecryptToken succeeds", func(t *testing.T) {
		t.Parallel()
		agent := &Agent{Token: tokenCipher}

		plaintext, err := agent.DecryptToken(key)
		if err != nil {
			t.Fatalf("DecryptToken returned error: %v", err)
		}
		if plaintext != "agent-token" {
			t.Fatalf("want agent-token, got %s", plaintext)
		}
	})

	t.Run("DecryptRefreshToken succeeds", func(t *testing.T) {
		t.Parallel()
		agent := &Agent{RefreshToken: refreshCipher}

		plaintext, err := agent.DecryptRefreshToken(key)
		if err != nil {
			t.Fatalf("DecryptRefreshToken returned error: %v", err)
		}
		if plaintext != "refresh-token" {
			t.Fatalf("want refresh-token, got %s", plaintext)
		}
	})

	t.Run("empty cipher returns error", func(t *testing.T) {
		t.Parallel()
		agent := &Agent{}

		if _, err := agent.DecryptToken(key); err == nil {
			t.Fatalf("expected error for empty token")
		}
		if _, err := agent.DecryptRefreshToken(key); err == nil {
			t.Fatalf("expected error for empty refresh token")
		}
	})
}

func mustEncryptWithKey(t *testing.T, key []byte, plaintext []byte) []byte {
	t.Helper()

	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("failed to create cipher: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("failed to create gcm: %v", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	return gcm.Seal(nonce, nonce, plaintext, nil)
}
