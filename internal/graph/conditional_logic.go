package graph

import (
	"context"

	"github.com/dyike/CortexGo/consts"
	"github.com/dyike/CortexGo/internal/models"
)

// ConditionalLogic manages workflow transitions and debate continuation logic
type ConditionalLogic struct {
	MaxDebateRounds     int
	MaxRiskDiscussRounds int
}

// NewConditionalLogic creates a new conditional logic manager
func NewConditionalLogic() *ConditionalLogic {
	return &ConditionalLogic{
		MaxDebateRounds:      2,
		MaxRiskDiscussRounds: 2,
	}
}

// DetermineNextPhase determines the next phase based on current state
func (cl *ConditionalLogic) DetermineNextPhase(ctx context.Context, state *models.TradingState) string {
	switch state.Phase {
	case "analysis":
		return cl.handleAnalysisPhase(state)
	case "debate":
		return cl.handleDebatePhase(state)
	case "trading":
		return cl.handleTradingPhase(state)
	case "risk":
		return cl.handleRiskPhase(state)
	default:
		return consts.MarketAnalyst
	}
}

// handleAnalysisPhase manages the sequential execution of analysts
func (cl *ConditionalLogic) handleAnalysisPhase(state *models.TradingState) string {
	// Check if all analysts have completed
	if !state.MarketAnalystComplete {
		return consts.MarketAnalyst
	}
	if !state.SocialAnalystComplete {
		return consts.SocialMediaAnalyst
	}
	if !state.NewsAnalystComplete {
		return consts.NewsAnalyst
	}
	if !state.FundamentalsAnalystComplete {
		return consts.FundamentalsAnalyst
	}
	
	// All analysts complete, move to debate phase
	state.AnalysisPhaseComplete = true
	state.Phase = "debate"
	return consts.BullResearcher
}

// handleDebatePhase manages the bull/bear researcher debate
func (cl *ConditionalLogic) handleDebatePhase(state *models.TradingState) string {
	// Check if we've reached max debate rounds
	if state.InvestmentDebateState.CurrentRound >= cl.MaxDebateRounds {
		state.DebatePhaseComplete = true
		state.Phase = "trading"
		return consts.ResearchManager
	}
	
	// Alternate between Bull and Bear researchers
	if state.InvestmentDebateState.Count%2 == 0 {
		return consts.BullResearcher
	}
	return consts.BearResearcher
}

// handleTradingPhase manages the trading decision phase
func (cl *ConditionalLogic) handleTradingPhase(state *models.TradingState) string {
	if !state.TradingPhaseComplete {
		state.TradingPhaseComplete = true
		state.Phase = "risk"
		return consts.Trader
	}
	// Move to risk phase
	return consts.RiskyAnalyst
}

// handleRiskPhase manages the three-way risk analysis debate
func (cl *ConditionalLogic) handleRiskPhase(state *models.TradingState) string {
	// Check if we've reached max risk discussion rounds
	if state.RiskDebateState.CurrentRound >= cl.MaxRiskDiscussRounds {
		state.RiskPhaseComplete = true
		state.WorkflowComplete = true
		return consts.RiskJudge
	}
	
	// Rotate between Risky, Safe, and Neutral analysts
	switch state.RiskDebateState.LatestSpeaker {
	case consts.RiskyAnalyst:
		return consts.SafeAnalyst
	case consts.SafeAnalyst:
		return consts.NeutralAnalyst
	case consts.NeutralAnalyst:
		return consts.RiskyAnalyst
	default:
		return consts.RiskyAnalyst
	}
}

// ShouldContinueDebate determines if the debate should continue
func (cl *ConditionalLogic) ShouldContinueDebate(state *models.TradingState) bool {
	return state.InvestmentDebateState.CurrentRound < cl.MaxDebateRounds
}

// ShouldContinueRiskDiscussion determines if risk discussion should continue
func (cl *ConditionalLogic) ShouldContinueRiskDiscussion(state *models.TradingState) bool {
	return state.RiskDebateState.CurrentRound < cl.MaxRiskDiscussRounds
}

// MarkAnalystComplete marks an analyst as completed and updates phase tracking
func (cl *ConditionalLogic) MarkAnalystComplete(ctx context.Context, analyst string, state *models.TradingState) {
	switch analyst {
	case consts.MarketAnalyst:
		state.MarketAnalystComplete = true
	case consts.SocialMediaAnalyst:
		state.SocialAnalystComplete = true
	case consts.NewsAnalyst:
		state.NewsAnalystComplete = true
	case consts.FundamentalsAnalyst:
		state.FundamentalsAnalystComplete = true
	}
	
	// Check if analysis phase is complete
	if state.MarketAnalystComplete && state.SocialAnalystComplete && 
	   state.NewsAnalystComplete && state.FundamentalsAnalystComplete {
		state.AnalysisPhaseComplete = true
		state.Phase = "debate"
	}
}

// ConditionalAgentHandOff provides smart routing based on workflow state
func ConditionalAgentHandOff(ctx context.Context, input *models.TradingState) (next string, err error) {
	cl := NewConditionalLogic()
	
	// If Goto is set by agent logic, respect it
	if input.Goto != "" {
		return input.Goto, nil
	}
	
	// Otherwise, use conditional logic to determine next step
	return cl.DetermineNextPhase(ctx, input), nil
}