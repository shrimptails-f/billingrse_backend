package test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMailAccountConnectionRelinkScenario(t *testing.T) {
	t.Parallel()

	env := newMailAccountConnectionScenarioEnv(t)

	firstAuthorizeResp := env.authorize(env.userID)
	require.Equal(t, 200, firstAuthorizeResp.Code)

	firstState := env.mustExtractState(env.mustDecodeAuthorizeResponse(firstAuthorizeResp).AuthorizationURL)
	require.NotEmpty(t, firstState)

	firstIssuedAt := time.Date(2026, 3, 20, 2, 0, 0, 0, time.UTC)
	firstExpiry := firstIssuedAt.Add(1 * time.Hour)
	env.queueOAuthSuccess("User@Gmail.com", "first-access-token", "first-refresh-token", firstExpiry)

	firstCallbackResp := env.callback(env.userID, "first-auth-code", firstState)
	require.Equal(t, 200, firstCallbackResp.Code)
	assert.Equal(t, "Gmail連携が完了しました。", env.mustDecodeCallbackResponse(firstCallbackResp).Message)

	require.Equal(t, int64(1), env.mustCountCredentials(env.userID))
	firstRow := env.mustListCredentialRows(env.userID)[0]
	require.NotEmpty(t, firstRow.ID)
	assert.Equal(t, "gmail", firstRow.Type)
	assert.Equal(t, "user@gmail.com", firstRow.GmailAddress)
	assert.NotEqual(t, "first-access-token", firstRow.AccessToken)
	assert.NotEqual(t, "first-refresh-token", firstRow.RefreshToken)

	firstStored := env.mustGetCredentialByID(firstRow.ID)
	require.Equal(t, firstRow.ID, firstStored.ID)
	require.Equal(t, firstRow.AccessToken, firstStored.AccessToken)
	require.Equal(t, firstRow.RefreshToken, firstStored.RefreshToken)

	secondAuthorizeResp := env.authorize(env.userID)
	require.Equal(t, 200, secondAuthorizeResp.Code)

	secondState := env.mustExtractState(env.mustDecodeAuthorizeResponse(secondAuthorizeResp).AuthorizationURL)
	require.NotEmpty(t, secondState)

	secondIssuedAt := time.Date(2026, 3, 20, 3, 0, 0, 0, time.UTC)
	secondExpiry := secondIssuedAt.Add(1 * time.Hour)
	env.queueOAuthSuccess("user@gmail.com", "second-access-token", "second-refresh-token", secondExpiry)

	secondCallbackResp := env.callback(env.userID, "second-auth-code", secondState)
	require.Equal(t, 200, secondCallbackResp.Code)
	assert.Equal(t, "Gmail連携が完了しました。", env.mustDecodeCallbackResponse(secondCallbackResp).Message)

	require.Equal(t, int64(1), env.mustCountCredentials(env.userID))
	secondRow := env.mustListCredentialRows(env.userID)[0]
	require.Equal(t, firstRow.ID, secondRow.ID)
	assert.Equal(t, "gmail", secondRow.Type)
	assert.Equal(t, "user@gmail.com", secondRow.GmailAddress)
	assert.NotEqual(t, firstRow.AccessToken, secondRow.AccessToken)
	assert.NotEqual(t, firstRow.RefreshToken, secondRow.RefreshToken)
	assert.NotEqual(t, "second-access-token", secondRow.AccessToken)
	assert.NotEqual(t, "second-refresh-token", secondRow.RefreshToken)

	secondStored := env.mustGetCredentialByID(secondRow.ID)
	require.Equal(t, secondRow.AccessToken, secondStored.AccessToken)
	require.Equal(t, secondRow.RefreshToken, secondStored.RefreshToken)

	listResp := env.listConnections(env.userID)
	require.Equal(t, 200, listResp.Code)
	listBody := env.mustDecodeListResponse(listResp)
	require.Len(t, listBody.Items, 1)
	assert.Equal(t, secondRow.ID, listBody.Items[0].ID)
	assert.Equal(t, "gmail", listBody.Items[0].Provider)
	assert.Equal(t, "user@gmail.com", listBody.Items[0].AccountIdentifier)

	disconnectResp := env.disconnect(env.userID, secondRow.ID)
	require.Equal(t, 204, disconnectResp.Code)
	assert.Empty(t, disconnectResp.Body.String())

	finalListResp := env.listConnections(env.userID)
	require.Equal(t, 200, finalListResp.Code)
	finalListBody := env.mustDecodeListResponse(finalListResp)
	assert.Empty(t, finalListBody.Items)
	require.Equal(t, int64(0), env.mustCountCredentials(env.userID))
}
