package test

import (
	"business/internal/app/middleware"
	authpresentation "business/internal/app/presentation/auth"
	macpresentation "business/internal/app/presentation/mailaccountconnection"
	manualpresentation "business/internal/app/presentation/manualmailworkflow"
	v1 "business/internal/app/router"
	authapp "business/internal/auth/application"
	authdomain "business/internal/auth/domain"
	authinfra "business/internal/auth/infrastructure"
	"business/internal/library/crypto"
	"business/internal/library/gmailService"
	"business/internal/library/logger"
	"business/internal/library/mysql"
	macapp "business/internal/mailaccountconnection/application"
	macinfra "business/internal/mailaccountconnection/infrastructure"
	manualapp "business/internal/manualmailworkflow/application"
	mocklibrary "business/test/mock/library"
	model "business/tools/migrations/models"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/dig"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

const (
	scenarioJWTSecret          = "scenario-jwt-secret"
	scenarioAllowedOrigin      = "http://localhost:3000"
	scenarioPendingStatePrefix = "__pending_state__:"
)

type mailAccountConnectionScenarioEnv struct {
	t              *testing.T
	db             *gorm.DB
	router         *gin.Engine
	cleanup        func() error
	osw            *mocklibrary.OsWrapperMock
	exchanger      *scenarioOAuthTokenExchanger
	profileFetcher *scenarioGmailProfileFetcher
	userID         uint
	otherUserID    uint
}

type scenarioAuthorizeResponse struct {
	AuthorizationURL string    `json:"authorization_url"`
	ExpiresAt        time.Time `json:"expires_at"`
}

type scenarioCallbackResponse struct {
	Message string `json:"message"`
}

type scenarioListResponse struct {
	Items []scenarioConnectionItem `json:"items"`
}

type scenarioConnectionItem struct {
	ID                uint      `json:"id"`
	Provider          string    `json:"provider"`
	AccountIdentifier string    `json:"account_identifier"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type scenarioErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type scenarioExchangeResponse struct {
	token *oauth2.Token
	err   error
}

type scenarioProfileResponse struct {
	email string
	err   error
}

type scenarioOAuthTokenExchanger struct {
	t         *testing.T
	responses []scenarioExchangeResponse
}

type scenarioGmailProfileFetcher struct {
	t         *testing.T
	responses []scenarioProfileResponse
}

type scenarioStubAuthUseCase struct{}

type scenarioStubManualMailWorkflowUseCase struct{}

func newMailAccountConnectionScenarioEnv(t *testing.T) *mailAccountConnectionScenarioEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	mysqlConn, cleanup, err := mysql.CreateNewTestDB()
	if err != nil {
		skipScenarioDBUnavailable(t, err)
	}
	require.NoError(t, err)

	t.Cleanup(func() {
		if cleanup != nil {
			require.NoError(t, cleanup())
		}
	})

	require.NoError(t, mysqlConn.DB.AutoMigrate(&model.User{}, &model.EmailCredential{}))

	osw := mocklibrary.NewOsWrapperMock(map[string]string{
		"APP":                       "test",
		"DOMAIN":                    "localhost",
		"JWT_SECRET_KEY":            scenarioJWTSecret,
		"EMAIL_GMAIL_CLIENT_ID":     "test-client-id",
		"EMAIL_GMAIL_CLIENT_SECRET": "test-client-secret",
		"EMAIL_GMAIL_REDIRECT_URL":  "http://localhost:3000/mail/callback",
		"EMAIL_TOKEN_KEY_V1":        "01234567890123456789012345678901",
		"EMAIL_TOKEN_SALT":          "test-salt-value",
	})

	log := logger.NewNop()
	authRepo := authinfra.NewRepository(mysqlConn.DB, log)
	authMiddleware := middleware.NewAuthMiddleware(osw, authRepo, log)
	authController := authpresentation.NewController(&scenarioStubAuthUseCase{}, log, osw)

	oauthCfg := gmailService.NewOAuthConfigLoader(osw)
	exchanger := &scenarioOAuthTokenExchanger{t: t}
	profileFetcher := &scenarioGmailProfileFetcher{t: t}
	vault, err := crypto.NewVault(crypto.VaultConfig{
		KeyMaterial: []byte("01234567890123456789012345678901"),
		Salt:        []byte("test-salt-value"),
		Info:        "email-credential-encryption",
		BcryptCost:  bcrypt.MinCost,
	})
	require.NoError(t, err)

	macRepo := macinfra.NewRepository(mysqlConn.DB, log)
	macUseCase := macapp.NewUseCase(macRepo, oauthCfg, exchanger, profileFetcher, vault, nil, log)
	macController := macpresentation.NewController(macUseCase, log)
	manualController := manualpresentation.NewController(&scenarioStubManualMailWorkflowUseCase{}, log)

	router := gin.New()
	container := dig.New()
	require.NoError(t, container.Provide(func() *authpresentation.Controller { return authController }))
	require.NoError(t, container.Provide(func() *middleware.AuthMiddleware { return authMiddleware }))
	require.NoError(t, container.Provide(func() *macpresentation.Controller { return macController }))
	require.NoError(t, container.Provide(func() *manualpresentation.Controller { return manualController }))
	_, err = v1.Router(router, container, log, scenarioAllowedOrigin)
	require.NoError(t, err)

	env := &mailAccountConnectionScenarioEnv{
		t:              t,
		db:             mysqlConn.DB,
		router:         router,
		cleanup:        cleanup,
		osw:            osw,
		exchanger:      exchanger,
		profileFetcher: profileFetcher,
	}
	env.userID = env.mustCreateVerifiedUser("scenario-user-1", "scenario-user-1@example.com")
	env.otherUserID = env.mustCreateVerifiedUser("scenario-user-2", "scenario-user-2@example.com")
	return env
}

func skipScenarioDBUnavailable(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	msg := err.Error()
	skipPatterns := []string{
		"dial tcp",
		"connect: connection refused",
		"lookup mysql",
		"access denied",
		"environment variable MYSQL_",
		"environment variable DB_HOST",
	}
	for _, pattern := range skipPatterns {
		if strings.Contains(msg, pattern) {
			t.Skipf("Skipping MailAccountConnection scenario test: %v", err)
		}
	}
}

func (e *mailAccountConnectionScenarioEnv) mustCreateVerifiedUser(name, email string) uint {
	e.t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	user := model.User{
		Name:            name,
		Email:           email,
		Password:        "$2a$10$abcdefghijklmnopqrstuv",
		EmailVerified:   true,
		EmailVerifiedAt: &now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	require.NoError(e.t, e.db.Create(&user).Error)
	return user.ID
}

func (e *mailAccountConnectionScenarioEnv) queueOAuthSuccess(email, accessToken, refreshToken string, expiry time.Time) {
	e.t.Helper()
	e.exchanger.Push(&oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Expiry:       expiry,
	}, nil)
	e.profileFetcher.Push(email, nil)
}

func (e *mailAccountConnectionScenarioEnv) queueOAuthExchangeFailure(err error) {
	e.t.Helper()
	e.exchanger.Push(nil, err)
}

func (e *mailAccountConnectionScenarioEnv) queueProfileFailure(token *oauth2.Token, err error) {
	e.t.Helper()
	e.exchanger.Push(token, nil)
	e.profileFetcher.Push("", err)
}

func (e *mailAccountConnectionScenarioEnv) authorize(userID uint) *httptest.ResponseRecorder {
	e.t.Helper()
	return e.doJSON(http.MethodPost, "/api/v1/mail-account-connections/gmail/authorize", userID, nil)
}

func (e *mailAccountConnectionScenarioEnv) callback(userID uint, code, state string) *httptest.ResponseRecorder {
	e.t.Helper()
	return e.doJSON(http.MethodPost, "/api/v1/mail-account-connections/gmail/callback", userID, map[string]string{
		"code":  code,
		"state": state,
	})
}

func (e *mailAccountConnectionScenarioEnv) listConnections(userID uint) *httptest.ResponseRecorder {
	e.t.Helper()
	return e.doJSON(http.MethodGet, "/api/v1/mail-account-connections", userID, nil)
}

func (e *mailAccountConnectionScenarioEnv) disconnect(userID, connectionID uint) *httptest.ResponseRecorder {
	e.t.Helper()
	return e.doJSON(http.MethodDelete, fmt.Sprintf("/api/v1/mail-account-connections/%d", connectionID), userID, nil)
}

func (e *mailAccountConnectionScenarioEnv) doJSON(method, path string, userID uint, body any) *httptest.ResponseRecorder {
	e.t.Helper()

	var reqBody *bytes.Reader
	if body == nil {
		reqBody = bytes.NewReader(nil)
	} else {
		payload, err := json.Marshal(body)
		require.NoError(e.t, err)
		reqBody = bytes.NewReader(payload)
	}

	req := httptest.NewRequest(method, path, reqBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+e.mustIssueJWT(userID))

	resp := httptest.NewRecorder()
	e.router.ServeHTTP(resp, req)
	return resp
}

func (e *mailAccountConnectionScenarioEnv) mustIssueJWT(userID uint) string {
	e.t.Helper()
	claims := authdomain.AuthClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(scenarioJWTSecret))
	require.NoError(e.t, err)
	return signed
}

func (e *mailAccountConnectionScenarioEnv) mustDecodeAuthorizeResponse(resp *httptest.ResponseRecorder) scenarioAuthorizeResponse {
	e.t.Helper()
	var out scenarioAuthorizeResponse
	e.mustDecodeResponse(resp, &out)
	return out
}

func (e *mailAccountConnectionScenarioEnv) mustDecodeCallbackResponse(resp *httptest.ResponseRecorder) scenarioCallbackResponse {
	e.t.Helper()
	var out scenarioCallbackResponse
	e.mustDecodeResponse(resp, &out)
	return out
}

func (e *mailAccountConnectionScenarioEnv) mustDecodeListResponse(resp *httptest.ResponseRecorder) scenarioListResponse {
	e.t.Helper()
	var out scenarioListResponse
	e.mustDecodeResponse(resp, &out)
	return out
}

func (e *mailAccountConnectionScenarioEnv) mustDecodeErrorResponse(resp *httptest.ResponseRecorder) scenarioErrorResponse {
	e.t.Helper()
	var out scenarioErrorResponse
	e.mustDecodeResponse(resp, &out)
	return out
}

func (e *mailAccountConnectionScenarioEnv) mustDecodeResponse(resp *httptest.ResponseRecorder, out any) {
	e.t.Helper()
	require.NoError(e.t, json.Unmarshal(resp.Body.Bytes(), out))
}

func (e *mailAccountConnectionScenarioEnv) mustExtractState(authorizationURL string) string {
	e.t.Helper()
	parsed, err := url.Parse(authorizationURL)
	require.NoError(e.t, err)
	state := parsed.Query().Get("state")
	require.NotEmpty(e.t, state)
	return state
}

func (e *mailAccountConnectionScenarioEnv) mustGetPendingState(state string) model.EmailCredential {
	e.t.Helper()
	var row model.EmailCredential
	require.NoError(e.t, e.db.Where("o_auth_state = ?", state).First(&row).Error)
	return row
}

func (e *mailAccountConnectionScenarioEnv) mustExpirePendingState(state string, expiresAt time.Time) {
	e.t.Helper()
	require.NoError(e.t, e.db.Model(&model.EmailCredential{}).
		Where("o_auth_state = ?", state).
		Update("o_auth_state_expires_at", expiresAt).Error)
}

func (e *mailAccountConnectionScenarioEnv) mustListCredentialRows(userID uint) []model.EmailCredential {
	e.t.Helper()
	var rows []model.EmailCredential
	require.NoError(e.t, e.db.Where("user_id = ? AND o_auth_state IS NULL", userID).
		Order("created_at ASC, id ASC").
		Find(&rows).Error)
	return rows
}

func (e *mailAccountConnectionScenarioEnv) mustGetCredentialByID(id uint) model.EmailCredential {
	e.t.Helper()
	var row model.EmailCredential
	require.NoError(e.t, e.db.Where("id = ?", id).First(&row).Error)
	return row
}

func (e *mailAccountConnectionScenarioEnv) mustFindCredentialByAddress(userID uint, gmailAddress string) model.EmailCredential {
	e.t.Helper()
	var row model.EmailCredential
	require.NoError(e.t, e.db.Where("user_id = ? AND gmail_address = ? AND o_auth_state IS NULL", userID, gmailAddress).First(&row).Error)
	return row
}

func (e *mailAccountConnectionScenarioEnv) mustCountCredentials(userID uint) int64 {
	e.t.Helper()
	var count int64
	require.NoError(e.t, e.db.Model(&model.EmailCredential{}).
		Where("user_id = ? AND o_auth_state IS NULL", userID).
		Count(&count).Error)
	return count
}

func pendingStatePlaceholder(state string) string {
	return scenarioPendingStatePrefix + state
}

func (f *scenarioOAuthTokenExchanger) Push(token *oauth2.Token, err error) {
	f.responses = append(f.responses, scenarioExchangeResponse{token: token, err: err})
}

func (f *scenarioOAuthTokenExchanger) Exchange(ctx context.Context, cfg *oauth2.Config, code string) (*oauth2.Token, error) {
	if len(f.responses) == 0 {
		f.t.Fatalf("unexpected oauth exchange call for code=%q", code)
		return nil, errors.New("unexpected oauth exchange call")
	}
	resp := f.responses[0]
	f.responses = f.responses[1:]
	return resp.token, resp.err
}

func (f *scenarioGmailProfileFetcher) Push(email string, err error) {
	f.responses = append(f.responses, scenarioProfileResponse{email: email, err: err})
}

func (f *scenarioGmailProfileFetcher) GetEmailAddress(ctx context.Context, token *oauth2.Token, cfg *oauth2.Config) (string, error) {
	if len(f.responses) == 0 {
		f.t.Fatalf("unexpected gmail profile fetch")
		return "", errors.New("unexpected gmail profile fetch")
	}
	resp := f.responses[0]
	f.responses = f.responses[1:]
	return resp.email, resp.err
}

func (s *scenarioStubAuthUseCase) Login(ctx context.Context, req authdomain.LoginRequest) (string, error) {
	return "dummy-token", nil
}

func (s *scenarioStubAuthUseCase) LoginTokens(ctx context.Context, req authdomain.LoginRequest) (authdomain.AuthTokens, error) {
	return authdomain.AuthTokens{}, nil
}

func (s *scenarioStubAuthUseCase) Register(ctx context.Context, req authdomain.RegisterRequest) (authdomain.User, error) {
	return authdomain.User{}, nil
}

func (s *scenarioStubAuthUseCase) VerifyEmail(ctx context.Context, req authdomain.VerifyEmailRequest) (authdomain.User, error) {
	return authdomain.User{}, nil
}

func (s *scenarioStubAuthUseCase) ResendVerificationEmail(ctx context.Context, req authdomain.ResendVerificationRequest) error {
	return nil
}

func (s *scenarioStubAuthUseCase) Refresh(ctx context.Context, req authdomain.RefreshRequest) (authdomain.AuthTokens, error) {
	return authdomain.AuthTokens{}, nil
}

func (s *scenarioStubAuthUseCase) Logout(ctx context.Context, req authdomain.LogoutRequest) error {
	return nil
}

var _ authapp.AuthUseCaseInterface = (*scenarioStubAuthUseCase)(nil)

func (s *scenarioStubManualMailWorkflowUseCase) Execute(ctx context.Context, cmd manualapp.Command) (manualapp.Result, error) {
	return manualapp.Result{}, nil
}

var _ manualapp.UseCase = (*scenarioStubManualMailWorkflowUseCase)(nil)
