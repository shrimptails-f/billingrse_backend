package crypto

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVault(t *testing.T) {
	t.Parallel()
	t.Run("success with valid parameters", func(t *testing.T) {
		t.Parallel()
		cfg := VaultConfig{
			KeyMaterial: []byte("this-is-a-32-byte-key-material!!"),
			Salt:        []byte("test-salt"),
			Info:        "test-vault",
		}

		vault, err := NewVault(cfg)

		require.NoError(t, err)
		assert.NotNil(t, vault)
	})

	t.Run("error when key material is too short", func(t *testing.T) {
		t.Parallel()
		cfg := VaultConfig{
			KeyMaterial: []byte("short-key"),
			Salt:        []byte("test-salt"),
			Info:        "test-vault",
		}

		vault, err := NewVault(cfg)

		assert.Error(t, err)
		assert.Nil(t, vault)
		assert.Contains(t, err.Error(), "key material must be at least 32 bytes")
	})

	t.Run("error when salt is empty", func(t *testing.T) {
		t.Parallel()
		cfg := VaultConfig{
			KeyMaterial: []byte("this-is-a-32-byte-key-material!!"),
			Salt:        []byte{},
			Info:        "test-vault",
		}

		vault, err := NewVault(cfg)

		assert.Error(t, err)
		assert.Nil(t, vault)
		assert.Contains(t, err.Error(), "salt must not be empty")
	})

	t.Run("error when info is empty", func(t *testing.T) {
		t.Parallel()
		cfg := VaultConfig{
			KeyMaterial: []byte("this-is-a-32-byte-key-material!!"),
			Salt:        []byte("test-salt"),
			Info:        "",
		}

		vault, err := NewVault(cfg)

		assert.Error(t, err)
		assert.Nil(t, vault)
		assert.Contains(t, err.Error(), "info must not be empty")
	})
}

func TestEncryptDecrypt(t *testing.T) {
	t.Parallel()
	cfg := VaultConfig{
		KeyMaterial: []byte("this-is-a-32-byte-key-material!!"),
		Salt:        []byte("test-salt"),
		Info:        "test-vault",
	}
	vault, err := NewVault(cfg)
	require.NoError(t, err)

	t.Run("encrypt and decrypt returns original plaintext", func(t *testing.T) {
		t.Parallel()
		plaintext := "sk-test-token-1234567890"

		ciphertext, err := vault.Encrypt(plaintext)
		require.NoError(t, err)
		assert.NotEmpty(t, ciphertext)
		assert.NotEqual(t, plaintext, string(ciphertext))

		decrypted, err := vault.Decrypt(ciphertext)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("encrypt produces different ciphertext for same plaintext", func(t *testing.T) {
		t.Parallel()
		plaintext := "sk-test-token-1234567890"

		ciphertext1, err := vault.Encrypt(plaintext)
		require.NoError(t, err)

		ciphertext2, err := vault.Encrypt(plaintext)
		require.NoError(t, err)

		// Different nonces should produce different ciphertexts
		assert.NotEqual(t, ciphertext1, ciphertext2)

		// But both should decrypt to the same plaintext
		decrypted1, err := vault.Decrypt(ciphertext1)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted1)

		decrypted2, err := vault.Decrypt(ciphertext2)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted2)
	})

	t.Run("decrypt with tampered ciphertext fails", func(t *testing.T) {
		t.Parallel()
		plaintext := "sk-test-token-1234567890"

		ciphertext, err := vault.Encrypt(plaintext)
		require.NoError(t, err)

		// Tamper with the ciphertext
		ciphertext[len(ciphertext)-1] ^= 0xFF

		decrypted, err := vault.Decrypt(ciphertext)
		assert.Error(t, err)
		assert.Empty(t, decrypted)
		assert.Contains(t, err.Error(), "failed to decrypt")
	})

	t.Run("decrypt with short ciphertext fails", func(t *testing.T) {
		t.Parallel()
		shortCiphertext := []byte("short")

		decrypted, err := vault.Decrypt(shortCiphertext)
		assert.Error(t, err)
		assert.Empty(t, decrypted)
		assert.Contains(t, err.Error(), "ciphertext too short")
	})

	t.Run("encrypt long token", func(t *testing.T) {
		t.Parallel()
		plaintext := strings.Repeat("a", 1000)

		ciphertext, err := vault.Encrypt(plaintext)
		require.NoError(t, err)

		decrypted, err := vault.Decrypt(ciphertext)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})
}

func TestEncryptToStringDecryptFromString(t *testing.T) {
	t.Parallel()
	cfg := VaultConfig{
		KeyMaterial: []byte("this-is-a-32-byte-key-material!!"),
		Salt:        []byte("test-salt"),
		Info:        "test-vault",
	}
	vault, err := NewVault(cfg)
	require.NoError(t, err)

	t.Run("encrypt to string and decrypt from string", func(t *testing.T) {
		t.Parallel()
		plaintext := "ya29.test-access-token"

		ciphertext, err := vault.EncryptToString(plaintext)
		require.NoError(t, err)
		assert.NotEmpty(t, ciphertext)
		assert.NotEqual(t, plaintext, ciphertext)

		decrypted, err := vault.DecryptFromString(ciphertext)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("decrypt from string with invalid base64 fails", func(t *testing.T) {
		t.Parallel()
		invalidBase64 := "not-valid-base64!!!"

		decrypted, err := vault.DecryptFromString(invalidBase64)
		assert.Error(t, err)
		assert.Empty(t, decrypted)
		assert.Contains(t, err.Error(), "failed to decode base64")
	})
}

func TestDigest(t *testing.T) {
	t.Parallel()
	cfg := VaultConfig{
		KeyMaterial: []byte("this-is-a-32-byte-key-material!!"),
		Salt:        []byte("test-salt"),
		Info:        "test-vault",
	}
	vault, err := NewVault(cfg)
	require.NoError(t, err)

	t.Run("digest generates consistent hash", func(t *testing.T) {
		t.Parallel()
		plaintext := "sk-test-token-1234567890"

		digest1, err := vault.Digest(plaintext)
		require.NoError(t, err)
		assert.NotEmpty(t, digest1)
		assert.Equal(t, 32, len(digest1)) // SHA-256 produces 32 bytes

		digest2, err := vault.Digest(plaintext)
		require.NoError(t, err)

		// Same plaintext should produce same digest
		assert.Equal(t, digest1, digest2)
	})

	t.Run("different plaintexts produce different digests", func(t *testing.T) {
		t.Parallel()
		plaintext1 := "sk-test-token-1234567890"
		plaintext2 := "sk-test-token-0987654321"

		digest1, err := vault.Digest(plaintext1)
		require.NoError(t, err)

		digest2, err := vault.Digest(plaintext2)
		require.NoError(t, err)

		assert.NotEqual(t, digest1, digest2)
	})

	t.Run("digest with different salt produces different result", func(t *testing.T) {
		t.Parallel()
		plaintext := "sk-test-token-1234567890"

		cfg1 := VaultConfig{
			KeyMaterial: []byte("this-is-a-32-byte-key-material!!"),
			Salt:        []byte("salt1"),
			Info:        "test-vault",
		}
		vault1, err := NewVault(cfg1)
		require.NoError(t, err)

		cfg2 := VaultConfig{
			KeyMaterial: []byte("this-is-a-32-byte-key-material!!"),
			Salt:        []byte("salt2"),
			Info:        "test-vault",
		}
		vault2, err := NewVault(cfg2)
		require.NoError(t, err)

		digest1, err := vault1.Digest(plaintext)
		require.NoError(t, err)

		digest2, err := vault2.Digest(plaintext)
		require.NoError(t, err)

		assert.NotEqual(t, digest1, digest2)
	})
}

func TestDigestToString(t *testing.T) {
	t.Parallel()
	cfg := VaultConfig{
		KeyMaterial: []byte("this-is-a-32-byte-key-material!!"),
		Salt:        []byte("test-salt"),
		Info:        "test-vault",
	}
	vault, err := NewVault(cfg)
	require.NoError(t, err)

	t.Run("digest to string returns base64", func(t *testing.T) {
		t.Parallel()
		plaintext := "ya29.test-access-token"

		digestStr, err := vault.DigestToString(plaintext)
		require.NoError(t, err)
		assert.NotEmpty(t, digestStr)
		assert.NotEqual(t, plaintext, digestStr)

		// Verify consistency
		digestStr2, err := vault.DigestToString(plaintext)
		require.NoError(t, err)
		assert.Equal(t, digestStr, digestStr2)
	})
}

func TestIntegration(t *testing.T) {
	t.Parallel()
	cfg := VaultConfig{
		KeyMaterial: []byte("this-is-a-32-byte-key-material!!"),
		Salt:        []byte("test-salt"),
		Info:        "email-credential-encryption",
	}
	vault, err := NewVault(cfg)
	require.NoError(t, err)

	t.Run("full workflow with access_token and refresh_token", func(t *testing.T) {
		t.Parallel()
		accessToken := "ya29.test-access-token"
		refreshToken := "1//test-refresh-token"

		// Encrypt both (as string for DB storage)
		encAccessToken, err := vault.EncryptToString(accessToken)
		require.NoError(t, err)

		encRefreshToken, err := vault.EncryptToString(refreshToken)
		require.NoError(t, err)

		// Generate digests (as string for DB storage)
		accessTokenDigest, err := vault.DigestToString(accessToken)
		require.NoError(t, err)

		refreshTokenDigest, err := vault.DigestToString(refreshToken)
		require.NoError(t, err)

		// Verify encrypted values are different from plaintext
		assert.NotEqual(t, accessToken, encAccessToken)
		assert.NotEqual(t, refreshToken, encRefreshToken)
		assert.NotEqual(t, accessToken, accessTokenDigest)
		assert.NotEqual(t, refreshToken, refreshTokenDigest)

		// Decrypt and verify
		decAccessToken, err := vault.DecryptFromString(encAccessToken)
		require.NoError(t, err)
		assert.Equal(t, accessToken, decAccessToken)

		decRefreshToken, err := vault.DecryptFromString(encRefreshToken)
		require.NoError(t, err)
		assert.Equal(t, refreshToken, decRefreshToken)

		// Verify digests match
		accessTokenDigest2, err := vault.DigestToString(accessToken)
		require.NoError(t, err)
		assert.Equal(t, accessTokenDigest, accessTokenDigest2)

		refreshTokenDigest2, err := vault.DigestToString(refreshToken)
		require.NoError(t, err)
		assert.Equal(t, refreshTokenDigest, refreshTokenDigest2)
	})
}
