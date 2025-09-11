package graph

import (
	"context"
	"strings"

	"github.com/cloudwego/eino/compose"
	"github.com/dyike/CortexGo/consts"
	"github.com/dyike/CortexGo/internal/models"
)

func ShouldContinueDebate(ctx context.Context, input string) (next string, err error) {
	next = consts.BullResearcher
	_ = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		if state.InvestmentDebateState.Count >= 2 {
			next = consts.ResearchManager
			return nil
		}
		curResp := state.InvestmentDebateState.CurrentResponse
		if strings.HasPrefix(curResp, "Bull") {
			next = consts.BearResearcher
			return nil
		}
		return nil
	})
	return next, nil
}

func ShouldContinueRiskAnalysis(ctx context.Context, input string) (next string, err error) {
	next = consts.RiskyAnalyst
	_ = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		if state.RiskDebateState.Count >= 3 {
			next = consts.RiskJudge
			return nil
		}
		latestSpeaker := state.RiskDebateState.LatestSpeaker
		if strings.HasPrefix(latestSpeaker, "Risky") {
			next = consts.SafeAnalyst
			return nil
		}
		if strings.HasPrefix(latestSpeaker, "Safe") {
			next = consts.NeutralAnalyst
			return nil
		}
		return nil
	})
	return next, nil
}
