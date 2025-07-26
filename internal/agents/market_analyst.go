package agents

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/consts"
)

func marketAnalystRouter(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*TradingState](ctx, func(_ context.Context, state *TradingState) error {
		defer func() {
			output = state.Goto
		}()

		state.Goto = consts.SocialMediaAnalyst
		if len(input.ToolCalls) > 0 && input.ToolCalls[0].Function.Name == "submit_market_analysis" {
			argMap := map[string]interface{}{}
			_ = json.Unmarshal([]byte(input.ToolCalls[0].Function.Arguments), &argMap)

			if analysis, ok := argMap["analysis"].(string); ok {
				state.MarketReport = analysis
			}

			if next, ok := argMap["next_step"].(string); ok && next != "" {
				switch next {
				case "social":
					state.Goto = consts.SocialMediaAnalyst
				case "fundamentals":
					state.Goto = consts.FundamentalsAnalyst
				case "news":
					state.Goto = consts.NewsAnalyst
				case "bull_researcher":
					state.Goto = consts.BullResearcher
				default:
					state.Goto = consts.SocialMediaAnalyst
				}
			}
		}
		return nil
	})
	return output, nil
}

func loadMarketAnalystMessages(ctx context.Context, name string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*TradingState](ctx, func(_ context.Context, state *TradingState) error {
		systemPrompt := `You are a senior market analyst specializing in technical analysis and market data interpretation.

Your responsibilities:
1. Analyze market data, price trends, and volume patterns
2. Provide technical analysis insights using indicators like RSI, MACD, moving averages
3. Evaluate market sentiment and trading signals  
4. Determine next analysis step in the workflow

Current context:
- Company: ` + state.CompanyOfInterest + `
- Trade Date: ` + state.TradeDate + `
- Market Data: Price movements, volume analysis, technical indicators

When you complete your analysis, use the submit_market_analysis tool to provide:
- Comprehensive market analysis including technical indicators
- Trading signals and market sentiment assessment
- Next analysis step (social/fundamentals/news/bull_researcher)

Focus on price action, support/resistance levels, momentum indicators, and volume analysis.`

		output = []*schema.Message{
			schema.SystemMessage(systemPrompt),
		}

		if state.MarketData != nil {
			marketContext := fmt.Sprintf("Current market data for %s: Price: %.2f, Volume: %d, High: %.2f, Low: %.2f",
				state.MarketData.Symbol, state.MarketData.Price, state.MarketData.Volume,
				state.MarketData.High, state.MarketData.Low)
			output = append(output, schema.UserMessage(marketContext))
		}

		return nil
	})
	return output, err
}

func NewMarketAnalystNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()

	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadMarketAnalystMessages))
	_ = g.AddChatModelNode("agent", ChatModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(marketAnalystRouter))

	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)

	return g
}
