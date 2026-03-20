package application

import (
	"business/internal/emailcredential/domain"
	"business/internal/library/logger"
	mocklibrary "business/test/mock/library"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/oauth2"
)

// --- mocks ---

type mockRepo struct {
	mock.Mock
}

func (m *mockRepo) SavePendingState(ctx context.Context, ps domain.OAuthPendingState) error {
	args := m.Called(ctx, ps)
	return args.Error(0)
}

func (m *mockRepo) FindPendingStateByState(ctx context.Context, state string) (domain.OAuthPendingState, error) {
	args := m.Called(ctx, state)
	return args.Get(0).(domain.OAuthPendingState), args.Error(1)
}

func (m *mockRepo) ConsumePendingState(ctx context.Context, id uint, consumedAt time.Time) error {
	args := m.Called(ctx, id, consumedAt)
	return args.Error(0)
}

func (m *mockRepo) FindCredentialByUserAndGmail(ctx context.Context, userID uint, gmailAddress string) (domain.EmailCredential, error) {
	args := m.Called(ctx, userID, gmailAddress)
	return args.Get(0).(domain.EmailCredential), args.Error(1)
}

func (m *mockRepo) ListCredentialsByUser(ctx context.Context, userID uint) ([]domain.EmailCredential, error) {
	args := m.Called(ctx, userID)
	credentials, _ := args.Get(0).([]domain.EmailCredential)
	return credentials, args.Error(1)
}

func (m *mockRepo) CreateCredential(ctx context.Context, cred domain.EmailCredential) error {
	args := m.Called(ctx, cred)
	return args.Error(0)
}

func (m *mockRepo) UpdateCredentialTokens(ctx context.Context, cred domain.EmailCredential) error {
	args := m.Called(ctx, cred)
	return args.Error(0)
}

type mockOAuthCfg struct {
	mock.Mock
}

func (m *mockOAuthCfg) GetGmailOAuthConfig(ctx context.Context) (*oauth2.Config, error) {
	args := m.Called(ctx)
	cfg, _ := args.Get(0).(*oauth2.Config)
	return cfg, args.Error(1)
}

type mockExchanger struct {
	mock.Mock
}

func (m *mockExchanger) Exchange(ctx context.Context, cfg *oauth2.Config, code string) (*oauth2.Token, error) {
	args := m.Called(ctx, cfg, code)
	tok, _ := args.Get(0).(*oauth2.Token)
	return tok, args.Error(1)
}

type mockProfiler struct {
	mock.Mock
}

func (m *mockProfiler) GetEmailAddress(ctx context.Context, token *oauth2.Token, cfg *oauth2.Config) (string, error) {
	args := m.Called(ctx, token, cfg)
	return args.String(0), args.Error(1)
}

type fixedClock struct {
	now time.Time
}

func (c *fixedClock) Now() time.Time                         { return c.now }
func (c *fixedClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

func testOSW() *mocklibrary.OsWrapperMock {
	return mocklibrary.NewOsWrapperMock(map[string]string{
		"EMAIL_TOKEN_KEY_V1": "01234567890123456789012345678901", // 32 bytes
		"EMAIL_TOKEN_SALT":   "test-salt-value",
	})
}

func testClock(t time.Time) *fixedClock {
	return &fixedClock{now: t}
}

func testOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "http://localhost/callback",
		Scopes:       []string{"https://www.googleapis.com/auth/gmail.readonly"},
	}
}

func testToken() *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  "access-tok",
		RefreshToken: "refresh-tok",
		Expiry:       time.Date(2026, 3, 19, 13, 0, 0, 0, time.UTC),
	}
}

func testListCredential(id uint, provider string, accountIdentifier string, createdAt time.Time) domain.EmailCredential {
	return domain.EmailCredential{
		ID:                 id,
		UserID:             1,
		Type:               provider,
		GmailAddress:       accountIdentifier,
		KeyVersion:         1,
		AccessToken:        "unused-access-token",
		AccessTokenDigest:  "unused-access-token-digest",
		RefreshToken:       "unused-refresh-token",
		RefreshTokenDigest: "unused-refresh-token-digest",
		TokenExpiry:        nil,
		CreatedAt:          createdAt,
		UpdatedAt:          createdAt.Add(5 * time.Minute),
	}
}

func newTestUseCase(
	repo *mockRepo,
	oauthCfg *mockOAuthCfg,
	exchanger *mockExchanger,
	profiler *mockProfiler,
	osw *mocklibrary.OsWrapperMock,
	clock *fixedClock,
) UseCaseInterface {
	var log logger.Interface = mocklibrary.NewNopLogger()
	return NewUseCase(repo, oauthCfg, exchanger, profiler, osw, clock, log)
}

// --- Authorize tests ---

func TestAuthorize_Success(t *testing.T) {
	t.Parallel()
	repo := new(mockRepo)
	oauthCfg := new(mockOAuthCfg)
	now := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)

	oauthCfg.On("GetGmailOAuthConfig", mock.Anything).Return(testOAuthConfig(), nil)
	repo.On("SavePendingState", mock.Anything, mock.MatchedBy(func(ps domain.OAuthPendingState) bool {
		return ps.UserID == 1 && ps.State != "" && ps.ExpiresAt.After(now)
	})).Return(nil)

	uc := newTestUseCase(repo, oauthCfg, nil, nil, testOSW(), testClock(now))
	result, err := uc.Authorize(context.Background(), 1)

	assert.NoError(t, err)
	assert.NotEmpty(t, result.AuthorizationURL)
	assert.Contains(t, result.AuthorizationURL, "access_type=offline")
	assert.Contains(t, result.AuthorizationURL, "prompt=consent")
	assert.Contains(t, result.AuthorizationURL, "state=")
	assert.Contains(t, result.AuthorizationURL, "client_id=client-id")
	assert.Equal(t, now.Add(10*time.Minute), result.ExpiresAt)
	repo.AssertExpectations(t)
}

// --- Callback tests ---

func validPendingState(now time.Time) domain.OAuthPendingState {
	return domain.OAuthPendingState{
		ID:        1,
		UserID:    1,
		State:     "valid-state",
		ExpiresAt: now.Add(5 * time.Minute),
		CreatedAt: now.Add(-5 * time.Minute),
	}
}

func TestCallback_StateMismatch(t *testing.T) {
	t.Parallel()
	repo := new(mockRepo)
	oauthCfg := new(mockOAuthCfg)
	now := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)

	repo.On("FindPendingStateByState", mock.Anything, "bad-state").
		Return(domain.OAuthPendingState{}, domain.ErrPendingStateNotFound)

	uc := newTestUseCase(repo, oauthCfg, nil, nil, testOSW(), testClock(now))
	err := uc.Callback(context.Background(), 1, "code", "bad-state")

	assert.ErrorIs(t, err, domain.ErrOAuthStateMismatch)
	repo.AssertExpectations(t)
}

func TestCallback_StateMismatch_DBError(t *testing.T) {
	t.Parallel()
	repo := new(mockRepo)
	now := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)

	repo.On("FindPendingStateByState", mock.Anything, "state").
		Return(domain.OAuthPendingState{}, errors.New("db connection error"))

	uc := newTestUseCase(repo, new(mockOAuthCfg), nil, nil, testOSW(), testClock(now))
	err := uc.Callback(context.Background(), 1, "code", "state")

	assert.Error(t, err)
	assert.NotErrorIs(t, err, domain.ErrOAuthStateMismatch)
	assert.Contains(t, err.Error(), "failed to look up pending state")
	repo.AssertExpectations(t)
}

func TestCallback_StateExpired(t *testing.T) {
	t.Parallel()
	repo := new(mockRepo)
	now := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)

	expired := domain.OAuthPendingState{
		ID:        1,
		UserID:    1,
		State:     "expired-state",
		ExpiresAt: now.Add(-1 * time.Minute), // already expired
		CreatedAt: now.Add(-15 * time.Minute),
	}
	repo.On("FindPendingStateByState", mock.Anything, "expired-state").Return(expired, nil)

	uc := newTestUseCase(repo, new(mockOAuthCfg), nil, nil, testOSW(), testClock(now))
	err := uc.Callback(context.Background(), 1, "code", "expired-state")

	assert.ErrorIs(t, err, domain.ErrOAuthStateExpired)
	repo.AssertExpectations(t)
}

func TestCallback_NewConnection_Success(t *testing.T) {
	t.Parallel()
	repo := new(mockRepo)
	oauthCfg := new(mockOAuthCfg)
	exchanger := new(mockExchanger)
	profiler := new(mockProfiler)
	now := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)
	cfg := testOAuthConfig()
	tok := testToken()
	ps := validPendingState(now)

	repo.On("FindPendingStateByState", mock.Anything, "valid-state").Return(ps, nil)
	repo.On("ConsumePendingState", mock.Anything, uint(1), now).Return(nil)
	oauthCfg.On("GetGmailOAuthConfig", mock.Anything).Return(cfg, nil)
	exchanger.On("Exchange", mock.Anything, cfg, "auth-code").Return(tok, nil)
	profiler.On("GetEmailAddress", mock.Anything, tok, cfg).Return("User@Gmail.com", nil)
	repo.On("FindCredentialByUserAndGmail", mock.Anything, uint(1), "user@gmail.com").
		Return(domain.EmailCredential{}, domain.ErrCredentialNotFound)
	repo.On("CreateCredential", mock.Anything, mock.MatchedBy(func(c domain.EmailCredential) bool {
		return c.UserID == 1 &&
			c.Type == "gmail" &&
			c.GmailAddress == "user@gmail.com" &&
			c.AccessToken != "" &&
			c.RefreshToken != ""
	})).Return(nil)

	uc := newTestUseCase(repo, oauthCfg, exchanger, profiler, testOSW(), testClock(now))
	err := uc.Callback(context.Background(), 1, "auth-code", "valid-state")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
	exchanger.AssertExpectations(t)
	profiler.AssertExpectations(t)
}

func TestCallback_DifferentGmail_CreatesNew(t *testing.T) {
	t.Parallel()
	repo := new(mockRepo)
	oauthCfg := new(mockOAuthCfg)
	exchanger := new(mockExchanger)
	profiler := new(mockProfiler)
	now := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)
	cfg := testOAuthConfig()
	tok := testToken()
	ps := validPendingState(now)

	repo.On("FindPendingStateByState", mock.Anything, "valid-state").Return(ps, nil)
	repo.On("ConsumePendingState", mock.Anything, uint(1), now).Return(nil)
	oauthCfg.On("GetGmailOAuthConfig", mock.Anything).Return(cfg, nil)
	exchanger.On("Exchange", mock.Anything, cfg, "code").Return(tok, nil)
	profiler.On("GetEmailAddress", mock.Anything, tok, cfg).Return("other@gmail.com", nil)
	repo.On("FindCredentialByUserAndGmail", mock.Anything, uint(1), "other@gmail.com").
		Return(domain.EmailCredential{}, domain.ErrCredentialNotFound)
	repo.On("CreateCredential", mock.Anything, mock.MatchedBy(func(c domain.EmailCredential) bool {
		return c.GmailAddress == "other@gmail.com"
	})).Return(nil)

	uc := newTestUseCase(repo, oauthCfg, exchanger, profiler, testOSW(), testClock(now))
	err := uc.Callback(context.Background(), 1, "code", "valid-state")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestCallback_SameGmail_Updates(t *testing.T) {
	t.Parallel()
	repo := new(mockRepo)
	oauthCfg := new(mockOAuthCfg)
	exchanger := new(mockExchanger)
	profiler := new(mockProfiler)
	now := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)
	cfg := testOAuthConfig()
	tok := testToken()
	ps := validPendingState(now)

	existing := domain.EmailCredential{
		ID:                 42,
		UserID:             1,
		Type:               "gmail",
		GmailAddress:       "user@gmail.com",
		AccessToken:        "old-enc-access",
		AccessTokenDigest:  "old-access-dig",
		RefreshToken:       "old-enc-refresh",
		RefreshTokenDigest: "old-refresh-dig",
	}

	repo.On("FindPendingStateByState", mock.Anything, "valid-state").Return(ps, nil)
	repo.On("ConsumePendingState", mock.Anything, uint(1), now).Return(nil)
	oauthCfg.On("GetGmailOAuthConfig", mock.Anything).Return(cfg, nil)
	exchanger.On("Exchange", mock.Anything, cfg, "code").Return(tok, nil)
	profiler.On("GetEmailAddress", mock.Anything, tok, cfg).Return("user@gmail.com", nil)
	repo.On("FindCredentialByUserAndGmail", mock.Anything, uint(1), "user@gmail.com").
		Return(existing, nil)
	repo.On("UpdateCredentialTokens", mock.Anything, mock.MatchedBy(func(c domain.EmailCredential) bool {
		return c.ID == 42 &&
			c.AccessToken != "old-enc-access" &&
			c.RefreshToken != "old-enc-refresh"
	})).Return(nil)

	uc := newTestUseCase(repo, oauthCfg, exchanger, profiler, testOSW(), testClock(now))
	err := uc.Callback(context.Background(), 1, "code", "valid-state")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestCallback_SameGmail_NoRefreshToken_KeepsExisting(t *testing.T) {
	t.Parallel()
	repo := new(mockRepo)
	oauthCfg := new(mockOAuthCfg)
	exchanger := new(mockExchanger)
	profiler := new(mockProfiler)
	now := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)
	cfg := testOAuthConfig()
	// Token without refresh_token
	tok := &oauth2.Token{
		AccessToken: "new-access",
		Expiry:      time.Date(2026, 3, 19, 13, 0, 0, 0, time.UTC),
	}
	ps := validPendingState(now)

	existing := domain.EmailCredential{
		ID:                 42,
		UserID:             1,
		Type:               "gmail",
		GmailAddress:       "user@gmail.com",
		RefreshToken:       "existing-enc-refresh",
		RefreshTokenDigest: "existing-refresh-dig",
	}

	repo.On("FindPendingStateByState", mock.Anything, "valid-state").Return(ps, nil)
	repo.On("ConsumePendingState", mock.Anything, uint(1), now).Return(nil)
	oauthCfg.On("GetGmailOAuthConfig", mock.Anything).Return(cfg, nil)
	exchanger.On("Exchange", mock.Anything, cfg, "code").Return(tok, nil)
	profiler.On("GetEmailAddress", mock.Anything, tok, cfg).Return("user@gmail.com", nil)
	repo.On("FindCredentialByUserAndGmail", mock.Anything, uint(1), "user@gmail.com").
		Return(existing, nil)
	repo.On("UpdateCredentialTokens", mock.Anything, mock.MatchedBy(func(c domain.EmailCredential) bool {
		// refresh_token should remain the same
		return c.ID == 42 &&
			c.RefreshToken == "existing-enc-refresh" &&
			c.RefreshTokenDigest == "existing-refresh-dig"
	})).Return(nil)

	uc := newTestUseCase(repo, oauthCfg, exchanger, profiler, testOSW(), testClock(now))
	err := uc.Callback(context.Background(), 1, "code", "valid-state")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestCallback_NewConnection_NoRefreshToken_Fails(t *testing.T) {
	t.Parallel()
	repo := new(mockRepo)
	oauthCfg := new(mockOAuthCfg)
	exchanger := new(mockExchanger)
	profiler := new(mockProfiler)
	now := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)
	cfg := testOAuthConfig()
	tok := &oauth2.Token{
		AccessToken: "access-tok",
		Expiry:      time.Date(2026, 3, 19, 13, 0, 0, 0, time.UTC),
	} // no refresh_token
	ps := validPendingState(now)

	repo.On("FindPendingStateByState", mock.Anything, "valid-state").Return(ps, nil)
	repo.On("ConsumePendingState", mock.Anything, uint(1), now).Return(nil)
	oauthCfg.On("GetGmailOAuthConfig", mock.Anything).Return(cfg, nil)
	exchanger.On("Exchange", mock.Anything, cfg, "code").Return(tok, nil)
	profiler.On("GetEmailAddress", mock.Anything, tok, cfg).Return("new@gmail.com", nil)
	repo.On("FindCredentialByUserAndGmail", mock.Anything, uint(1), "new@gmail.com").
		Return(domain.EmailCredential{}, domain.ErrCredentialNotFound)

	uc := newTestUseCase(repo, oauthCfg, exchanger, profiler, testOSW(), testClock(now))
	err := uc.Callback(context.Background(), 1, "code", "valid-state")

	assert.ErrorIs(t, err, domain.ErrRefreshTokenMissing)
	repo.AssertExpectations(t)
}

func TestCallback_ExchangeFailed(t *testing.T) {
	t.Parallel()
	repo := new(mockRepo)
	oauthCfg := new(mockOAuthCfg)
	exchanger := new(mockExchanger)
	now := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)
	cfg := testOAuthConfig()
	ps := validPendingState(now)

	repo.On("FindPendingStateByState", mock.Anything, "valid-state").Return(ps, nil)
	repo.On("ConsumePendingState", mock.Anything, uint(1), now).Return(nil)
	oauthCfg.On("GetGmailOAuthConfig", mock.Anything).Return(cfg, nil)
	exchanger.On("Exchange", mock.Anything, cfg, "code").Return((*oauth2.Token)(nil), errors.New("google error"))

	uc := newTestUseCase(repo, oauthCfg, exchanger, nil, testOSW(), testClock(now))
	err := uc.Callback(context.Background(), 1, "code", "valid-state")

	assert.ErrorIs(t, err, domain.ErrOAuthExchangeFailed)
	repo.AssertExpectations(t)
}

func TestCallback_CredentialLookup_DBError(t *testing.T) {
	t.Parallel()
	repo := new(mockRepo)
	oauthCfg := new(mockOAuthCfg)
	exchanger := new(mockExchanger)
	profiler := new(mockProfiler)
	now := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)
	cfg := testOAuthConfig()
	tok := testToken()
	ps := validPendingState(now)

	repo.On("FindPendingStateByState", mock.Anything, "valid-state").Return(ps, nil)
	repo.On("ConsumePendingState", mock.Anything, uint(1), now).Return(nil)
	oauthCfg.On("GetGmailOAuthConfig", mock.Anything).Return(cfg, nil)
	exchanger.On("Exchange", mock.Anything, cfg, "code").Return(tok, nil)
	profiler.On("GetEmailAddress", mock.Anything, tok, cfg).Return("user@gmail.com", nil)
	repo.On("FindCredentialByUserAndGmail", mock.Anything, uint(1), "user@gmail.com").
		Return(domain.EmailCredential{}, errors.New("db connection timeout"))

	uc := newTestUseCase(repo, oauthCfg, exchanger, profiler, testOSW(), testClock(now))
	err := uc.Callback(context.Background(), 1, "code", "valid-state")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to look up credential")
	assert.NotErrorIs(t, err, domain.ErrCredentialNotFound)
	repo.AssertExpectations(t)
}

func TestListConnections_Empty(t *testing.T) {
	t.Parallel()
	repo := new(mockRepo)
	now := time.Date(2026, 3, 20, 2, 0, 0, 0, time.UTC)

	repo.On("ListCredentialsByUser", mock.Anything, uint(1)).Return([]domain.EmailCredential{}, nil)

	uc := newTestUseCase(repo, new(mockOAuthCfg), nil, nil, testOSW(), testClock(now))
	connections, err := uc.ListConnections(context.Background(), 1)

	assert.NoError(t, err)
	assert.Empty(t, connections)
	repo.AssertExpectations(t)
}

func TestListConnections_Success(t *testing.T) {
	t.Parallel()
	repo := new(mockRepo)
	now := time.Date(2026, 3, 20, 2, 10, 0, 0, time.UTC)
	credential := testListCredential(12, "gmail", "User@Gmail.com", now.Add(-24*time.Hour))
	unsupported := testListCredential(13, "outlook", "user@outlook.com", now.Add(-23*time.Hour))

	repo.On("ListCredentialsByUser", mock.Anything, uint(1)).Return([]domain.EmailCredential{credential, unsupported}, nil)

	uc := newTestUseCase(repo, new(mockOAuthCfg), nil, nil, testOSW(), testClock(now))
	connections, err := uc.ListConnections(context.Background(), 1)

	assert.NoError(t, err)
	assert.Len(t, connections, 2)
	assert.Equal(t, uint(12), connections[0].ID)
	assert.Equal(t, "gmail", connections[0].Provider)
	assert.Equal(t, "user@gmail.com", connections[0].AccountIdentifier)
	assert.Equal(t, credential.CreatedAt, connections[0].CreatedAt)
	assert.Equal(t, credential.UpdatedAt, connections[0].UpdatedAt)
	assert.Equal(t, "outlook", connections[1].Provider)
	assert.Equal(t, "user@outlook.com", connections[1].AccountIdentifier)
	repo.AssertExpectations(t)
}

func TestListConnections_ListError(t *testing.T) {
	t.Parallel()
	repo := new(mockRepo)
	now := time.Date(2026, 3, 20, 2, 15, 0, 0, time.UTC)
	repo.On("ListCredentialsByUser", mock.Anything, uint(1)).Return(([]domain.EmailCredential)(nil), errors.New("db timeout"))

	uc := newTestUseCase(repo, new(mockOAuthCfg), nil, nil, testOSW(), testClock(now))
	connections, err := uc.ListConnections(context.Background(), 1)

	assert.Nil(t, connections)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list credentials")
	repo.AssertExpectations(t)
}
