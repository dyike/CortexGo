package agents

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/consts"
	"github.com/dyike/CortexGo/internal/models"
)

func researcherRouter(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*TradingState](ctx, func(_ context.Context, state *TradingState) error {
		defer func() {
			output = state.Goto
		}()

		state.Goto = consts.Coordinator
		if len(input.ToolCalls) > 0 && input.ToolCalls[0].Function.Name == "submit_research" {
			argMap := map[string]interface{}{}
			_ = json.Unmarshal([]byte(input.ToolCalls[0].Function.Arguments), &argMap)

			report := models.AnalysisReport{
				Analyst:    "MarketResearcher",
				Symbol:     state.CurrentSymbol,
				Date:       state.CurrentDate,
				Analysis:   fmt.Sprintf("%v", argMap["research_findings"]),
				Confidence: 0.85,
				Rating:     fmt.Sprintf("%v", argMap["market_outlook"]),
			}

			state.Reports = append(state.Reports, report)

			if next, ok := argMap["next_agent"].(string); ok && next != "" {
				switch next {
				case "trader":
					state.Goto = consts.Trader
				case "risk_manager":
					state.Goto = consts.RiskManager
				case "reporter":
					state.Goto = consts.Reporter
				case "analyst":
					state.Goto = consts.Analyst
				default:
					state.Goto = consts.Coordinator
				}
			}
		}
		return nil
	})
	return output, nil
}

func loadResearcherMessages(ctx context.Context, name string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*TradingState](ctx, func(_ context.Context, state *TradingState) error {
		systemPrompt := `You are a market researcher specializing in fundamental analysis and market intelligence.

Your responsibilities:
1. Conduct fundamental analysis of the company/asset
2. Research market trends and economic factors
3. Analyze competitive landscape
4. Provide market outlook and insights

Current context:
- Symbol: ` + state.CurrentSymbol + `
- Date: ` + state.CurrentDate.Format("2006-01-02") + `

Previous analysis reports:`

		for _, report := range state.Reports {
			systemPrompt += fmt.Sprintf("\n- %s: %s", report.Analyst, report.Analysis)
		}

		systemPrompt += `

When you complete your research, use the submit_research tool to provide:
- Detailed fundamental analysis
- Market outlook and trends
- Economic factors affecting the asset
- Next agent to activate

Focus on company financials, industry trends, economic indicators, and market sentiment.`

		output = []*schema.Message{
			schema.SystemMessage(systemPrompt),
		}

		researchPrompt := fmt.Sprintf("Please conduct comprehensive research on %s for trading decision making.", state.CurrentSymbol)
		output = append(output, schema.UserMessage(researchPrompt))

		return nil
	})
	return output, err
}

func NewResearcherNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()

	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadResearcherMessages))
	_ = g.AddChatModelNode("agent", ChatModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(researcherRouter))

	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)

	return g
}
