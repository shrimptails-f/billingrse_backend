//go:build integration
// +build integration

package secret

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIntegration_GetValue(t *testing.T) {
	ctx := context.Background()
	client, err := New(ctx)
	assert.NoError(t, err)

	val, err := client.GetValue(ctx, "test")

	assert.NoError(t, err)
	assert.NotEmpty(t, val)
	assert.Equal(t, val, "test")
}
