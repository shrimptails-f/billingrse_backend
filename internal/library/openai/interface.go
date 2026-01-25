package openai

import (
	cd "business/internal/common/domain"
	"context"
)

type UseCaserInterface interface {
	Chat(ctx context.Context, prompt string) ([]cd.AnalysisResult, error)
}
