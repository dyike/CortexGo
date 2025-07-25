package eino

import (
	"context"

	"github.com/cloudwego/eino/compose"
	"github.com/dyike/CortexGo/consts"
)

func agentHandOff(ctx context.Context, input string) (next string, err error) {
	_ = compose.ProcessState[*TradingState](ctx, func(_ context.Context, state *TradingState) error {
		next = state.Goto
		return nil
	})
	return next, nil
}

func NewTradingOrchestrator[I, O, S any](ctx context.Context, genFunc compose.GenLocalState[S]) compose.Runnable[I, O] {
	g := compose.NewGraph[I, O](
		compose.WithGenLocalState(genFunc),
	)

	outMap := map[string]bool{
		consts.Coordinator:  true,
		consts.Analyst:      true,
		consts.Researcher:   true,
		consts.Trader:       true,
		consts.RiskManager:  true,
		consts.Reporter:     true,
		compose.END:         true,
	}

	coordinatorGraph := NewCoordinatorNode[I, O](ctx)
	analystGraph := NewAnalystNode[I, O](ctx)
	researcherGraph := NewResearcherNode[I, O](ctx)
	traderGraph := NewTraderNode[I, O](ctx)
	riskManagerGraph := NewRiskManagerNode[I, O](ctx)
	reporterGraph := NewReporterNode[I, O](ctx)

	_ = g.AddGraphNode(consts.Coordinator, coordinatorGraph, compose.WithNodeName(consts.Coordinator))
	_ = g.AddGraphNode(consts.Analyst, analystGraph, compose.WithNodeName(consts.Analyst))
	_ = g.AddGraphNode(consts.Researcher, researcherGraph, compose.WithNodeName(consts.Researcher))
	_ = g.AddGraphNode(consts.Trader, traderGraph, compose.WithNodeName(consts.Trader))
	_ = g.AddGraphNode(consts.RiskManager, riskManagerGraph, compose.WithNodeName(consts.RiskManager))
	_ = g.AddGraphNode(consts.Reporter, reporterGraph, compose.WithNodeName(consts.Reporter))

	_ = g.AddBranch(consts.Coordinator, compose.NewGraphBranch(agentHandOff, outMap))
	_ = g.AddBranch(consts.Analyst, compose.NewGraphBranch(agentHandOff, outMap))
	_ = g.AddBranch(consts.Researcher, compose.NewGraphBranch(agentHandOff, outMap))
	_ = g.AddBranch(consts.Trader, compose.NewGraphBranch(agentHandOff, outMap))
	_ = g.AddBranch(consts.RiskManager, compose.NewGraphBranch(agentHandOff, outMap))
	_ = g.AddBranch(consts.Reporter, compose.NewGraphBranch(agentHandOff, outMap))

	_ = g.AddEdge(compose.START, consts.Coordinator)

	r, err := g.Compile(ctx,
		compose.WithGraphName("CortexGo-TradingSystem"),
		compose.WithNodeTriggerMode(compose.AnyPredecessor),
	)
	if err != nil {
		panic(err)
	}
	return r
}