package presentation

import (
	"os"
	"sync"
	"testing"
)

var appEnvMu sync.Mutex

// withAppEnv serializes APP environment mutations for tests and restores the
// previous value after the test completes.
func withAppEnv(t *testing.T, value string) {
	t.Helper()

	appEnvMu.Lock()

	prev, ok := os.LookupEnv("APP")
	if err := os.Setenv("APP", value); err != nil {
		appEnvMu.Unlock()
		t.Fatalf("failed to set APP: %v", err)
	}

	t.Cleanup(func() {
		if !ok {
			_ = os.Unsetenv("APP")
		} else {
			_ = os.Setenv("APP", prev)
		}
		appEnvMu.Unlock()
	})
}
