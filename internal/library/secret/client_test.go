package secret

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSecret_JSON(t *testing.T) {
	raw := `{"user":"alice","password":"secret"}`
	got, err := parseSecret(raw)
	require.NoError(t, err)
	require.Equal(t, map[string]string{"user": "alice", "password": "secret"}, got)
}

func TestParseSecret_NotJSON(t *testing.T) {
	_, err := parseSecret("plain")
	require.Error(t, err)
}
