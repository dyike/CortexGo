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

func traderRouter(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*TradingState](ctx, func(_ context.Context, state *TradingState) error {
		defer func() {
			output = state.Goto
		}()

		state.Goto = consts.Coordinator
		if len(input.ToolCalls) > 0 && input.ToolCalls[0].Function.Name == "execute_trade" {
			argMap := map[string]interface{}{}
			_ = json.Unmarshal([]byte(input.ToolCalls[0].Function.Arguments), &argMap)

			action := fmt.Sprintf("%v", argMap["action"])
			quantity := 0
			if q, ok := argMap["quantity"].(float64); ok {
				quantity = int(q)
			}

			state.Decision = &models.TradingDecision{
				Symbol:     state.CurrentSymbol,
				Action:     action,
				Quantity:   float64(quantity),
				Price:      state.MarketData.Price,
				Date:       state.CurrentDate,
				Confidence: 0.8,
				Reason:     fmt.Sprintf("%v", argMap["reasoning"]),
			}

			if next, ok := argMap["next_agent"].(string); ok && next != "" {
				switch next {
				case "risk_manager":
					state.Goto = consts.RiskManager
				case "reporter":
					state.Goto = consts.Reporter
				default:
					state.Goto = consts.Reporter
				}
			} else {
				state.Goto = consts.Reporter
			}
		}
		return nil
	})
	return output, nil
}

func loadTraderMessages(ctx context.Context, name string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*TradingState](ctx, func(_ context.Context, state *TradingState) error {
		systemPrompt := `You are an experienced trader responsible for making trading decisions based on analysis and research.

Your responsibilities:
1. Review all previous analysis and research
2. Make concrete trading decisions (buy/sell/hold)
3. Determine appropriate position sizes
4. Execute trades with clear reasoning

Current context:
- Symbol: ` + state.CurrentSymbol + `
- Date: ` + state.CurrentDate.Format("2006-01-02") + `
- Current Price: ` + fmt.Sprintf("%.2f", state.MarketData.Price) + `

Analysis reports:`

		for _, report := range state.Reports {
			systemPrompt += fmt.Sprintf("\n- %s: %s (Recommendation: %s)",
				report.Analyst, report.Analysis, report.Rating)
		}

		systemPrompt += `

Based on all available information, make a trading decision using the execute_trade tool.
Consider:
- Technical analysis insights
- Fundamental research findings
- Risk factors
- Market conditions
- Position sizing

Provide clear reasoning for your decision.`

		output = []*schema.Message{
			schema.SystemMessage(systemPrompt),
		}

		tradingPrompt := fmt.Sprintf("Based on all the analysis, make a trading decision for %s", state.CurrentSymbol)
		output = append(output, schema.UserMessage(tradingPrompt))

		return nil
	})
	return output, err
}

func NewTraderNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()

	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadTraderMessages))
	_ = g.AddChatModelNode("agent", ChatModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(traderRouter))

	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)

	return g
}
