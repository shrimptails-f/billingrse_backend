package test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMailAccountConnection_FullFlowScenario(t *testing.T) {
	env := newMailAccountConnectionScenarioEnv(t)

	authorizeResp := env.authorize(env.userID)
	require.Equal(t, 200, authorizeResp.Code)

	authorizeBody := env.mustDecodeAuthorizeResponse(authorizeResp)
	require.NotEmpty(t, authorizeBody.AuthorizationURL)
	require.False(t, authorizeBody.ExpiresAt.IsZero())

	state := env.mustExtractState(authorizeBody.AuthorizationURL)
	pending := env.mustGetPendingState(state)
	require.Equal(t, env.userID, pending.UserID)
	require.NotNil(t, pending.OAuthState)
	require.Equal(t, state, *pending.OAuthState)
	require.NotNil(t, pending.OAuthStateExpiresAt)
	require.True(t, pending.OAuthStateExpiresAt.After(pending.CreatedAt))
	require.Equal(t, pendingStatePlaceholder(state), pending.GmailAddress)

	rawAccessToken := "access-token-raw"
	rawRefreshToken := "refresh-token-raw"
	expiry := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	env.queueOAuthSuccess("user@gmail.com", rawAccessToken, rawRefreshToken, expiry)

	callbackResp := env.callback(env.userID, "auth-code-raw", state)
	require.Equal(t, 200, callbackResp.Code)

	callbackBody := env.mustDecodeCallbackResponse(callbackResp)
	require.Equal(t, "Gmail連携が完了しました。", callbackBody.Message)
	require.Equal(t, int64(1), env.mustCountCredentials(env.userID))

	stored := env.mustFindCredentialByAddress(env.userID, "user@gmail.com")
	require.Equal(t, env.userID, stored.UserID)
	require.Equal(t, "gmail", stored.Type)
	require.NotEmpty(t, stored.AccessToken)
	require.NotEmpty(t, stored.RefreshToken)
	assert.NotEqual(t, rawAccessToken, stored.AccessToken)
	assert.NotEqual(t, rawRefreshToken, stored.RefreshToken)
	assert.NotEqual(t, rawAccessToken, stored.AccessTokenDigest)
	assert.NotEqual(t, rawRefreshToken, stored.RefreshTokenDigest)

	listResp := env.listConnections(env.userID)
	require.Equal(t, 200, listResp.Code)

	listBody := env.mustDecodeListResponse(listResp)
	require.Len(t, listBody.Items, 1)
	require.Equal(t, int64(1), env.mustCountCredentials(env.userID))
	require.Equal(t, stored.ID, listBody.Items[0].ID)
	require.Equal(t, "gmail", listBody.Items[0].Provider)
	require.Equal(t, "user@gmail.com", listBody.Items[0].AccountIdentifier)
	require.False(t, listBody.Items[0].CreatedAt.IsZero())
	require.False(t, listBody.Items[0].UpdatedAt.IsZero())

	disconnectResp := env.disconnect(env.userID, listBody.Items[0].ID)
	require.Equal(t, 204, disconnectResp.Code)
	require.Empty(t, disconnectResp.Body.String())

	finalListResp := env.listConnections(env.userID)
	require.Equal(t, 200, finalListResp.Code)
	finalListBody := env.mustDecodeListResponse(finalListResp)
	require.Empty(t, finalListBody.Items)
	require.Equal(t, int64(0), env.mustCountCredentials(env.userID))
}
