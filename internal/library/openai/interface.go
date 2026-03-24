package openai

import (
	"context"
)

type UseCaserInterface interface {
	Chat(ctx context.Context, prompt string) (string, error)
}
