package graph

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/compose"
	"github.com/dyike/CortexGo/config"
	"github.com/dyike/CortexGo/internal/models"
)

type TradingAgentsGraph struct {
	config       *config.Config
	orchestrator compose.Runnable[*models.TradingState, *models.TradingState]
	debug        bool
	emit         func(event string, data *models.ChatResp)
}

func NewTradingAgentsGraph(debug bool, cfg *config.Config) *TradingAgentsGraph {
	return NewTradingAgentsGraphWithEmitter(cfg, nil)
}

func NewTradingAgentsGraphWithEmitter(cfg *config.Config, emit func(string, *models.ChatResp)) *TradingAgentsGraph {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	emitter := emit
	if emitter == nil {
		emitter = func(_ string, data *models.ChatResp) {
			if data == nil {
				return
			}
			txt := data.Content
			if strings.TrimSpace(txt) == "" {
				return
			}
			fmt.Print(txt)
		}
	}

	ctx := context.Background()
	// No need to initialize agents infrastructure when using DeepSeek successfully
	orchestrator := NewTradingOrchestrator[*models.TradingState, *models.TradingState, *models.TradingState](
		ctx,
		func(ctx context.Context) *models.TradingState {
			return &models.TradingState{Config: cfg}
		},
		cfg,
	)

	return &TradingAgentsGraph{
		config:       cfg,
		orchestrator: orchestrator,
		debug:        cfg.EinoDebugEnabled,
		emit:         emitter,
	}
}

func (g *TradingAgentsGraph) Propagate(symbol string, date string) (*models.TradingState, error) {
	ctx := context.Background()

	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %v", err)
	}

	userPrompt := fmt.Sprintf("Analyze trading opportunities for %s on %s", symbol, date)
	state := models.NewTradingState(symbol, parsedDate, userPrompt, g.config)
	if g.debug {
		fmt.Printf("Processing %s for date %s using eino orchestrator\n", symbol, date)
	}

	_, err = g.orchestrator.Stream(ctx, state,
		compose.WithCallbacks(&LoggerCallback{
			Emit: g.emit,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("orchestrator failed: %v", err)
	}

	return state, nil
}
