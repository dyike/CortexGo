package graph

import (
	"context"

	"github.com/cloudwego/eino/compose"
	"github.com/dyike/CortexGo/consts"
	"github.com/dyike/CortexGo/internal/agents/analysts"
	"github.com/dyike/CortexGo/internal/agents/managers"
	"github.com/dyike/CortexGo/internal/agents/researchers"
	"github.com/dyike/CortexGo/internal/agents/risk_mgmt"
	"github.com/dyike/CortexGo/internal/agents/trader"
	"github.com/dyike/CortexGo/internal/config"
)

func NewTradingOrchestrator[I, O, S any](ctx context.Context, genFunc compose.GenLocalState[S], cfg *config.Config) compose.Runnable[I, O] {
	g := compose.NewGraph[I, O](
		compose.WithGenLocalState(genFunc),
	)

	// Define output maps for conditional branches only
	debateOutMap := map[string]bool{
		consts.BullResearcher:  true,
		consts.BearResearcher:  true,
		consts.ResearchManager: true,
	}

	riskOutMap := map[string]bool{
		consts.RiskyAnalyst:   true,
		consts.SafeAnalyst:    true,
		consts.NeutralAnalyst: true,
		consts.RiskJudge:      true,
	}

	// 创建分析师节点 - use new ReAct-based MarketAnalyst
	marketAnalystGraph := analysts.NewMarketAnalyst[I, O](ctx, cfg)
	socialAnalystGraph := analysts.NewSocialAnalyst[I, O](ctx, cfg)
	newsAnalystGraph := analysts.NewNewsAnalyst[I, O](ctx, cfg)
	fundamentalsAnalystGraph := analysts.NewFundamentalsAnalystNode[I, O](ctx, cfg)

	// 创建研究员节点 - use simple nodes with proper type adapters
	bullResearcherGraph := researchers.NewBullResearcherNode[I, O](ctx, cfg)
	bearResearcherGraph := researchers.NewBearResearcherNode[I, O](ctx, cfg)
	researchManagerGraph := managers.NewResearchManagerNode[I, O](ctx)

	// 创建交易员节点 - use simple nodes with proper type adapters
	traderGraph := trader.NewTraderNode[I, O](ctx)

	// 创建风险分析节点 - use simple nodes with proper type adapters
	riskyAnalystGraph := risk_mgmt.NewRiskyAnalystNode[I, O](ctx)
	neutralAnalystGraph := risk_mgmt.NewNeutralAnalystNode[I, O](ctx)
	safeAnalystGraph := risk_mgmt.NewSafeAnalystNode[I, O](ctx)
	riskManagerGraph := managers.NewRiskManagerNode[I, O](ctx)

	// 添加所有节点
	// Analyst
	_ = g.AddGraphNode(consts.MarketAnalyst, marketAnalystGraph, compose.WithNodeName(consts.MarketAnalyst))
	_ = g.AddGraphNode(consts.SocialAnalyst, socialAnalystGraph, compose.WithNodeName(consts.SocialAnalyst))
	_ = g.AddGraphNode(consts.NewsAnalyst, newsAnalystGraph, compose.WithNodeName(consts.NewsAnalyst))
	_ = g.AddGraphNode(consts.FundamentalsAnalyst, fundamentalsAnalystGraph, compose.WithNodeName(consts.FundamentalsAnalyst))
	// Research
	_ = g.AddGraphNode(consts.BullResearcher, bullResearcherGraph, compose.WithNodeName(consts.BullResearcher))
	_ = g.AddGraphNode(consts.BearResearcher, bearResearcherGraph, compose.WithNodeName(consts.BearResearcher))
	_ = g.AddGraphNode(consts.ResearchManager, researchManagerGraph, compose.WithNodeName(consts.ResearchManager))
	// Trader
	_ = g.AddGraphNode(consts.Trader, traderGraph, compose.WithNodeName(consts.Trader))
	// Risk
	_ = g.AddGraphNode(consts.RiskyAnalyst, riskyAnalystGraph, compose.WithNodeName(consts.RiskyAnalyst))
	_ = g.AddGraphNode(consts.SafeAnalyst, safeAnalystGraph, compose.WithNodeName(consts.SafeAnalyst))
	_ = g.AddGraphNode(consts.NeutralAnalyst, neutralAnalystGraph, compose.WithNodeName(consts.NeutralAnalyst))
	_ = g.AddGraphNode(consts.RiskJudge, riskManagerGraph, compose.WithNodeName(consts.RiskJudge))

	// Sequential edges for analysis phase (linear flow)
	_ = g.AddEdge(compose.START, consts.MarketAnalyst)
	_ = g.AddEdge(consts.MarketAnalyst, consts.SocialAnalyst)
	_ = g.AddEdge(consts.SocialAnalyst, consts.NewsAnalyst)
	_ = g.AddEdge(consts.NewsAnalyst, consts.FundamentalsAnalyst)
	_ = g.AddEdge(consts.FundamentalsAnalyst, consts.BullResearcher)

	// // Conditional branches for debate phase (bull/bear cycle)
	_ = g.AddBranch(consts.BullResearcher, compose.NewGraphBranch(ShouldContinueDebate, debateOutMap))
	_ = g.AddBranch(consts.BearResearcher, compose.NewGraphBranch(ShouldContinueDebate, debateOutMap))

	// // Sequential edge to trading phase
	_ = g.AddEdge(consts.ResearchManager, consts.Trader)
	_ = g.AddEdge(consts.Trader, consts.RiskyAnalyst)

	// // Conditional branches for risk phase (three-way cycle)
	_ = g.AddBranch(consts.RiskyAnalyst, compose.NewGraphBranch(ShouldContinueRiskAnalysis, riskOutMap))
	_ = g.AddBranch(consts.SafeAnalyst, compose.NewGraphBranch(ShouldContinueRiskAnalysis, riskOutMap))
	_ = g.AddBranch(consts.NeutralAnalyst, compose.NewGraphBranch(ShouldContinueRiskAnalysis, riskOutMap))

	// Final edge to end
	_ = g.AddEdge(consts.RiskJudge, compose.END)

	r, err := g.Compile(ctx,
		compose.WithGraphName("CortexGo-TradingAgents"),
		compose.WithNodeTriggerMode(compose.AnyPredecessor),
	)
	if err != nil {
		panic(err)
	}
	return r
}
