package graph

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cloudwego/eino/compose"
	"github.com/dyike/CortexGo/internal/agents"
	"github.com/dyike/CortexGo/internal/config"
	einodebug "github.com/dyike/CortexGo/internal/debug"
	"github.com/dyike/CortexGo/internal/models"
)

type TradingAgentsGraph struct {
	config       *config.Config
	orchestrator compose.Runnable[*models.TradingState, *models.TradingState]
	debug        bool
	debugger     *einodebug.EinoDebugger
}

func NewTradingAgentsGraph(debug bool, cfg *config.Config) *TradingAgentsGraph {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	// Initialize agents infrastructure (required before creating orchestrator)
	if err := agents.InitModel(); err != nil {
		log.Printf("[TradingGraph] Failed to initialize agents model: %v", err)
	}

	ctx := context.Background()
	orchestrator := NewTradingOrchestrator[*models.TradingState, *models.TradingState, *models.TradingState](
		ctx,
		func(ctx context.Context) *models.TradingState {
			return &models.TradingState{}
		},
	)

	return &TradingAgentsGraph{
		config:       cfg,
		orchestrator: orchestrator,
		debug:        debug,
		debugger:     nil, // Will be initialized only when debug command is called
	}
}

func (g *TradingAgentsGraph) Propagate(symbol string, date string) (*models.TradingState, *models.TradingDecision, error) {
	ctx := context.Background()

	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid date format: %v", err)
	}

	userPrompt := fmt.Sprintf("Analyze trading opportunities for %s on %s", symbol, date)
	state := models.NewTradingState(symbol, parsedDate, userPrompt)

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

func (g *TradingAgentsGraph) StartDebugServer() error {
	if g.debugger != nil {
		return fmt.Errorf("debug server is already running")
	}

	// Initialize debugger
	g.debugger = einodebug.NewEinoDebugger(g.config)
	if err := g.debugger.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize debug server: %w", err)
	}

	log.Printf("[TradingGraph] Eino debug interface available at: %s", g.debugger.GetDebugURL())
	return nil
}

func (g *TradingAgentsGraph) IsDebugRunning() bool {
	return g.debugger != nil && g.debugger.IsEnabled()
}
