package test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMailAccountConnectionScenario_CallbackMismatchedState(t *testing.T) {
	env := newMailAccountConnectionScenarioEnv(t)

	authorizeResp := env.authorize(env.userID)
	require.Equal(t, http.StatusOK, authorizeResp.Code)

	authorizeBody := env.mustDecodeAuthorizeResponse(authorizeResp)
	state := env.mustExtractState(authorizeBody.AuthorizationURL)

	resp := env.callback(env.userID, "ignored-auth-code", state+"-mismatch")
	require.Equal(t, http.StatusConflict, resp.Code)

	errorBody := env.mustDecodeErrorResponse(resp)
	require.Equal(t, "oauth_state_mismatch", errorBody.Error.Code)
	require.Zero(t, env.mustCountCredentials(env.userID))
}

func TestMailAccountConnectionScenario_CallbackExpiredState(t *testing.T) {
	env := newMailAccountConnectionScenarioEnv(t)

	authorizeResp := env.authorize(env.userID)
	require.Equal(t, http.StatusOK, authorizeResp.Code)

	authorizeBody := env.mustDecodeAuthorizeResponse(authorizeResp)
	state := env.mustExtractState(authorizeBody.AuthorizationURL)
	env.mustExpirePendingState(state, time.Now().UTC().Add(-1*time.Minute))

	resp := env.callback(env.userID, "ignored-auth-code", state)
	require.Equal(t, http.StatusConflict, resp.Code)

	errorBody := env.mustDecodeErrorResponse(resp)
	require.Equal(t, "oauth_state_expired", errorBody.Error.Code)
	require.Zero(t, env.mustCountCredentials(env.userID))
}

func TestMailAccountConnectionScenario_DeleteOtherUsersConnectionKeepsOriginalCredential(t *testing.T) {
	env := newMailAccountConnectionScenarioEnv(t)

	authorizeResp := env.authorize(env.userID)
	require.Equal(t, http.StatusOK, authorizeResp.Code)

	authorizeBody := env.mustDecodeAuthorizeResponse(authorizeResp)
	state := env.mustExtractState(authorizeBody.AuthorizationURL)
	env.queueOAuthSuccess("scenario-user-1@gmail.com", "raw-access-token-1", "raw-refresh-token-1", time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC))

	callbackResp := env.callback(env.userID, "auth-code-1", state)
	require.Equal(t, http.StatusOK, callbackResp.Code)

	callbackBody := env.mustDecodeCallbackResponse(callbackResp)
	require.Equal(t, "Gmail連携が完了しました。", callbackBody.Message)

	require.Equal(t, int64(1), env.mustCountCredentials(env.userID))
	credentialRows := env.mustListCredentialRows(env.userID)
	require.Len(t, credentialRows, 1)

	createdCredential := credentialRows[0]
	require.Equal(t, env.userID, createdCredential.UserID)
	require.Equal(t, "gmail", createdCredential.Type)
	require.Equal(t, "scenario-user-1@gmail.com", createdCredential.GmailAddress)
	require.NotEqual(t, "raw-access-token-1", createdCredential.AccessToken)
	require.NotEqual(t, "raw-refresh-token-1", createdCredential.RefreshToken)

	resp := env.disconnect(env.otherUserID, createdCredential.ID)
	require.Equal(t, http.StatusNotFound, resp.Code)

	errorBody := env.mustDecodeErrorResponse(resp)
	require.Equal(t, "mail_account_connection_not_found", errorBody.Error.Code)
	require.Equal(t, int64(1), env.mustCountCredentials(env.userID))

	remainingCredential := env.mustGetCredentialByID(createdCredential.ID)
	require.Equal(t, createdCredential.ID, remainingCredential.ID)
	require.Equal(t, env.userID, remainingCredential.UserID)
	require.Equal(t, "scenario-user-1@gmail.com", remainingCredential.GmailAddress)
}
