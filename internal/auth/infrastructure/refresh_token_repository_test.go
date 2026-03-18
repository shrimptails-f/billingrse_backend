package infrastructure

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"business/internal/auth/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func refreshTokenDigestForTest(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func TestRepository_CreateRefreshToken_Success(t *testing.T) {
	t.Parallel()

	env := newAuthRepoTestEnv(t)
	defer func() { _ = env.clean() }()

	env.insertUser(t, "refresh-create@example.com")
	var userRec userRecord
	require.NoError(t, env.db.Where("email = ?", "refresh-create@example.com").First(&userRec).Error)

	rawToken := "refresh-token"
	now := env.nowUTC
	created, err := env.repo.CreateRefreshToken(context.Background(), domain.RefreshToken{
		UserID:      userRec.ID,
		Token:       rawToken,
		TokenDigest: refreshTokenDigestForTest(rawToken),
		ExpiresAt:   now.Add(30 * 24 * time.Hour),
		CreatedAt:   now,
	})

	assert.NoError(t, err)
	assert.NotZero(t, created.ID)
	assert.Equal(t, rawToken, created.Token)
	assert.Equal(t, userRec.ID, created.UserID)
}

func TestRepository_FindActiveRefreshTokenByDigest_Success(t *testing.T) {
	t.Parallel()

	env := newAuthRepoTestEnv(t)
	defer func() { _ = env.clean() }()

	env.insertUser(t, "refresh-find@example.com")
	var userRec userRecord
	require.NoError(t, env.db.Where("email = ?", "refresh-find@example.com").First(&userRec).Error)

	rawToken := "refresh-token"
	expiresAt := env.nowUTC.Add(30 * 24 * time.Hour)
	require.NoError(t, env.db.Create(&refreshTokenRecord{
		UserID:      userRec.ID,
		TokenDigest: refreshTokenDigestForTest(rawToken),
		ExpiresAt:   expiresAt,
		CreatedAt:   env.nowUTC,
	}).Error)

	token, err := env.repo.FindActiveRefreshTokenByDigest(context.Background(), refreshTokenDigestForTest(rawToken), env.nowUTC)

	assert.NoError(t, err)
	assert.Equal(t, userRec.ID, token.UserID)
	assert.WithinDuration(t, expiresAt, token.ExpiresAt, time.Second)
}

func TestRepository_FindActiveRefreshTokenByDigest_NotFound(t *testing.T) {
	t.Parallel()

	env := newAuthRepoTestEnv(t)
	defer func() { _ = env.clean() }()

	_, err := env.repo.FindActiveRefreshTokenByDigest(context.Background(), "missing", env.nowUTC)

	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestRepository_RotateRefreshToken_Success(t *testing.T) {
	t.Parallel()

	env := newAuthRepoTestEnv(t)
	defer func() { _ = env.clean() }()

	env.insertUser(t, "refresh-rotate@example.com")
	var userRec userRecord
	require.NoError(t, env.db.Where("email = ?", "refresh-rotate@example.com").First(&userRec).Error)

	currentRaw := "current-refresh-token"
	currentDigest := refreshTokenDigestForTest(currentRaw)
	require.NoError(t, env.db.Create(&refreshTokenRecord{
		UserID:      userRec.ID,
		TokenDigest: currentDigest,
		ExpiresAt:   env.nowUTC.Add(30 * 24 * time.Hour),
		CreatedAt:   env.nowUTC,
	}).Error)

	var currentRecord refreshTokenRecord
	require.NoError(t, env.db.Where("token_digest = ?", currentDigest).First(&currentRecord).Error)

	nextRaw := "next-refresh-token"
	nextDigest := refreshTokenDigestForTest(nextRaw)
	rotated, err := env.repo.RotateRefreshToken(context.Background(), currentRecord.ID, domain.RefreshToken{
		UserID:      userRec.ID,
		Token:       nextRaw,
		TokenDigest: nextDigest,
		ExpiresAt:   env.nowUTC.Add(30 * 24 * time.Hour),
		CreatedAt:   env.nowUTC,
	}, env.nowUTC)

	assert.NoError(t, err)
	assert.Equal(t, nextRaw, rotated.Token)
	assert.NotZero(t, rotated.ID)

	var oldRecord refreshTokenRecord
	require.NoError(t, env.db.Where("id = ?", currentRecord.ID).First(&oldRecord).Error)
	require.NotNil(t, oldRecord.RevokedAt)
	require.NotNil(t, oldRecord.LastUsedAt)
	require.NotNil(t, oldRecord.ReplacedByTokenID)
	assert.Equal(t, rotated.ID, *oldRecord.ReplacedByTokenID)

	var newRecord refreshTokenRecord
	require.NoError(t, env.db.Where("id = ?", rotated.ID).First(&newRecord).Error)
	assert.Equal(t, nextDigest, newRecord.TokenDigest)
}

func TestRepository_RevokeRefreshTokenByDigest_Success(t *testing.T) {
	t.Parallel()

	env := newAuthRepoTestEnv(t)
	defer func() { _ = env.clean() }()

	env.insertUser(t, "refresh-revoke@example.com")
	var userRec userRecord
	require.NoError(t, env.db.Where("email = ?", "refresh-revoke@example.com").First(&userRec).Error)

	rawToken := "refresh-token"
	digest := refreshTokenDigestForTest(rawToken)
	require.NoError(t, env.db.Create(&refreshTokenRecord{
		UserID:      userRec.ID,
		TokenDigest: digest,
		ExpiresAt:   env.nowUTC.Add(30 * 24 * time.Hour),
		CreatedAt:   env.nowUTC,
	}).Error)

	require.NoError(t, env.repo.RevokeRefreshTokenByDigest(context.Background(), digest, env.nowUTC))

	var record refreshTokenRecord
	require.NoError(t, env.db.Where("token_digest = ?", digest).First(&record).Error)
	require.NotNil(t, record.RevokedAt)
	require.NotNil(t, record.LastUsedAt)
}
