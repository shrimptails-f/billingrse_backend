package application

import (
	"business/internal/library/crypto"
	"business/internal/library/logger"
	"business/internal/library/oswrapper"
	"business/internal/library/timewrapper"
	"business/internal/mailaccountconnection/domain"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

const (
	oauthStateTTL   = 10 * time.Minute
	oauthStateBytes = 32
	credentialType  = "gmail"
	vaultInfo       = "email_credential"
	defaultKeyVer   = int16(1)
)

// Repository defines persistence operations for mail account connections.
type Repository interface {
	SavePendingState(ctx context.Context, ps domain.OAuthPendingState) error
	FindPendingStateByState(ctx context.Context, state string) (domain.OAuthPendingState, error)
	ConsumePendingState(ctx context.Context, id uint, consumedAt time.Time) error
	FindCredentialByUserAndGmail(ctx context.Context, userID uint, gmailAddress string) (domain.EmailCredential, error)
	ListCredentialsByUser(ctx context.Context, userID uint) ([]domain.EmailCredential, error)
	DeleteCredentialByIDAndUser(ctx context.Context, credentialID, userID uint) error
	CreateCredential(ctx context.Context, cred domain.EmailCredential) error
	UpdateCredentialTokens(ctx context.Context, cred domain.EmailCredential) error
}

// OAuthConfigProvider resolves the Gmail OAuth2 config.
type OAuthConfigProvider interface {
	GetGmailOAuthConfig(ctx context.Context) (*oauth2.Config, error)
}

// OAuthTokenExchanger exchanges an auth code for tokens.
type OAuthTokenExchanger interface {
	Exchange(ctx context.Context, cfg *oauth2.Config, code string) (*oauth2.Token, error)
}

// GmailProfileFetcher fetches the Gmail address for the authenticated user.
type GmailProfileFetcher interface {
	GetEmailAddress(ctx context.Context, token *oauth2.Token, cfg *oauth2.Config) (string, error)
}

// UseCaseInterface defines the mail account connection use case operations.
type UseCaseInterface interface {
	Authorize(ctx context.Context, userID uint) (AuthorizeResult, error)
	Callback(ctx context.Context, userID uint, code, state string) error
	ListConnections(ctx context.Context, userID uint) ([]domain.ConnectionView, error)
	Disconnect(ctx context.Context, userID uint, connectionID uint) error
}

// AuthorizeResult holds the result of the authorize use case.
type AuthorizeResult struct {
	AuthorizationURL string
	ExpiresAt        time.Time
}

// UseCase implements mail account connection business logic.
type UseCase struct {
	repo      Repository
	oauthCfg  OAuthConfigProvider
	exchanger OAuthTokenExchanger
	profiler  GmailProfileFetcher
	osw       oswrapper.OsWapperInterface
	clock     timewrapper.ClockInterface
	log       logger.Interface
}

// NewUseCase creates a new UseCase.
func NewUseCase(
	repo Repository,
	oauthCfg OAuthConfigProvider,
	exchanger OAuthTokenExchanger,
	profiler GmailProfileFetcher,
	osw oswrapper.OsWapperInterface,
	clock timewrapper.ClockInterface,
	log logger.Interface,
) *UseCase {
	if clock == nil {
		clock = timewrapper.NewClock()
	}
	if log == nil {
		log = logger.NewNop()
	}
	return &UseCase{
		repo:      repo,
		oauthCfg:  oauthCfg,
		exchanger: exchanger,
		profiler:  profiler,
		osw:       osw,
		clock:     clock,
		log:       log.With(logger.Component("mail_account_connection_usecase")),
	}
}

// Authorize generates an OAuth authorization URL and saves a pending state.
func (uc *UseCase) Authorize(ctx context.Context, userID uint) (AuthorizeResult, error) {
	reqLog := uc.log
	if l, err := uc.log.WithContext(ctx); err == nil {
		reqLog = l
	}

	cfg, err := uc.oauthCfg.GetGmailOAuthConfig(ctx)
	if err != nil {
		reqLog.Error("oauth_config_load_failed", logger.Err(err))
		return AuthorizeResult{}, fmt.Errorf("failed to load oauth config: %w", err)
	}

	stateBytes := make([]byte, oauthStateBytes)
	if _, err := rand.Read(stateBytes); err != nil {
		reqLog.Error("state_generation_failed", logger.Err(err))
		return AuthorizeResult{}, fmt.Errorf("failed to generate state: %w", err)
	}
	state := hex.EncodeToString(stateBytes)

	now := uc.clock.Now()
	expiresAt := now.Add(oauthStateTTL)

	ps := domain.OAuthPendingState{
		UserID:    userID,
		State:     state,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}
	if err := uc.repo.SavePendingState(ctx, ps); err != nil {
		reqLog.Error("pending_state_save_failed", logger.Err(err))
		return AuthorizeResult{}, fmt.Errorf("failed to save pending state: %w", err)
	}

	url := cfg.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
	)

	reqLog.Info("oauth_authorize_initiated",
		logger.UserID(userID),
	)

	return AuthorizeResult{
		AuthorizationURL: url,
		ExpiresAt:        expiresAt,
	}, nil
}

// Callback validates state, exchanges code for tokens, and saves the credential.
func (uc *UseCase) Callback(ctx context.Context, userID uint, code, state string) error {
	reqLog := uc.log
	if l, err := uc.log.WithContext(ctx); err == nil {
		reqLog = l
	}

	// 1. Validate state
	ps, err := uc.repo.FindPendingStateByState(ctx, state)
	if err != nil {
		if errors.Is(err, domain.ErrPendingStateNotFound) {
			reqLog.Info("oauth_state_mismatch", logger.UserID(userID))
			return domain.ErrOAuthStateMismatch
		}
		reqLog.Error("pending_state_lookup_failed", logger.Err(err))
		return fmt.Errorf("failed to look up pending state: %w", err)
	}
	if ps.UserID != userID {
		reqLog.Info("oauth_state_mismatch", logger.UserID(userID))
		return domain.ErrOAuthStateMismatch
	}
	if ps.ConsumedAt != nil {
		reqLog.Info("oauth_state_mismatch", logger.UserID(userID))
		return domain.ErrOAuthStateMismatch
	}

	now := uc.clock.Now()
	if !now.Before(ps.ExpiresAt) {
		reqLog.Info("oauth_state_expired", logger.UserID(userID))
		return domain.ErrOAuthStateExpired
	}

	// 2. Consume state
	if err := uc.repo.ConsumePendingState(ctx, ps.ID, now); err != nil {
		reqLog.Error("pending_state_consume_failed", logger.Err(err))
		return fmt.Errorf("failed to consume pending state: %w", err)
	}

	// 3. Exchange code for token
	cfg, err := uc.oauthCfg.GetGmailOAuthConfig(ctx)
	if err != nil {
		reqLog.Error("oauth_config_load_failed", logger.Err(err))
		return domain.ErrOAuthExchangeFailed
	}

	token, err := uc.exchanger.Exchange(ctx, cfg, code)
	if err != nil {
		reqLog.Error("oauth_token_exchange_failed", logger.Err(err))
		return domain.ErrOAuthExchangeFailed
	}

	// 4. Fetch Gmail address
	gmailAddr, err := uc.profiler.GetEmailAddress(ctx, token, cfg)
	if err != nil {
		reqLog.Error("gmail_profile_fetch_failed", logger.Err(err))
		return domain.ErrGmailProfileFetchFailed
	}
	normalizedAddr := strings.ToLower(strings.TrimSpace(gmailAddr))

	// 5. Build vault for encryption
	vault, err := uc.buildVault()
	if err != nil {
		reqLog.Error("vault_build_failed", logger.Err(err))
		return domain.ErrVaultEncryptFailed
	}

	// 6. Check existing credential (distinguish not-found from DB error)
	existing, err := uc.repo.FindCredentialByUserAndGmail(ctx, userID, normalizedAddr)
	var isNew bool
	if err != nil {
		if errors.Is(err, domain.ErrCredentialNotFound) {
			isNew = true
		} else {
			reqLog.Error("credential_lookup_failed", logger.Err(err))
			return fmt.Errorf("failed to look up credential: %w", err)
		}
	}

	// 7. Encrypt access_token
	encAccess, err := vault.EncryptToString(token.AccessToken)
	if err != nil {
		reqLog.Error("access_token_encrypt_failed", logger.Err(err))
		return domain.ErrVaultEncryptFailed
	}
	digestAccess, err := vault.DigestToString(token.AccessToken)
	if err != nil {
		reqLog.Error("access_token_digest_failed", logger.Err(err))
		return domain.ErrVaultEncryptFailed
	}

	expiry := token.Expiry

	if isNew {
		// 8a. New connection: refresh_token is mandatory
		if token.RefreshToken == "" {
			reqLog.Error("refresh_token_missing_new_connection", logger.UserID(userID))
			return domain.ErrRefreshTokenMissing
		}

		encRefresh, err := vault.EncryptToString(token.RefreshToken)
		if err != nil {
			reqLog.Error("refresh_token_encrypt_failed", logger.Err(err))
			return domain.ErrVaultEncryptFailed
		}
		digestRefresh, err := vault.DigestToString(token.RefreshToken)
		if err != nil {
			reqLog.Error("refresh_token_digest_failed", logger.Err(err))
			return domain.ErrVaultEncryptFailed
		}

		cred := domain.EmailCredential{
			UserID:             userID,
			Type:               credentialType,
			GmailAddress:       normalizedAddr,
			KeyVersion:         defaultKeyVer,
			AccessToken:        encAccess,
			AccessTokenDigest:  digestAccess,
			RefreshToken:       encRefresh,
			RefreshTokenDigest: digestRefresh,
			TokenExpiry:        &expiry,
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		if err := uc.repo.CreateCredential(ctx, cred); err != nil {
			reqLog.Error("credential_create_failed", logger.Err(err))
			return fmt.Errorf("failed to create credential: %w", err)
		}
		reqLog.Info("gmail_connection_created",
			logger.UserID(userID),
			logger.String("provider", credentialType),
		)
	} else {
		// 8b. Re-link: update existing credential
		existing.AccessToken = encAccess
		existing.AccessTokenDigest = digestAccess

		if token.RefreshToken != "" {
			encRefresh, err := vault.EncryptToString(token.RefreshToken)
			if err != nil {
				reqLog.Error("refresh_token_encrypt_failed", logger.Err(err))
				return domain.ErrVaultEncryptFailed
			}
			digestRefresh, err := vault.DigestToString(token.RefreshToken)
			if err != nil {
				reqLog.Error("refresh_token_digest_failed", logger.Err(err))
				return domain.ErrVaultEncryptFailed
			}
			existing.RefreshToken = encRefresh
			existing.RefreshTokenDigest = digestRefresh
		}
		// else: keep existing refresh_token as-is

		existing.TokenExpiry = &expiry
		existing.UpdatedAt = now
		if err := uc.repo.UpdateCredentialTokens(ctx, existing); err != nil {
			reqLog.Error("credential_update_failed", logger.Err(err))
			return fmt.Errorf("failed to update credential: %w", err)
		}
		reqLog.Info("gmail_connection_updated",
			logger.UserID(userID),
			logger.String("provider", credentialType),
		)
	}

	return nil
}

// ListConnections lists the caller's stored mail account connections without probing the provider.
func (uc *UseCase) ListConnections(ctx context.Context, userID uint) ([]domain.ConnectionView, error) {
	reqLog := uc.log
	if l, err := uc.log.WithContext(ctx); err == nil {
		reqLog = l
	}

	credentials, err := uc.repo.ListCredentialsByUser(ctx, userID)
	if err != nil {
		reqLog.Error("credential_list_failed", logger.Err(err))
		return nil, fmt.Errorf("failed to list credentials: %w", err)
	}

	if len(credentials) == 0 {
		return []domain.ConnectionView{}, nil
	}

	connections := make([]domain.ConnectionView, 0, len(credentials))
	for _, credential := range credentials {
		connections = append(connections, domain.ConnectionView{
			ID:                credential.ID,
			Provider:          strings.ToLower(strings.TrimSpace(credential.Type)),
			AccountIdentifier: strings.ToLower(strings.TrimSpace(credential.GmailAddress)),
			CreatedAt:         credential.CreatedAt,
			UpdatedAt:         credential.UpdatedAt,
		})
	}

	return connections, nil
}

// Disconnect removes the caller's stored mail account connection.
func (uc *UseCase) Disconnect(ctx context.Context, userID uint, connectionID uint) error {
	reqLog := uc.log
	if l, err := uc.log.WithContext(ctx); err == nil {
		reqLog = l
	}

	err := uc.repo.DeleteCredentialByIDAndUser(ctx, connectionID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrCredentialNotFound) {
			reqLog.Info("mail_account_connection_not_found",
				logger.UserID(userID),
				logger.Uint("connection_id", connectionID),
			)
			return domain.ErrCredentialNotFound
		}

		reqLog.Error("mail_account_connection_disconnect_failed",
			logger.UserID(userID),
			logger.Uint("connection_id", connectionID),
			logger.Err(err),
		)
		return fmt.Errorf("failed to disconnect credential: %w", err)
	}

	reqLog.Info("mail_account_connection_disconnected",
		logger.UserID(userID),
		logger.Uint("connection_id", connectionID),
	)

	return nil
}

func (uc *UseCase) buildVault() (*crypto.Vault, error) {
	keyMaterial, err := uc.osw.GetEnv("EMAIL_TOKEN_KEY_V1")
	if err != nil {
		return nil, fmt.Errorf("failed to read EMAIL_TOKEN_KEY_V1: %w", err)
	}
	salt, err := uc.osw.GetEnv("EMAIL_TOKEN_SALT")
	if err != nil {
		return nil, fmt.Errorf("failed to read EMAIL_TOKEN_SALT: %w", err)
	}
	return crypto.NewVault(crypto.VaultConfig{
		KeyMaterial: []byte(keyMaterial),
		Salt:        []byte(salt),
		Info:        vaultInfo,
	})
}
