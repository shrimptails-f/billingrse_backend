package test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMailAccountConnectionMultipleGmailAccountsScenario(t *testing.T) {
	env := newMailAccountConnectionScenarioEnv(t)

	firstAuthorize := env.authorize(env.userID)
	require.Equal(t, http.StatusOK, firstAuthorize.Code)

	firstAuthorizeBody := env.mustDecodeAuthorizeResponse(firstAuthorize)
	firstState := env.mustExtractState(firstAuthorizeBody.AuthorizationURL)
	firstPending := env.mustGetPendingState(firstState)
	require.Equal(t, env.userID, firstPending.UserID)
	require.NotNil(t, firstPending.OAuthState)
	require.Equal(t, firstState, *firstPending.OAuthState)
	require.NotNil(t, firstPending.OAuthStateExpiresAt)

	firstRawAccess := "first-access-token"
	firstRawRefresh := "first-refresh-token"
	env.queueOAuthSuccess("first@gmail.com", firstRawAccess, firstRawRefresh, time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC))

	firstCallback := env.callback(env.userID, "first-code", firstState)
	require.Equal(t, http.StatusOK, firstCallback.Code)
	firstCallbackBody := env.mustDecodeCallbackResponse(firstCallback)
	assert.Equal(t, "Gmail連携が完了しました。", firstCallbackBody.Message)

	require.Equal(t, int64(1), env.mustCountCredentials(env.userID))
	firstRow := env.mustFindCredentialByAddress(env.userID, "first@gmail.com")
	require.Equal(t, "gmail", firstRow.Type)
	require.Equal(t, "first@gmail.com", firstRow.GmailAddress)
	require.NotEmpty(t, firstRow.AccessToken)
	require.NotEmpty(t, firstRow.AccessTokenDigest)
	require.NotEmpty(t, firstRow.RefreshToken)
	require.NotEmpty(t, firstRow.RefreshTokenDigest)
	assert.NotEqual(t, firstRawAccess, firstRow.AccessToken)
	assert.NotEqual(t, firstRawRefresh, firstRow.RefreshToken)

	secondAuthorize := env.authorize(env.userID)
	require.Equal(t, http.StatusOK, secondAuthorize.Code)

	secondAuthorizeBody := env.mustDecodeAuthorizeResponse(secondAuthorize)
	secondState := env.mustExtractState(secondAuthorizeBody.AuthorizationURL)
	secondPending := env.mustGetPendingState(secondState)
	require.Equal(t, env.userID, secondPending.UserID)
	require.NotNil(t, secondPending.OAuthState)
	require.Equal(t, secondState, *secondPending.OAuthState)
	require.NotNil(t, secondPending.OAuthStateExpiresAt)

	secondRawAccess := "second-access-token"
	secondRawRefresh := "second-refresh-token"
	env.queueOAuthSuccess("second@gmail.com", secondRawAccess, secondRawRefresh, time.Date(2026, 3, 20, 10, 5, 0, 0, time.UTC))

	secondCallback := env.callback(env.userID, "second-code", secondState)
	require.Equal(t, http.StatusOK, secondCallback.Code)
	secondCallbackBody := env.mustDecodeCallbackResponse(secondCallback)
	assert.Equal(t, "Gmail連携が完了しました。", secondCallbackBody.Message)

	require.Equal(t, int64(2), env.mustCountCredentials(env.userID))
	secondRow := env.mustFindCredentialByAddress(env.userID, "second@gmail.com")
	require.Equal(t, "gmail", secondRow.Type)
	require.Equal(t, "second@gmail.com", secondRow.GmailAddress)
	require.NotEqual(t, firstRow.ID, secondRow.ID)
	require.NotEmpty(t, secondRow.AccessToken)
	require.NotEmpty(t, secondRow.AccessTokenDigest)
	require.NotEmpty(t, secondRow.RefreshToken)
	require.NotEmpty(t, secondRow.RefreshTokenDigest)
	assert.NotEqual(t, secondRawAccess, secondRow.AccessToken)
	assert.NotEqual(t, secondRawRefresh, secondRow.RefreshToken)

	listResp := env.listConnections(env.userID)
	require.Equal(t, http.StatusOK, listResp.Code)
	listBody := env.mustDecodeListResponse(listResp)
	require.Len(t, listBody.Items, 2)

	identifiers := []string{listBody.Items[0].AccountIdentifier, listBody.Items[1].AccountIdentifier}
	assert.ElementsMatch(t, []string{"first@gmail.com", "second@gmail.com"}, identifiers)
	for _, item := range listBody.Items {
		assert.Equal(t, "gmail", item.Provider)
		assert.NotZero(t, item.ID)
		assert.False(t, item.CreatedAt.IsZero())
		assert.False(t, item.UpdatedAt.IsZero())
	}
}
