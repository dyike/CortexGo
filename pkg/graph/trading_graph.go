package graph

import (
	"context"
	"fmt"
	"time"

	"github.com/dyike/CortexGo/pkg/agents"
	"github.com/dyike/CortexGo/pkg/config"
	"github.com/dyike/CortexGo/pkg/models"
)

type TradingAgentsGraph struct {
	config     *config.Config
	analysts   []agents.Agent
	researcher agents.Agent
	trader     agents.Agent
	riskMgr    agents.Agent
	debug      bool
}

func NewTradingAgentsGraph(debug bool, cfg *config.Config) *TradingAgentsGraph {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	graph := &TradingAgentsGraph{
		config: cfg,
		debug:  debug,
	}

	graph.analysts = []agents.Agent{
		agents.NewFundamentalAnalyst(cfg),
		agents.NewSentimentAnalyst(cfg),
		agents.NewTechnicalAnalyst(cfg),
		agents.NewNewsAnalyst(cfg),
	}

	graph.researcher = agents.NewResearcher(cfg)
	graph.trader = agents.NewTrader(cfg)
	graph.riskMgr = agents.NewRiskManager(cfg)

	return graph
}

func (g *TradingAgentsGraph) Propagate(symbol string, date string) (*models.AgentState, *models.TradingDecision, error) {
	ctx := context.Background()
	
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid date format: %v", err)
	}

	state := &models.AgentState{
		CurrentSymbol: symbol,
		CurrentDate:   parsedDate,
		Reports:       []models.AnalysisReport{},
		MarketData: &models.MarketData{
			Symbol:    symbol,
			Price:     125.50,
			Volume:    1000000,
			Timestamp: parsedDate,
			High:      127.80,
			Low:       123.20,
			Open:      124.00,
			Close:     125.50,
		},
		Metadata: make(map[string]interface{}),
	}

	if g.debug {
		fmt.Printf("Processing %s for date %s\n", symbol, date)
	}

	for _, analyst := range g.analysts {
		if g.debug {
			fmt.Printf("Running %s...\n", analyst.Name())
		}
		state, err = analyst.Process(ctx, state)
		if err != nil {
			return nil, nil, fmt.Errorf("analyst %s failed: %v", analyst.Name(), err)
		}
	}

	if g.debug {
		fmt.Printf("Running researcher...\n")
	}
	state, err = g.researcher.Process(ctx, state)
	if err != nil {
		return nil, nil, fmt.Errorf("researcher failed: %v", err)
	}

	if g.debug {
		fmt.Printf("Running trader...\n")
	}
	state, err = g.trader.Process(ctx, state)
	if err != nil {
		return nil, nil, fmt.Errorf("trader failed: %v", err)
	}

	if g.debug {
		fmt.Printf("Running risk manager...\n")
	}
	state, err = g.riskMgr.Process(ctx, state)
	if err != nil {
		return nil, nil, fmt.Errorf("risk manager failed: %v", err)
	}

	if g.debug {
		fmt.Printf("Trading decision completed for %s\n", symbol)
	}

	return state, state.Decision, nil
}

func (g *TradingAgentsGraph) ReflectAndRemember(positionReturns float64) error {
	if g.debug {
		fmt.Printf("Reflecting on position returns: %.2f\n", positionReturns)
	}
	
	return nil
}