package graph

import (
	"github.com/dyike/CortexGo/internal/models"
)

// ConditionalLogic manages debate and risk discussion cycles
type ConditionalLogic struct {
	MaxDebateRounds      int
	MaxRiskDiscussRounds int
}

// NewConditionalLogic creates a new conditional logic manager
func NewConditionalLogic() *ConditionalLogic {
	return &ConditionalLogic{
		MaxDebateRounds:      1, // Match LangGraph's max_debate_rounds
		MaxRiskDiscussRounds: 1, // Match LangGraph's max_risk_discuss_rounds
	}
}

// ShouldContinueDebate determines if the debate should continue - matches LangGraph logic
func (cl *ConditionalLogic) ShouldContinueDebate(state *models.TradingState) bool {
	// Check if we've reached max debate rounds (2 * max_debate_rounds for back-and-forth)
	return state.InvestmentDebateState.Count < 2*cl.MaxDebateRounds
}

// ShouldContinueRiskDiscussion determines if risk discussion should continue - matches LangGraph logic
func (cl *ConditionalLogic) ShouldContinueRiskDiscussion(state *models.TradingState) bool {
	// Check if we've reached max risk discussion rounds (3 * max_risk_discuss_rounds for three-way)
	return state.RiskDebateState.Count < 3*cl.MaxRiskDiscussRounds
}

// UpdateDebateState updates the debate state counters - matches LangGraph behavior
func (cl *ConditionalLogic) UpdateDebateState(state *models.TradingState, speaker string) {
	state.InvestmentDebateState.Count++
	state.InvestmentDebateState.CurrentResponse = speaker

	// Update round counter every 2 exchanges (bull + bear = 1 round)
	if state.InvestmentDebateState.Count%2 == 0 {
		state.InvestmentDebateState.CurrentRound++
	}
}

// UpdateRiskState updates the risk discussion state counters - matches LangGraph behavior
func (cl *ConditionalLogic) UpdateRiskState(state *models.TradingState, speaker string) {
	state.RiskDebateState.Count++
	state.RiskDebateState.LatestSpeaker = speaker

	// Update round counter every 3 exchanges (risky + safe + neutral = 1 round)
	if state.RiskDebateState.Count%3 == 0 {
		state.RiskDebateState.CurrentRound++
	}
}

