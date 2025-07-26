package agents

import (
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/internal/models"
)

type TradingState struct {
	Messages         []*schema.Message          `json:"messages"`
	CurrentSymbol    string                     `json:"current_symbol"`
	CurrentDate      time.Time                  `json:"current_date"`
	MarketData       *models.MarketData         `json:"market_data"`
	Reports          []models.AnalysisReport    `json:"reports"`
	Discussions      []models.AnalystDiscussion `json:"discussions"`
	TeamConsensus    *models.Consensus          `json:"team_consensus"`
	Decision         *models.TradingDecision    `json:"decision"`
	Metadata         map[string]interface{}     `json:"metadata"`
	Goto             string                     `json:"goto"`
	MaxIterations    int                        `json:"max_iterations"`
	CurrentIteration int                        `json:"current_iteration"`
}

func NewTradingState(symbol string, date time.Time, userPrompt string) *TradingState {
	return &TradingState{
		Messages: []*schema.Message{
			schema.UserMessage(userPrompt),
		},
		CurrentSymbol: symbol,
		CurrentDate:   date,
		MarketData: &models.MarketData{
			Symbol:    symbol,
			Timestamp: date,
		},
		Reports:          []models.AnalysisReport{},
		Discussions:      []models.AnalystDiscussion{},
		Metadata:         make(map[string]interface{}),
		MaxIterations:    10,
		CurrentIteration: 0,
		Goto:             "coordinator",
	}
}
