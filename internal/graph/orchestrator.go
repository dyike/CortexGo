package graph

import (
	"context"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/consts"
	"github.com/dyike/CortexGo/internal/agents"
	"github.com/dyike/CortexGo/internal/models"
)

func agentHandOff(ctx context.Context, input *models.TradingState) (next string, err error) {
	return ConditionalAgentHandOff(ctx, input)
}

func createTypedAgentNode[I, O any](ctx context.Context, role string) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()
	
	// Create a loader that accepts the correct input type but ignores it
	typedLoader := func(ctx context.Context, input I, opts ...any) ([]*schema.Message, error) {
		return agents.SimpleLoader("You are a " + role + ".")(ctx, "", opts...)
	}
	
	// Create a router that accepts the correct message type 
	typedRouter := func(ctx context.Context, input *schema.Message, opts ...any) (O, error) {
		nextNode, err := agents.SimpleRouter(consts.SocialMediaAnalyst)(ctx, input, opts...)
		if err != nil {
			var zero O
			return zero, err
		}
		// Convert string to O type - this is a type assertion that may fail
		if result, ok := any(nextNode).(O); ok {
			return result, nil
		}
		var zero O
		return zero, nil
	}
	
	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(typedLoader))
	_ = g.AddChatModelNode("agent", agents.ChatModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(typedRouter))
	
	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)
	
	return g
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

	// 创建分析师节点 - use simple nodes with proper type adapters
	marketAnalystGraph := createTypedAgentNode[I, O](ctx, "market analyst")
	socialAnalystGraph := createTypedAgentNode[I, O](ctx, "social media analyst")
	newsAnalystGraph := createTypedAgentNode[I, O](ctx, "news analyst")
	fundamentalsAnalystGraph := createTypedAgentNode[I, O](ctx, "fundamentals analyst")

	// 创建研究员节点 - use simple nodes with proper type adapters
	bullResearcherGraph := createTypedAgentNode[I, O](ctx, "bull researcher")
	bearResearcherGraph := createTypedAgentNode[I, O](ctx, "bear researcher")
	researchManagerGraph := createTypedAgentNode[I, O](ctx, "research manager")

	// 创建交易员节点 - use simple nodes with proper type adapters
	traderGraph := createTypedAgentNode[I, O](ctx, "trader")

	// 创建风险分析节点 - use simple nodes with proper type adapters
	riskyAnalystGraph := createTypedAgentNode[I, O](ctx, "risky analyst")
	safeAnalystGraph := createTypedAgentNode[I, O](ctx, "safe analyst")
	neutralAnalystGraph := createTypedAgentNode[I, O](ctx, "neutral analyst")
	riskJudgeGraph := createTypedAgentNode[I, O](ctx, "risk judge")

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

