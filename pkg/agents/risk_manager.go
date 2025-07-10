package agents

import (
	"context"
	"fmt"

	"github.com/dyike/CortexGo/pkg/config"
	"github.com/dyike/CortexGo/pkg/models"
)

type RiskManager struct {
	*BaseAgent
	maxRiskThreshold float64
	maxPositionSize  float64
}

func NewRiskManager(config *config.Config) *RiskManager {
	return &RiskManager{
		BaseAgent:        NewBaseAgent("risk_manager", config),
		maxRiskThreshold: 0.7,
		maxPositionSize:  1000.0,
	}
}

func (r *RiskManager) Process(ctx context.Context, state *models.AgentState) (*models.AgentState, error) {
	if state.Decision == nil {
		return state, fmt.Errorf("no trading decision available for risk assessment")
	}

	approved, adjustedDecision := r.assessRisk(state.Decision)
	
	if !approved {
		adjustedDecision.Action = "HOLD"
		adjustedDecision.Quantity = 0
		adjustedDecision.Reason = fmt.Sprintf("RISK REJECTED: %s", adjustedDecision.Reason)
	}

	state.Decision = adjustedDecision
	
	riskReport := models.AnalysisReport{
		Analyst:    r.Name(),
		Symbol:     state.CurrentSymbol,
		Date:       state.CurrentDate,
		Analysis:   r.generateRiskAnalysis(state.Decision, approved),
		Rating:     r.getRiskRating(state.Decision.Risk),
		Confidence: 0.9,
		Metrics: map[string]interface{}{
			"approved":         approved,
			"original_risk":    state.Decision.Risk,
			"position_size":    state.Decision.Quantity,
			"risk_threshold":   r.maxRiskThreshold,
			"position_limit":   r.maxPositionSize,
		},
	}
	
	state.Reports = append(state.Reports, riskReport)
	return state, nil
}

func (r *RiskManager) assessRisk(decision *models.TradingDecision) (bool, *models.TradingDecision) {
	adjustedDecision := *decision
	approved := true

	if decision.Risk > r.maxRiskThreshold {
		approved = false
		adjustedDecision.Reason = fmt.Sprintf("High risk (%.2f > %.2f): %s", 
			decision.Risk, r.maxRiskThreshold, decision.Reason)
	}

	if decision.Quantity > r.maxPositionSize {
		adjustedDecision.Quantity = r.maxPositionSize
		adjustedDecision.Reason = fmt.Sprintf("Position size limited to %.0f: %s", 
			r.maxPositionSize, decision.Reason)
	}

	if decision.Confidence < 0.3 {
		approved = false
		adjustedDecision.Reason = fmt.Sprintf("Low confidence (%.2f < 0.3): %s", 
			decision.Confidence, decision.Reason)
	}

	return approved, &adjustedDecision
}

func (r *RiskManager) generateRiskAnalysis(decision *models.TradingDecision, approved bool) string {
	status := "APPROVED"
	if !approved {
		status = "REJECTED"
	}

	return fmt.Sprintf(
		"Risk Management Analysis for %s:\n"+
			"Decision: %s\n"+
			"Risk Level: %.2f\n"+
			"Position Size: %.0f\n"+
			"Confidence: %.2f\n"+
			"Status: %s\n"+
			"Rationale: %s",
		decision.Symbol,
		decision.Action,
		decision.Risk,
		decision.Quantity,
		decision.Confidence,
		status,
		decision.Reason,
	)
}

func (r *RiskManager) getRiskRating(risk float64) string {
	if risk > 0.7 {
		return "HIGH_RISK"
	} else if risk > 0.4 {
		return "MEDIUM_RISK"
	}
	return "LOW_RISK"
}