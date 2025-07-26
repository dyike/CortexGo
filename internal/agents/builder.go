package agents

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
		consts.MarketAnalyst:       true,
		consts.SocialMediaAnalyst:  true,
		consts.NewsAnalyst:         true,
		consts.FundamentalsAnalyst: true,
		consts.BullResearcher:      true,
		consts.BearResearcher:      true,
		consts.ResearchManager:     true,
		consts.Trader:              true,
		consts.RiskyAnalyst:        true,
		consts.SafeAnalyst:         true,
		consts.NeutralAnalyst:      true,
		consts.RiskJudge:           true,
		compose.END:                true,
	}

	// 创建分析师节点
	marketAnalystGraph := NewMarketAnalystNode[I, O](ctx)
	socialAnalystGraph := NewSocialMediaAnalystNode[I, O](ctx)
	newsAnalystGraph := NewNewsAnalystNode[I, O](ctx)
	fundamentalsAnalystGraph := NewFundamentalsAnalystNode[I, O](ctx)

	// 创建研究员节点
	bullResearcherGraph := NewBullResearcherNode[I, O](ctx)
	bearResearcherGraph := NewBearResearcherNode[I, O](ctx)
	researchManagerGraph := NewResearchManagerNode[I, O](ctx)

	// 创建交易员节点
	traderGraph := NewTraderNode[I, O](ctx)

	// 创建风险分析节点
	riskyAnalystGraph := NewRiskyAnalystNode[I, O](ctx)
	safeAnalystGraph := NewSafeAnalystNode[I, O](ctx)
	neutralAnalystGraph := NewNeutralAnalystNode[I, O](ctx)
	riskJudgeGraph := NewRiskJudgeNode[I, O](ctx)

	// 添加所有节点
	_ = g.AddGraphNode(consts.MarketAnalyst, marketAnalystGraph, compose.WithNodeName(consts.MarketAnalyst))
	_ = g.AddGraphNode(consts.SocialMediaAnalyst, socialAnalystGraph, compose.WithNodeName(consts.SocialMediaAnalyst))
	_ = g.AddGraphNode(consts.NewsAnalyst, newsAnalystGraph, compose.WithNodeName(consts.NewsAnalyst))
	_ = g.AddGraphNode(consts.FundamentalsAnalyst, fundamentalsAnalystGraph, compose.WithNodeName(consts.FundamentalsAnalyst))
	_ = g.AddGraphNode(consts.BullResearcher, bullResearcherGraph, compose.WithNodeName(consts.BullResearcher))
	_ = g.AddGraphNode(consts.BearResearcher, bearResearcherGraph, compose.WithNodeName(consts.BearResearcher))
	_ = g.AddGraphNode(consts.ResearchManager, researchManagerGraph, compose.WithNodeName(consts.ResearchManager))
	_ = g.AddGraphNode(consts.Trader, traderGraph, compose.WithNodeName(consts.Trader))
	_ = g.AddGraphNode(consts.RiskyAnalyst, riskyAnalystGraph, compose.WithNodeName(consts.RiskyAnalyst))
	_ = g.AddGraphNode(consts.SafeAnalyst, safeAnalystGraph, compose.WithNodeName(consts.SafeAnalyst))
	_ = g.AddGraphNode(consts.NeutralAnalyst, neutralAnalystGraph, compose.WithNodeName(consts.NeutralAnalyst))
	_ = g.AddGraphNode(consts.RiskJudge, riskJudgeGraph, compose.WithNodeName(consts.RiskJudge))

	// 添加分支路由
	_ = g.AddBranch(consts.MarketAnalyst, compose.NewGraphBranch(agentHandOff, outMap))
	_ = g.AddBranch(consts.SocialMediaAnalyst, compose.NewGraphBranch(agentHandOff, outMap))
	_ = g.AddBranch(consts.NewsAnalyst, compose.NewGraphBranch(agentHandOff, outMap))
	_ = g.AddBranch(consts.FundamentalsAnalyst, compose.NewGraphBranch(agentHandOff, outMap))
	_ = g.AddBranch(consts.BullResearcher, compose.NewGraphBranch(agentHandOff, outMap))
	_ = g.AddBranch(consts.BearResearcher, compose.NewGraphBranch(agentHandOff, outMap))
	_ = g.AddBranch(consts.ResearchManager, compose.NewGraphBranch(agentHandOff, outMap))
	_ = g.AddBranch(consts.Trader, compose.NewGraphBranch(agentHandOff, outMap))
	_ = g.AddBranch(consts.RiskyAnalyst, compose.NewGraphBranch(agentHandOff, outMap))
	_ = g.AddBranch(consts.SafeAnalyst, compose.NewGraphBranch(agentHandOff, outMap))
	_ = g.AddBranch(consts.NeutralAnalyst, compose.NewGraphBranch(agentHandOff, outMap))
	_ = g.AddBranch(consts.RiskJudge, compose.NewGraphBranch(agentHandOff, outMap))

	// 设置开始节点为市场分析师
	_ = g.AddEdge(compose.START, consts.MarketAnalyst)

	r, err := g.Compile(ctx,
		compose.WithGraphName("CortexGo-TradingAgents"),
		compose.WithNodeTriggerMode(compose.AnyPredecessor),
	)
	if err != nil {
		panic(err)
	}
	return r
}
