package graph

import (
	"context"

	"github.com/dyike/CortexGo/internal/models"
)

func ShouldContinueDebate(ctx context.Context, input *models.TradingState) (next string, err error) {
	return "", nil
}

func ShouldContinueRiskAnalysis(ctx context.Context, input *models.TradingState) (next string, err error) {
	return "", nil
}
