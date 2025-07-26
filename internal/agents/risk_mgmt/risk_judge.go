package risk_mgmt

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/internal/agents"
	"github.com/dyike/CortexGo/internal/processing"
	"github.com/dyike/CortexGo/internal/models"
)

func riskJudgeRouter(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		defer func() {
			output = state.Goto
		}()

		// Mark risk phase and workflow as complete
		state.RiskPhaseComplete = true
		state.WorkflowComplete = true
		state.Goto = compose.END
		
		if len(input.ToolCalls) > 0 && input.ToolCalls[0].Function.Name == "submit_final_decision" {
			argMap := map[string]interface{}{}
			_ = json.Unmarshal([]byte(input.ToolCalls[0].Function.Arguments), &argMap)

			if decision, ok := argMap["final_decision"].(string); ok {
				state.FinalTradeDecision = decision
			}
		}

		// Process the final trading signal
		if decision, err := processing.ProcessSignal(ctx, state); err == nil {
			state.Decision = decision
		}
		
		return nil
	})
	return output, nil
}

func loadRiskJudgeMessages(ctx context.Context, name string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		systemPrompt := `You are the final risk judge who makes ultimate trading decisions based on comprehensive analysis and risk assessment.

Your responsibilities:
1. Review all analysis reports, debate outcomes, and risk discussions
2. Make the final trading decision with risk management
3. Provide clear rationale for the decision
4. Ensure the decision aligns with risk tolerance

Complete Analysis Summary:
- Market Analysis: ` + state.MarketReport + `
- Social Analysis: ` + state.SentimentReport + `
- News Analysis: ` + state.NewsReport + `
- Fundamentals Analysis: ` + state.FundamentalsReport + `
- Investment Debate Decision: ` + state.InvestmentDebateState.JudgeDecision + `
- Trading Plan: ` + state.TraderInvestmentPlan + `
- Risk Discussion Summary: ` + state.RiskDebateState.History + `

Risk Analysis from Specialists:
- Risky Perspective: ` + state.RiskDebateState.RiskyHistory + `
- Safe Perspective: ` + state.RiskDebateState.SafeHistory + `
- Neutral Perspective: ` + state.RiskDebateState.NeutralHistory + `

When you complete your analysis, use the submit_final_decision tool to provide:
- Final trading decision (BUY/SELL/HOLD)
- Risk-adjusted position sizing
- Entry/exit strategy with risk management
- Clear rationale based on all analysis

Make a decision that balances opportunity with risk management.`

		output = []*schema.Message{
			schema.SystemMessage(systemPrompt),
		}

		userMessage := fmt.Sprintf("Make the final trading decision for %s considering all analysis and risk factors", state.CompanyOfInterest)
		output = append(output, schema.UserMessage(userMessage))

		return nil
	})
	return output, err
}

func NewRiskJudgeNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()

	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadRiskJudgeMessages))
	_ = g.AddChatModelNode("agent", agents.ChatModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(riskJudgeRouter))

	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)

	return g
}

