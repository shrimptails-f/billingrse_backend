package infrastructure

import (
	"context"
	"strings"
	"testing"
	"time"

	"business/internal/auth/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRepository_CreateEmailVerificationToken_Success(t *testing.T) {
	t.Parallel()
	env := newAuthRepoTestEnv(t)
	defer func() { _ = env.clean() }()

	env.insertUser(t, "token@example.com")
	var userRec userRecord
	err := env.db.Where("email = ?", "token@example.com").First(&userRec).Error
	require.NoError(t, err)

	expiresAt := env.nowUTC.Add(3 * time.Hour)
	token := domain.EmailVerificationToken{
		UserID:    userRec.ID,
		Token:     "test-token-uuid",
		ExpiresAt: expiresAt,
		CreatedAt: env.nowUTC,
	}

	created, err := env.repo.CreateEmailVerificationToken(context.Background(), token)

	assert.NoError(t, err)
	assert.NotZero(t, created.ID)
	assert.Equal(t, userRec.ID, created.UserID)
	assert.Equal(t, "test-token-uuid", created.Token)
	assert.WithinDuration(t, expiresAt, created.ExpiresAt, time.Second)
}

func TestRepository_InvalidateActiveTokens_Success(t *testing.T) {
	t.Parallel()
	env := newAuthRepoTestEnv(t)
	defer func() { _ = env.clean() }()

	env.insertUser(t, "invalidate@example.com")
	var userRec userRecord
	err := env.db.Where("email = ?", "invalidate@example.com").First(&userRec).Error
	require.NoError(t, err)

	// Create 3 active tokens and 1 already consumed token
	for i := 0; i < 3; i++ {
		err = env.db.Create(&emailVerificationTokenRecord{
			UserID:    userRec.ID,
			Token:     "active-token-" + string(rune('1'+i)),
			ExpiresAt: env.nowUTC.Add(3 * time.Hour),
			CreatedAt: env.nowUTC,
		}).Error
		require.NoError(t, err)
	}

	consumedTime := env.nowUTC.Add(-1 * time.Hour)
	err = env.db.Create(&emailVerificationTokenRecord{
		UserID:     userRec.ID,
		Token:      "consumed-token",
		ExpiresAt:  env.nowUTC.Add(2 * time.Hour),
		CreatedAt:  env.nowUTC.Add(-2 * time.Hour),
		ConsumedAt: &consumedTime,
	}).Error
	require.NoError(t, err)

	invalidateTime := env.nowUTC
	err = env.repo.InvalidateActiveTokens(context.Background(), userRec.ID, invalidateTime)
	assert.NoError(t, err)

	// Check that 3 active tokens are now consumed
	var tokens []emailVerificationTokenRecord
	err = env.db.Where("user_id = ?", userRec.ID).Find(&tokens).Error
	require.NoError(t, err)
	assert.Len(t, tokens, 4)

	activeConsumed := 0
	for _, tk := range tokens {
		if strings.HasPrefix(tk.Token, "active-token-") {
			assert.NotNil(t, tk.ConsumedAt)
			assert.WithinDuration(t, invalidateTime, *tk.ConsumedAt, time.Second)
			activeConsumed++
		} else if tk.Token == "consumed-token" {
			assert.WithinDuration(t, consumedTime, *tk.ConsumedAt, time.Second)
		}
	}
	assert.Equal(t, 3, activeConsumed)
}

func TestRepository_GetEmailVerificationToken_Success(t *testing.T) {
	t.Parallel()
	env := newAuthRepoTestEnv(t)
	defer func() { _ = env.clean() }()

	env.insertUser(t, "gettoken@example.com")
	var userRec userRecord
	err := env.db.Where("email = ?", "gettoken@example.com").First(&userRec).Error
	require.NoError(t, err)

	expiresAt := env.nowUTC.Add(3 * time.Hour)
	err = env.db.Create(&emailVerificationTokenRecord{
		UserID:    userRec.ID,
		Token:     "get-me",
		ExpiresAt: expiresAt,
		CreatedAt: env.nowUTC,
	}).Error
	require.NoError(t, err)

	token, err := env.repo.GetEmailVerificationToken(context.Background(), "get-me")

	assert.NoError(t, err)
	assert.Equal(t, "get-me", token.Token)
	assert.Equal(t, userRec.ID, token.UserID)
	assert.WithinDuration(t, expiresAt, token.ExpiresAt, time.Second)
	assert.Nil(t, token.ConsumedAt)
}

func TestRepository_GetEmailVerificationToken_NotFound(t *testing.T) {
	t.Parallel()
	env := newAuthRepoTestEnv(t)
	defer func() { _ = env.clean() }()

	_, err := env.repo.GetEmailVerificationToken(context.Background(), "nonexistent")

	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestRepository_ConsumeTokenAndVerifyUser_Success(t *testing.T) {
	t.Parallel()
	env := newAuthRepoTestEnv(t)
	defer func() { _ = env.clean() }()

	env.insertUser(t, "consume@example.com")
	var userRec userRecord
	err := env.db.Where("email = ?", "consume@example.com").First(&userRec).Error
	require.NoError(t, err)

	var tokenRec emailVerificationTokenRecord
	err = env.db.Create(&emailVerificationTokenRecord{
		UserID:    userRec.ID,
		Token:     "consume-token",
		ExpiresAt: env.nowUTC.Add(3 * time.Hour),
		CreatedAt: env.nowUTC,
	}).Error
	require.NoError(t, err)
	err = env.db.Where("token = ?", "consume-token").First(&tokenRec).Error
	require.NoError(t, err)

	consumedAt := env.nowUTC.Add(1 * time.Minute)
	user, err := env.repo.ConsumeTokenAndVerifyUser(context.Background(), tokenRec.ID, userRec.ID, consumedAt)

	assert.NoError(t, err)
	assert.True(t, user.EmailVerified)
	assert.NotNil(t, user.EmailVerifiedAt)
	assert.WithinDuration(t, consumedAt, *user.EmailVerifiedAt, time.Second)

	// Verify token is consumed
	var updatedToken emailVerificationTokenRecord
	err = env.db.Where("id = ?", tokenRec.ID).First(&updatedToken).Error
	require.NoError(t, err)
	assert.NotNil(t, updatedToken.ConsumedAt)
	assert.WithinDuration(t, consumedAt, *updatedToken.ConsumedAt, time.Second)

	// Verify user is verified
	var updatedUser userRecord
	err = env.db.Where("id = ?", userRec.ID).First(&updatedUser).Error
	require.NoError(t, err)
	assert.True(t, updatedUser.EmailVerified)
	assert.NotNil(t, updatedUser.EmailVerifiedAt)
}

func TestRepository_GetLatestTokenForUser_Success(t *testing.T) {
	t.Parallel()
	env := newAuthRepoTestEnv(t)
	defer func() { _ = env.clean() }()

	env.insertUser(t, "latest@example.com")
	var userRec userRecord
	err := env.db.Where("email = ?", "latest@example.com").First(&userRec).Error
	require.NoError(t, err)

	// Create 3 tokens at different times
	oldTime := env.nowUTC.Add(-2 * time.Hour)
	middleTime := env.nowUTC.Add(-1 * time.Hour)

	err = env.db.Create(&emailVerificationTokenRecord{
		UserID:    userRec.ID,
		Token:     "old-token",
		ExpiresAt: env.nowUTC.Add(1 * time.Hour),
		CreatedAt: oldTime,
	}).Error
	require.NoError(t, err)

	err = env.db.Create(&emailVerificationTokenRecord{
		UserID:    userRec.ID,
		Token:     "middle-token",
		ExpiresAt: env.nowUTC.Add(2 * time.Hour),
		CreatedAt: middleTime,
	}).Error
	require.NoError(t, err)

	err = env.db.Create(&emailVerificationTokenRecord{
		UserID:    userRec.ID,
		Token:     "latest-token",
		ExpiresAt: env.nowUTC.Add(3 * time.Hour),
		CreatedAt: env.nowUTC,
	}).Error
	require.NoError(t, err)

	token, err := env.repo.GetLatestTokenForUser(context.Background(), userRec.ID)

	assert.NoError(t, err)
	assert.Equal(t, "latest-token", token.Token)
	assert.WithinDuration(t, env.nowUTC, token.CreatedAt, time.Second)
}

func TestRepository_GetLatestTokenForUser_NotFound(t *testing.T) {
	t.Parallel()
	env := newAuthRepoTestEnv(t)
	defer func() { _ = env.clean() }()

	_, err := env.repo.GetLatestTokenForUser(context.Background(), 99999)

	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestRepository_DeleteTokenByID_Success(t *testing.T) {
	t.Parallel()
	env := newAuthRepoTestEnv(t)
	defer func() { _ = env.clean() }()

	env.insertUser(t, "delete@example.com")
	var userRec userRecord
	err := env.db.Where("email = ?", "delete@example.com").First(&userRec).Error
	require.NoError(t, err)

	var tokenRec emailVerificationTokenRecord
	err = env.db.Create(&emailVerificationTokenRecord{
		UserID:    userRec.ID,
		Token:     "delete-me",
		ExpiresAt: env.nowUTC.Add(3 * time.Hour),
		CreatedAt: env.nowUTC,
	}).Error
	require.NoError(t, err)
	err = env.db.Where("token = ?", "delete-me").First(&tokenRec).Error
	require.NoError(t, err)

	err = env.repo.DeleteTokenByID(context.Background(), tokenRec.ID)
	assert.NoError(t, err)

	// Verify deletion
	var count int64
	err = env.db.Model(&emailVerificationTokenRecord{}).Where("id = ?", tokenRec.ID).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestRepository_GetActiveTokenForUser_Success(t *testing.T) {
	t.Parallel()
	env := newAuthRepoTestEnv(t)
	defer func() { _ = env.clean() }()

	env.insertUser(t, "active@example.com")
	var userRec userRecord
	err := env.db.Where("email = ?", "active@example.com").First(&userRec).Error
	require.NoError(t, err)

	// Create an active token
	expiresAt := env.nowUTC.Add(3 * time.Hour)
	err = env.db.Create(&emailVerificationTokenRecord{
		UserID:    userRec.ID,
		Token:     "active-token",
		ExpiresAt: expiresAt,
		CreatedAt: env.nowUTC,
	}).Error
	require.NoError(t, err)

	token, err := env.repo.GetActiveTokenForUser(context.Background(), userRec.ID, env.nowUTC)

	assert.NoError(t, err)
	assert.Equal(t, "active-token", token.Token)
	assert.Equal(t, userRec.ID, token.UserID)
	assert.Nil(t, token.ConsumedAt)
}

func TestRepository_GetActiveTokenForUser_NotFound(t *testing.T) {
	t.Parallel()
	env := newAuthRepoTestEnv(t)
	defer func() { _ = env.clean() }()

	_, err := env.repo.GetActiveTokenForUser(context.Background(), 99999, env.nowUTC)

	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestRepository_GetActiveTokenForUser_ExpiredToken(t *testing.T) {
	t.Parallel()
	env := newAuthRepoTestEnv(t)
	defer func() { _ = env.clean() }()

	env.insertUser(t, "expired@example.com")
	var userRec userRecord
	err := env.db.Where("email = ?", "expired@example.com").First(&userRec).Error
	require.NoError(t, err)

	// Create an expired token
	expiresAt := env.nowUTC.Add(-1 * time.Hour)
	err = env.db.Create(&emailVerificationTokenRecord{
		UserID:    userRec.ID,
		Token:     "expired-token",
		ExpiresAt: expiresAt,
		CreatedAt: env.nowUTC.Add(-4 * time.Hour),
	}).Error
	require.NoError(t, err)

	_, err = env.repo.GetActiveTokenForUser(context.Background(), userRec.ID, env.nowUTC)

	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestRepository_GetActiveTokenForUser_ConsumedToken(t *testing.T) {
	t.Parallel()
	env := newAuthRepoTestEnv(t)
	defer func() { _ = env.clean() }()

	env.insertUser(t, "consumed@example.com")
	var userRec userRecord
	err := env.db.Where("email = ?", "consumed@example.com").First(&userRec).Error
	require.NoError(t, err)

	// Create a consumed token
	consumedAt := env.nowUTC.Add(-1 * time.Hour)
	expiresAt := env.nowUTC.Add(2 * time.Hour)
	err = env.db.Create(&emailVerificationTokenRecord{
		UserID:     userRec.ID,
		Token:      "consumed-token",
		ExpiresAt:  expiresAt,
		CreatedAt:  env.nowUTC.Add(-2 * time.Hour),
		ConsumedAt: &consumedAt,
	}).Error
	require.NoError(t, err)

	_, err = env.repo.GetActiveTokenForUser(context.Background(), userRec.ID, env.nowUTC)

	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestRepository_CreateEmailVerificationToken_Upsert(t *testing.T) {
	t.Parallel()
	env := newAuthRepoTestEnv(t)
	defer func() { _ = env.clean() }()

	env.insertUser(t, "upsert@example.com")
	var userRec userRecord
	err := env.db.Where("email = ?", "upsert@example.com").First(&userRec).Error
	require.NoError(t, err)

	// Mirror production schema where user_id has a unique constraint so upsert logic triggers.
	err = env.db.Exec("CREATE UNIQUE INDEX idx_user_id_unique ON email_verification_tokens (user_id)").Error
	require.NoError(t, err)

	// Create first token
	expiresAt1 := env.nowUTC.Add(3 * time.Hour)
	token1 := domain.EmailVerificationToken{
		UserID:    userRec.ID,
		Token:     "first-token",
		ExpiresAt: expiresAt1,
		CreatedAt: env.nowUTC,
	}

	created1, err := env.repo.CreateEmailVerificationToken(context.Background(), token1)
	assert.NoError(t, err)
	assert.Equal(t, "first-token", created1.Token)

	// Verify only one token exists
	var count1 int64
	err = env.db.Model(&emailVerificationTokenRecord{}).Where("user_id = ?", userRec.ID).Count(&count1).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count1)

	// Create second token for same user (should replace first)
	expiresAt2 := env.nowUTC.Add(3 * time.Hour)
	token2 := domain.EmailVerificationToken{
		UserID:    userRec.ID,
		Token:     "second-token",
		ExpiresAt: expiresAt2,
		CreatedAt: env.nowUTC,
	}

	created2, err := env.repo.CreateEmailVerificationToken(context.Background(), token2)
	assert.NoError(t, err)
	assert.Equal(t, "second-token", created2.Token)

	// Verify still only one token exists
	var count2 int64
	err = env.db.Model(&emailVerificationTokenRecord{}).Where("user_id = ?", userRec.ID).Count(&count2).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count2)

	// Verify the token is the new one
	var record emailVerificationTokenRecord
	err = env.db.Where("user_id = ?", userRec.ID).First(&record).Error
	require.NoError(t, err)
	assert.Equal(t, "second-token", record.Token)
}
