package analysts

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/consts"
	"github.com/dyike/CortexGo/internal/agents"
	"github.com/dyike/CortexGo/internal/models"
)

func fundamentalsAnalystRouter(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		defer func() {
			output = state.Goto
		}()

		// Mark fundamentals analyst as complete and transition to debate phase
		state.FundamentalsAnalystComplete = true
		state.AnalysisPhaseComplete = true
		state.Phase = "debate"
		state.Goto = consts.BullResearcher
		
		if len(input.ToolCalls) > 0 && input.ToolCalls[0].Function.Name == "submit_fundamentals_analysis" {
			argMap := map[string]interface{}{}
			_ = json.Unmarshal([]byte(input.ToolCalls[0].Function.Arguments), &argMap)

			if analysis, ok := argMap["analysis"].(string); ok {
				state.FundamentalsReport = analysis
			}
		}
		return nil
	})
	return output, nil
}

func loadFundamentalsAnalystMessages(ctx context.Context, name string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		systemPrompt := `You are a fundamental analyst specializing in company financial analysis and valuation.

Your responsibilities:
1. Analyze financial statements, balance sheets, and income statements
2. Evaluate key financial ratios like P/E, ROE, debt-to-equity
3. Assess company fundamentals and intrinsic value
4. Determine next analysis step in the workflow

Current context:
- Company: ` + state.CompanyOfInterest + `
- Trade Date: ` + state.TradeDate + `
- Previous Market Analysis: ` + state.MarketReport + `
- Previous Social Analysis: ` + state.SentimentReport + `
- Previous News Analysis: ` + state.NewsReport + `

When you complete your analysis, use the submit_fundamentals_analysis tool to provide:
- Fundamental analysis including financial ratios and metrics
- Company valuation assessment and financial health evaluation
- Long-term growth prospects and competitive position analysis
- Next analysis step (bull_researcher)

Focus on financial strength, valuation metrics, and long-term investment potential.`

		output = []*schema.Message{
			schema.SystemMessage(systemPrompt),
		}

		userMessage := fmt.Sprintf("Analyze fundamental metrics and financial health for %s on %s",
			state.CompanyOfInterest, state.TradeDate)
		output = append(output, schema.UserMessage(userMessage))

		return nil
	})
	return output, err
}

func NewFundamentalsAnalystNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()

	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadFundamentalsAnalystMessages))
	_ = g.AddChatModelNode("agent", agents.ChatModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(fundamentalsAnalystRouter))

	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)

	return g
}
