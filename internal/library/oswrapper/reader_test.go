package oswrapper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"business/internal/library/secret"
)

type stubSecretClient struct {
	values map[string]string
	calls  map[string]int
}

var _ secret.Client = (*stubSecretClient)(nil)

func (s *stubSecretClient) GetValue(_ context.Context, key string) (string, error) {
	if s.calls == nil {
		s.calls = map[string]int{}
	}
	s.calls[key]++
	return s.values[key], nil
}

func TestGetEnv_UsesEnvironmentVariablesInLocal(t *testing.T) {
	t.Setenv("APP", "local")
	t.Setenv("MYSQL_USER", "env-user")

	osw, err := New(nil, nil)
	require.NoError(t, err)

	got, err := osw.GetEnv("MYSQL_USER")
	require.NoError(t, err)
	require.Equal(t, "env-user", got)
}

func TestGetEnv_UsesDbCacheAndAppSecretInNonLocal(t *testing.T) {
	t.Setenv("APP", "prod")

	appSecretClient := &stubSecretClient{
		values: map[string]string{
			"OPENAI_API_KEY": "app-secret",
			"DB_HOST":        "db-host",
			"DB_PORT":        "3306",
		},
	}
	dbSecretClient := &stubSecretClient{
		values: map[string]string{
			"username": "db-user",
			"password": "db-pass",
		},
	}

	osw, err := New(appSecretClient, dbSecretClient)
	require.NoError(t, err)

	dbUser, err := osw.GetEnv("MYSQL_USER")
	require.NoError(t, err)
	require.Equal(t, "db-user", dbUser)

	dbHost, err := osw.GetEnv("DB_HOST")
	require.NoError(t, err)
	require.Equal(t, "db-host", dbHost)

	apiKey, err := osw.GetEnv("OPENAI_API_KEY")
	require.NoError(t, err)
	require.Equal(t, "app-secret", apiKey)

	require.Equal(t, 1, dbSecretClient.calls["username"])
	require.Equal(t, 1, appSecretClient.calls["DB_HOST"])
	require.Equal(t, 1, appSecretClient.calls["OPENAI_API_KEY"])
}
