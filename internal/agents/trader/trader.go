package trader

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/config"
	"github.com/dyike/CortexGo/consts"
	"github.com/dyike/CortexGo/internal/agents"
	"github.com/dyike/CortexGo/internal/models"
)

func traderRouter(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		defer func() {
			output = state.Goto
		}()

		// Mark trading phase as complete and transition to risk phase
		state.TradingPhaseComplete = true
		state.Phase = "risk"
		state.Goto = consts.RiskyAnalyst

		if len(input.ToolCalls) > 0 && input.ToolCalls[0].Function.Name == "submit_trading_plan" {
			argMap := map[string]interface{}{}
			_ = json.Unmarshal([]byte(input.ToolCalls[0].Function.Arguments), &argMap)

			if plan, ok := argMap["trading_plan"].(string); ok {
				state.TraderInvestmentPlan = plan
			}
		}
		return nil
	})
	return output, nil
}

func loadTraderMessages(ctx context.Context, name string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		systemPrompt := `You are a professional trader responsible for creating executable trading plans based on research and analysis.

Your responsibilities:
1. Review all analysis reports and research manager decision
2. Create specific trading plans with entry/exit points
3. Determine position sizing and risk parameters
4. Prepare trading plan for risk assessment

Current context:
- Company: ` + state.CompanyOfInterest + `
- Trade Date: ` + state.TradeDate + `
- Market Analysis: ` + state.MarketReport + `
- Social Analysis: ` + state.SocialReport + `
- News Analysis: ` + state.NewsReport + `
- Fundamentals Analysis: ` + state.FundamentalsReport + `
- Research Decision: ` + state.InvestmentDebateState.JudgeDecision + `

When you complete your analysis, use the submit_trading_plan tool to provide:
- Specific trading plan with entry/exit points
- Position sizing and risk parameters
- Expected return and risk assessment
- Trading execution strategy

Focus on creating actionable trading plans that can be executed in real markets.`

		output = []*schema.Message{
			schema.SystemMessage(systemPrompt),
		}

		userMessage := fmt.Sprintf("Create a trading plan for %s based on all available analysis", state.CompanyOfInterest)
		output = append(output, schema.UserMessage(userMessage))

		return nil
	})
	return output, err
}

func NewTraderNode[I, O any](ctx context.Context, cfg *config.Config) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()

	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadTraderMessages))
	_ = g.AddChatModelNode("agent", agents.ChatModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(traderRouter))

	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)

	return g
}
