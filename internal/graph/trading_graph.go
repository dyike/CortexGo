package graph

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/compose"
	"github.com/dyike/CortexGo/internal/agents"
	"github.com/dyike/CortexGo/internal/config"
	"github.com/dyike/CortexGo/internal/models"
)

type TradingAgentsGraph struct {
	config      *config.Config
	orchestrator compose.Runnable[*agents.TradingState, *agents.TradingState]
	debug       bool
}

func NewTradingAgentsGraph(debug bool, cfg *config.Config) *TradingAgentsGraph {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	ctx := context.Background()
	orchestrator := agents.NewTradingOrchestrator[*agents.TradingState, *agents.TradingState, *agents.TradingState](
		ctx,
		func(ctx context.Context) *agents.TradingState {
			return &agents.TradingState{}
		},
	)

	return &TradingAgentsGraph{
		config:       cfg,
		orchestrator: orchestrator,
		debug:        debug,
	}
}

func (g *TradingAgentsGraph) Propagate(symbol string, date string) (*agents.TradingState, *models.TradingDecision, error) {
	ctx := context.Background()

	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid date format: %v", err)
	}

	userPrompt := fmt.Sprintf("Analyze trading opportunities for %s on %s", symbol, date)
	state := agents.NewTradingState(symbol, parsedDate, userPrompt)

	// Set market data
	state.MarketData = &models.MarketData{
		Symbol:    symbol,
		Price:     125.50,
		Volume:    1000000,
		Timestamp: parsedDate,
		High:      127.80,
		Low:       123.20,
		Open:      124.00,
		Close:     125.50,
	}

	if g.debug {
		fmt.Printf("Processing %s for date %s using eino orchestrator\n", symbol, date)
	}

	// Run the orchestrator
	result, err := g.orchestrator.Invoke(ctx, state)
	if err != nil {
		return nil, nil, fmt.Errorf("orchestrator failed: %v", err)
	}

	if g.debug {
		fmt.Printf("Trading decision completed for %s\n", symbol)
	}

	return result, result.Decision, nil
}

func (g *TradingAgentsGraph) ReflectAndRemember(positionReturns float64) error {
	if g.debug {
		fmt.Printf("Reflecting on position returns: %.2f\n", positionReturns)
	}

	return nil
}
