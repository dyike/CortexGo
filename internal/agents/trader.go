package agents

import (
	"context"
	"fmt"

	"github.com/dyike/CortexGo/internal/config"
	"github.com/dyike/CortexGo/internal/models"
)

type Trader struct {
	*BaseAgent
}

func NewTrader(config *config.Config) *Trader {
	return &Trader{
		BaseAgent: NewBaseAgent("trader", config),
	}
}

func (t *Trader) Process(ctx context.Context, state *models.AgentState) (*models.AgentState, error) {
	if len(state.Reports) == 0 {
		return state, fmt.Errorf("no reports available for trading decision")
	}

	var researchReport *models.AnalysisReport
	for _, report := range state.Reports {
		if report.Analyst == "researcher" {
			researchReport = &report
			break
		}
	}

	if researchReport == nil {
		return state, fmt.Errorf("no research report available for trading decision")
	}

	decision := t.makeTradingDecision(state, researchReport)
	state.Decision = decision

	return state, nil
}

func (t *Trader) makeTradingDecision(state *models.AgentState, researchReport *models.AnalysisReport) *models.TradingDecision {
	var action string
	var quantity float64
	var reason string

	switch researchReport.Rating {
	case "BUY":
		action = "BUY"
		quantity = t.calculateQuantity(researchReport.Confidence, "BUY")
		reason = fmt.Sprintf("Strong consensus BUY signal with %.2f confidence", researchReport.Confidence)
	case "SELL":
		action = "SELL"
		quantity = t.calculateQuantity(researchReport.Confidence, "SELL")
		reason = fmt.Sprintf("Consensus SELL signal with %.2f confidence", researchReport.Confidence)
	default:
		action = "HOLD"
		quantity = 0
		reason = fmt.Sprintf("Neutral or uncertain signal (%.2f confidence), maintaining current position", researchReport.Confidence)
	}

	var price float64
	if state.MarketData != nil {
		price = state.MarketData.Price
	} else {
		price = 100.0 // Default price if market data not available
	}

	risk := t.calculateRisk(researchReport)

	return &models.TradingDecision{
		Symbol:     state.CurrentSymbol,
		Date:       state.CurrentDate,
		Action:     action,
		Quantity:   quantity,
		Price:      price,
		Reason:     reason,
		Confidence: researchReport.Confidence,
		Risk:       risk,
	}
}

func (t *Trader) calculateQuantity(confidence float64, action string) float64 {
	if action == "HOLD" {
		return 0
	}

	baseQuantity := 100.0

	if confidence > 0.8 {
		return baseQuantity * 1.5
	} else if confidence > 0.6 {
		return baseQuantity
	} else if confidence > 0.4 {
		return baseQuantity * 0.5
	}

	return baseQuantity * 0.25
}

func (t *Trader) calculateRisk(researchReport *models.AnalysisReport) float64 {
	baseRisk := 0.5

	if researchReport.Confidence > 0.8 {
		return baseRisk * 0.7
	} else if researchReport.Confidence > 0.6 {
		return baseRisk
	} else if researchReport.Confidence > 0.4 {
		return baseRisk * 1.3
	}

	return baseRisk * 1.6
}
