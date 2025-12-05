package graph

import (
	"context"
	"strings"

	"github.com/cloudwego/eino/compose"
	"github.com/dyike/CortexGo/consts"
	"github.com/dyike/CortexGo/models"
)

func ShouldContinueDebate(ctx context.Context, _ string) (string, error) {
	var state *models.TradingState
	_ = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, s *models.TradingState) error {
		state = s
		return nil
	})
	if state == nil || state.InvestmentDebateState == nil {
		return consts.BullResearcher, nil
	}
	if state.InvestmentDebateState.Count >= 2 {
		return consts.ResearchManager, nil
	}
	curResp := state.InvestmentDebateState.CurrentResponse
	if strings.HasPrefix(curResp, "Bull") {
		return consts.BearResearcher, nil
	}
	return consts.BullResearcher, nil
}

func ShouldContinueRiskAnalysis(ctx context.Context, _ string) (string, error) {
	var state *models.TradingState
	_ = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, s *models.TradingState) error {
		state = s
		return nil
	})
	if state == nil || state.RiskDebateState == nil {
		return consts.RiskyAnalyst, nil
	}
	if state.RiskDebateState.Count >= 3 {
		return consts.RiskJudge, nil
	}
	latestSpeaker := state.RiskDebateState.LatestSpeaker
	if strings.HasPrefix(latestSpeaker, "Risky") {
		return consts.SafeAnalyst, nil
	}
	if strings.HasPrefix(latestSpeaker, "Safe") {
		return consts.NeutralAnalyst, nil
	}
	return consts.RiskyAnalyst, nil
}
