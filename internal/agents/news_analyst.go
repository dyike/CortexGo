package agents

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/consts"
)

func newsAnalystRouter(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*TradingState](ctx, func(_ context.Context, state *TradingState) error {
		defer func() {
			output = state.Goto
		}()

		state.Goto = consts.FundamentalsAnalyst
		if len(input.ToolCalls) > 0 && input.ToolCalls[0].Function.Name == "submit_news_analysis" {
			argMap := map[string]interface{}{}
			_ = json.Unmarshal([]byte(input.ToolCalls[0].Function.Arguments), &argMap)

			if analysis, ok := argMap["analysis"].(string); ok {
				state.NewsReport = analysis
			}

			if next, ok := argMap["next_step"].(string); ok && next != "" {
				switch next {
				case "fundamentals":
					state.Goto = consts.FundamentalsAnalyst
				case "bull_researcher":
					state.Goto = consts.BullResearcher
				default:
					state.Goto = consts.FundamentalsAnalyst
				}
			}
		}
		return nil
	})
	return output, nil
}

func loadNewsAnalystMessages(ctx context.Context, name string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*TradingState](ctx, func(_ context.Context, state *TradingState) error {
		systemPrompt := `You are a financial news analyst specializing in news impact analysis on stock movements.

Your responsibilities:
1. Analyze recent news, earnings reports, and corporate announcements
2. Evaluate news sentiment and potential market impact
3. Assess how news events may influence stock price movements
4. Determine next analysis step in the workflow

Current context:
- Company: ` + state.CompanyOfInterest + `
- Trade Date: ` + state.TradeDate + `
- Previous Market Analysis: ` + state.MarketReport + `
- Previous Social Analysis: ` + state.SentimentReport + `

When you complete your analysis, use the submit_news_analysis tool to provide:
- News analysis including recent events and announcements
- News sentiment assessment and market impact evaluation
- Corporate developments and earnings impact analysis
- Next analysis step (fundamentals/bull_researcher)

Focus on news relevance, sentiment impact, and correlation with stock movements.`

		output = []*schema.Message{
			schema.SystemMessage(systemPrompt),
		}

		userMessage := fmt.Sprintf("Analyze recent news and corporate announcements for %s on %s",
			state.CompanyOfInterest, state.TradeDate)
		output = append(output, schema.UserMessage(userMessage))

		return nil
	})
	return output, err
}

func NewNewsAnalystNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()

	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadNewsAnalystMessages))
	_ = g.AddChatModelNode("agent", ChatModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(newsAnalystRouter))

	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)

	return g
}
