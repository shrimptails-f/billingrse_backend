package infrastructure

import (
	"business/internal/library/logger"
	"business/internal/library/mysql"
	"business/internal/mailaccountconnection/domain"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type repoTestEnv struct {
	repo   *Repository
	db     *gorm.DB
	clean  func() error
	nowUTC time.Time
}

func newRepoTestEnv(t *testing.T) *repoTestEnv {
	t.Helper()

	mysqlConn, cleanup, err := mysql.CreateNewTestDB()
	if err != nil {
		skipIfDBUnavailable(t, err)
	}
	require.NoError(t, err)

	err = mysqlConn.DB.AutoMigrate(&credentialRecord{})
	require.NoError(t, err)

	return &repoTestEnv{
		repo:   NewRepository(mysqlConn.DB, logger.NewNop()),
		db:     mysqlConn.DB,
		clean:  cleanup,
		nowUTC: time.Now().UTC().Truncate(time.Second),
	}
}

func pendingStatePlaceholder(state string) string {
	return pendingGmailAddressForState(state)
}

func skipIfDBUnavailable(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "dial tcp") || strings.Contains(err.Error(), "lookup mysql") {
		t.Skipf("Skipping repository integration test: %v", err)
	}
}

// --- Pending State tests ---

func TestSavePendingState_And_FindByState(t *testing.T) {
	env := newRepoTestEnv(t)
	defer env.clean()
	ctx := context.Background()

	ps := domain.OAuthPendingState{
		UserID:    1,
		State:     "test-state-123",
		ExpiresAt: env.nowUTC.Add(10 * time.Minute),
		CreatedAt: env.nowUTC,
	}

	err := env.repo.SavePendingState(ctx, ps)
	require.NoError(t, err)

	found, err := env.repo.FindPendingStateByState(ctx, "test-state-123")
	require.NoError(t, err)
	assert.Equal(t, uint(1), found.UserID)
	assert.Equal(t, "test-state-123", found.State)
	assert.Nil(t, found.ConsumedAt)

	var stored credentialRecord
	err = env.db.WithContext(ctx).Where("id = ?", found.ID).First(&stored).Error
	require.NoError(t, err)
	require.NotNil(t, stored.OAuthState)
	require.NotNil(t, stored.OAuthStateExpiresAt)
	assert.Equal(t, "test-state-123", *stored.OAuthState)
	assert.Equal(t, pendingStatePlaceholder("test-state-123"), stored.GmailAddress)
}

func TestFindPendingStateByState_NotFound(t *testing.T) {
	env := newRepoTestEnv(t)
	defer env.clean()
	ctx := context.Background()

	_, err := env.repo.FindPendingStateByState(ctx, "nonexistent")
	assert.ErrorIs(t, err, domain.ErrPendingStateNotFound)
}

func TestConsumePendingState(t *testing.T) {
	env := newRepoTestEnv(t)
	defer env.clean()
	ctx := context.Background()

	ps := domain.OAuthPendingState{
		UserID:    1,
		State:     "consume-me",
		ExpiresAt: env.nowUTC.Add(10 * time.Minute),
		CreatedAt: env.nowUTC,
	}
	require.NoError(t, env.repo.SavePendingState(ctx, ps))

	found, err := env.repo.FindPendingStateByState(ctx, "consume-me")
	require.NoError(t, err)

	err = env.repo.ConsumePendingState(ctx, found.ID, env.nowUTC)
	require.NoError(t, err)

	// Verify consumed rows are removed from the backing store.
	_, err = env.repo.FindPendingStateByState(ctx, "consume-me")
	assert.ErrorIs(t, err, domain.ErrPendingStateNotFound)

	err = env.repo.ConsumePendingState(ctx, found.ID, env.nowUTC)
	assert.Error(t, err)
}

// --- Credential tests ---

func TestCreateCredential_And_FindByUserAndGmail(t *testing.T) {
	env := newRepoTestEnv(t)
	defer env.clean()
	ctx := context.Background()

	cred := domain.EmailCredential{
		UserID:             1,
		Type:               "gmail",
		GmailAddress:       "user@gmail.com",
		KeyVersion:         1,
		AccessToken:        "enc-access",
		AccessTokenDigest:  "digest-access",
		RefreshToken:       "enc-refresh",
		RefreshTokenDigest: "digest-refresh",
		TokenExpiry:        &env.nowUTC,
		CreatedAt:          env.nowUTC,
		UpdatedAt:          env.nowUTC,
	}

	err := env.repo.CreateCredential(ctx, cred)
	require.NoError(t, err)

	found, err := env.repo.FindCredentialByUserAndGmail(ctx, 1, "user@gmail.com")
	require.NoError(t, err)
	assert.Equal(t, "user@gmail.com", found.GmailAddress)
	assert.Equal(t, "enc-access", found.AccessToken)
	assert.Equal(t, "enc-refresh", found.RefreshToken)
}

func TestFindCredentialByUserAndGmail_NotFound(t *testing.T) {
	env := newRepoTestEnv(t)
	defer env.clean()
	ctx := context.Background()

	_, err := env.repo.FindCredentialByUserAndGmail(ctx, 999, "nope@gmail.com")
	assert.ErrorIs(t, err, domain.ErrCredentialNotFound)
}

func TestFindCredentialByUserAndGmail_NormalizesAddress(t *testing.T) {
	env := newRepoTestEnv(t)
	defer env.clean()
	ctx := context.Background()

	cred := domain.EmailCredential{
		UserID:             1,
		Type:               "gmail",
		GmailAddress:       "user@gmail.com",
		KeyVersion:         1,
		AccessToken:        "a",
		AccessTokenDigest:  "a",
		RefreshToken:       "r",
		RefreshTokenDigest: "r",
		CreatedAt:          env.nowUTC,
		UpdatedAt:          env.nowUTC,
	}
	require.NoError(t, env.repo.CreateCredential(ctx, cred))

	// Query with uppercase should still find it
	found, err := env.repo.FindCredentialByUserAndGmail(ctx, 1, "USER@GMAIL.COM")
	require.NoError(t, err)
	assert.Equal(t, "user@gmail.com", found.GmailAddress)
}

func TestUpdateCredentialTokens(t *testing.T) {
	env := newRepoTestEnv(t)
	defer env.clean()
	ctx := context.Background()

	cred := domain.EmailCredential{
		UserID:             1,
		Type:               "gmail",
		GmailAddress:       "update@gmail.com",
		KeyVersion:         1,
		AccessToken:        "old-access",
		AccessTokenDigest:  "old-access-dig",
		RefreshToken:       "old-refresh",
		RefreshTokenDigest: "old-refresh-dig",
		CreatedAt:          env.nowUTC,
		UpdatedAt:          env.nowUTC,
	}
	require.NoError(t, env.repo.CreateCredential(ctx, cred))

	found, err := env.repo.FindCredentialByUserAndGmail(ctx, 1, "update@gmail.com")
	require.NoError(t, err)

	newExpiry := env.nowUTC.Add(1 * time.Hour)
	found.AccessToken = "new-access"
	found.AccessTokenDigest = "new-access-dig"
	found.RefreshToken = "new-refresh"
	found.RefreshTokenDigest = "new-refresh-dig"
	found.TokenExpiry = &newExpiry
	found.UpdatedAt = env.nowUTC.Add(1 * time.Minute)

	err = env.repo.UpdateCredentialTokens(ctx, found)
	require.NoError(t, err)

	updated, err := env.repo.FindCredentialByUserAndGmail(ctx, 1, "update@gmail.com")
	require.NoError(t, err)
	assert.Equal(t, "new-access", updated.AccessToken)
	assert.Equal(t, "new-refresh", updated.RefreshToken)
}

func TestListCredentialsByUser_FiltersAndOrders(t *testing.T) {
	env := newRepoTestEnv(t)
	defer env.clean()
	ctx := context.Background()

	older := env.nowUTC.Add(-2 * time.Hour)
	newer := env.nowUTC.Add(-1 * time.Hour)

	require.NoError(t, env.repo.CreateCredential(ctx, domain.EmailCredential{
		UserID:             1,
		Type:               "gmail",
		GmailAddress:       "first@gmail.com",
		KeyVersion:         1,
		AccessToken:        "enc-1",
		AccessTokenDigest:  "dig-1",
		RefreshToken:       "refresh-1",
		RefreshTokenDigest: "refresh-dig-1",
		CreatedAt:          older,
		UpdatedAt:          older,
	}))
	require.NoError(t, env.repo.CreateCredential(ctx, domain.EmailCredential{
		UserID:             2,
		Type:               "gmail",
		GmailAddress:       "other-user@gmail.com",
		KeyVersion:         1,
		AccessToken:        "enc-2",
		AccessTokenDigest:  "dig-2",
		RefreshToken:       "refresh-2",
		RefreshTokenDigest: "refresh-dig-2",
		CreatedAt:          env.nowUTC,
		UpdatedAt:          env.nowUTC,
	}))
	require.NoError(t, env.repo.CreateCredential(ctx, domain.EmailCredential{
		UserID:             1,
		Type:               "gmail",
		GmailAddress:       "second@gmail.com",
		KeyVersion:         1,
		AccessToken:        "enc-3",
		AccessTokenDigest:  "dig-3",
		RefreshToken:       "refresh-3",
		RefreshTokenDigest: "refresh-dig-3",
		CreatedAt:          newer,
		UpdatedAt:          newer,
	}))
	require.NoError(t, env.db.WithContext(ctx).Create(&credentialRecord{
		UserID:              1,
		Type:                "gmail",
		GmailAddress:        pendingStatePlaceholder("pending-list"),
		KeyVersion:          1,
		AccessToken:         "",
		AccessTokenDigest:   "",
		RefreshToken:        "",
		RefreshTokenDigest:  "",
		OAuthState:          stringPtr("pending-list"),
		OAuthStateExpiresAt: timePtr(env.nowUTC.Add(10 * time.Minute)),
		CreatedAt:           env.nowUTC,
		UpdatedAt:           env.nowUTC,
	}).Error)

	credentials, err := env.repo.ListCredentialsByUser(ctx, 1)
	require.NoError(t, err)
	require.Len(t, credentials, 2)
	assert.Equal(t, "first@gmail.com", credentials[0].GmailAddress)
	assert.Equal(t, "second@gmail.com", credentials[1].GmailAddress)
}

func TestDeleteCredentialByIDAndUser_Success(t *testing.T) {
	env := newRepoTestEnv(t)
	defer env.clean()
	ctx := context.Background()

	cred := domain.EmailCredential{
		UserID:             1,
		Type:               "gmail",
		GmailAddress:       "delete-me@gmail.com",
		KeyVersion:         1,
		AccessToken:        "enc-access",
		AccessTokenDigest:  "dig-access",
		RefreshToken:       "enc-refresh",
		RefreshTokenDigest: "dig-refresh",
		CreatedAt:          env.nowUTC,
		UpdatedAt:          env.nowUTC,
	}
	require.NoError(t, env.repo.CreateCredential(ctx, cred))

	found, err := env.repo.FindCredentialByUserAndGmail(ctx, 1, "delete-me@gmail.com")
	require.NoError(t, err)

	err = env.repo.DeleteCredentialByIDAndUser(ctx, found.ID, 1)
	require.NoError(t, err)

	_, err = env.repo.FindCredentialByUserAndGmail(ctx, 1, "delete-me@gmail.com")
	assert.ErrorIs(t, err, domain.ErrCredentialNotFound)
}

func TestDeleteCredentialByIDAndUser_OtherUserCannotDelete(t *testing.T) {
	env := newRepoTestEnv(t)
	defer env.clean()
	ctx := context.Background()

	cred := domain.EmailCredential{
		UserID:             1,
		Type:               "gmail",
		GmailAddress:       "owner@gmail.com",
		KeyVersion:         1,
		AccessToken:        "enc-access",
		AccessTokenDigest:  "dig-access",
		RefreshToken:       "enc-refresh",
		RefreshTokenDigest: "dig-refresh",
		CreatedAt:          env.nowUTC,
		UpdatedAt:          env.nowUTC,
	}
	require.NoError(t, env.repo.CreateCredential(ctx, cred))

	found, err := env.repo.FindCredentialByUserAndGmail(ctx, 1, "owner@gmail.com")
	require.NoError(t, err)

	err = env.repo.DeleteCredentialByIDAndUser(ctx, found.ID, 2)
	assert.ErrorIs(t, err, domain.ErrCredentialNotFound)

	stillExists, err := env.repo.FindCredentialByUserAndGmail(ctx, 1, "owner@gmail.com")
	require.NoError(t, err)
	assert.Equal(t, found.ID, stillExists.ID)
}

func TestDeleteCredentialByIDAndUser_NotFound(t *testing.T) {
	env := newRepoTestEnv(t)
	defer env.clean()
	ctx := context.Background()

	err := env.repo.DeleteCredentialByIDAndUser(ctx, 9999, 1)
	assert.ErrorIs(t, err, domain.ErrCredentialNotFound)
}

func TestDeleteCredentialByIDAndUser_PendingStateNotDeletableAsConnection(t *testing.T) {
	env := newRepoTestEnv(t)
	defer env.clean()
	ctx := context.Background()

	state := "pending-delete"
	require.NoError(t, env.repo.SavePendingState(ctx, domain.OAuthPendingState{
		UserID:    1,
		State:     state,
		ExpiresAt: env.nowUTC.Add(10 * time.Minute),
		CreatedAt: env.nowUTC,
	}))

	pending, err := env.repo.FindPendingStateByState(ctx, state)
	require.NoError(t, err)

	err = env.repo.DeleteCredentialByIDAndUser(ctx, pending.ID, 1)
	assert.ErrorIs(t, err, domain.ErrCredentialNotFound)
}
