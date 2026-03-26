package infrastructure

import (
	"strings"
	"testing"
	"time"
)

type billingRepoFixedClock struct {
	now time.Time
}

func (c *billingRepoFixedClock) Now() time.Time {
	return c.now
}

func (c *billingRepoFixedClock) After(d time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	ch <- c.now.Add(d)
	return ch
}

func skipIfBillingRepoDBUnavailable(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "dial tcp") || strings.Contains(err.Error(), "lookup mysql") {
		t.Skipf("Skipping repository integration test: %v", err)
	}
}
