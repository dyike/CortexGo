package eino

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/consts"
	"github.com/dyike/CortexGo/pkg/models"
)

func riskManagerRouter(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*TradingState](ctx, func(_ context.Context, state *TradingState) error {
		defer func() {
			output = state.Goto
		}()

		state.Goto = consts.Reporter
		if len(input.ToolCalls) > 0 && input.ToolCalls[0].Function.Name == "assess_risk" {
			argMap := map[string]interface{}{}
			_ = json.Unmarshal([]byte(input.ToolCalls[0].Function.Arguments), &argMap)

			riskAssessment := models.AnalysisReport{
				Analyst:    "RiskManager",
				Symbol:     state.CurrentSymbol,
				Date:       state.CurrentDate,
				Analysis:   fmt.Sprintf("%v", argMap["risk_analysis"]),
				Confidence: 0.9,
				Rating:     fmt.Sprintf("%v", argMap["risk_recommendation"]),
			}

			state.Reports = append(state.Reports, riskAssessment)

			if state.Decision != nil {
				if approved, ok := argMap["approve_trade"].(bool); ok {
					if !approved {
						state.Decision.Action = "hold"
						state.Decision.Reason = fmt.Sprintf("Trade rejected by risk management: %v", argMap["risk_analysis"])
					}
				}
			}

			if next, ok := argMap["next_agent"].(string); ok && next != "" {
				switch next {
				case "trader":
					state.Goto = consts.Trader
				case "reporter":
					state.Goto = consts.Reporter
				default:
					state.Goto = consts.Reporter
				}
			}
		}
		return nil
	})
	return output, nil
}

func loadRiskManagerMessages(ctx context.Context, name string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*TradingState](ctx, func(_ context.Context, state *TradingState) error {
		systemPrompt := `You are a risk manager responsible for assessing and managing trading risks.

Your responsibilities:
1. Evaluate market risks and exposure
2. Assess position sizing and risk/reward ratios
3. Review trading decisions for risk compliance
4. Approve or reject trades based on risk parameters
5. Recommend risk mitigation strategies

Current context:
- Symbol: ` + state.CurrentSymbol + `
- Date: ` + state.CurrentDate.Format("2006-01-02")

		if state.Decision != nil {
			systemPrompt += fmt.Sprintf(`
- Proposed Trade: %s %.0f shares at %.2f
- Reasoning: %s`,
				state.Decision.Action, state.Decision.Quantity,
				state.Decision.Price, state.Decision.Reason)
		}

		systemPrompt += `

Analysis reports:`

		for _, report := range state.Reports {
			systemPrompt += fmt.Sprintf("\n- %s: %s", report.Analyst, report.Analysis)
		}

		systemPrompt += `

Assess the risks and provide recommendations using the assess_risk tool.
Consider:
- Market volatility and liquidity risks
- Position sizing relative to portfolio
- Maximum drawdown scenarios
- Risk/reward ratios
- Correlation risks
- Economic and sector-specific risks

Approve or reject the proposed trade based on risk parameters.`

		output = []*schema.Message{
			schema.SystemMessage(systemPrompt),
		}

		riskPrompt := "Please assess the risks associated with this trading decision and provide your risk management recommendations."
		output = append(output, schema.UserMessage(riskPrompt))

		return nil
	})
	return output, err
}

func NewRiskManagerNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()

	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadRiskManagerMessages))
	_ = g.AddChatModelNode("agent", ChatModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(riskManagerRouter))

	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)

	return g
}
