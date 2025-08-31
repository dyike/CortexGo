package managers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/internal/agents"
	"github.com/dyike/CortexGo/internal/config"
	"github.com/dyike/CortexGo/internal/models"
	"github.com/dyike/CortexGo/internal/processing"
)

func riskManagerRouter(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		defer func() {
			output = state.Goto
		}()

		// Mark risk phase and workflow as complete
		state.RiskPhaseComplete = true
		state.WorkflowComplete = true
		state.Goto = compose.END

		if len(input.ToolCalls) > 0 && input.ToolCalls[0].Function.Name == "submit_risk_management_decision" {
			argMap := map[string]interface{}{}
			_ = json.Unmarshal([]byte(input.ToolCalls[0].Function.Arguments), &argMap)

			if decision, ok := argMap["final_decision"].(string); ok {
				state.FinalTradeDecision = decision
			}

			if reasoning, ok := argMap["detailed_reasoning"].(string); ok {
				state.RiskDebateState.JudgeDecision = reasoning
			}

			// Store refined trading plan if provided
			if refinedPlan, ok := argMap["refined_trading_plan"].(string); ok {
				state.TraderInvestmentPlan = refinedPlan
			}

			// Store key argument summaries if provided
			if keyArgs, ok := argMap["key_arguments_summary"].(string); ok {
				// Could store this in a new field for future reference
				_ = keyArgs // For now, just acknowledge it exists
			}
		}

		// Process the final trading signal using signal processor
		if decision, err := processing.ProcessSignal(ctx, state); err == nil {
			state.Decision = decision
		}

		return nil
	})
	return output, nil
}

func loadRiskManagerMessages(ctx context.Context, name string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		// Get memory context for learning from past mistakes
		memoryContext := ""
		if len(state.PreviousDecisions) > 0 {
			memoryContext = fmt.Sprintf("\n**Learn from Past Mistakes**: Use lessons from previous decisions to address prior misjudgments. Recent decisions: %+v", state.PreviousDecisions)
		}

		// Get risk analyst debate summaries for comprehensive analysis
		riskyHistory := state.RiskDebateState.RiskyHistory
		safeHistory := state.RiskDebateState.SafeHistory
		neutralHistory := state.RiskDebateState.NeutralHistory
		fullDebateHistory := state.RiskDebateState.History

		systemPrompt := fmt.Sprintf(`You are the Risk Management Judge and Debate Facilitator, the final decision-maker who evaluates trading risk and makes ultimate investment decisions.

**Your Role**: Risk Management Judge and Final Decision Authority

**Your Responsibilities**:
1. **Evaluate Risk Debates**: Analyze arguments from three specialized risk analysts (Risky, Conservative, Neutral)
2. **Make Definitive Trading Decision**: Determine whether to Buy, Sell, or Hold - avoid defaulting to Hold unless strongly justified
3. **Integrate Multiple Perspectives**: Balance aggressive opportunities against conservative safety and neutral objectivity
4. **Provide Detailed Reasoning**: Anchor your decision in specific analyst arguments and evidence
5. **Refine Trading Strategy**: Improve the original trader's plan based on risk analyst insights

**Complete Investment Context**:
- **Company**: %s
- **Trade Date**: %s
- **Market Analysis**: %s
- **Social Sentiment**: %s  
- **News Analysis**: %s
- **Fundamentals**: %s
- **Research Decision**: %s
- **Original Trading Plan**: %s

**Risk Analyst Debate Analysis**:

**Risky Analyst's Arguments**: 
%s

**Conservative Analyst's Arguments**: 
%s

**Neutral Analyst's Arguments**: 
%s

**Complete Risk Discussion History**: 
%s
%s

**Decision Framework** - Use the submit_risk_management_decision tool with:

1. **Summarize Key Arguments**: Extract the strongest points from each risk analyst perspective
2. **Provide Your Rationale**: Support your recommendation with:
   - Direct quotes and references to specific analyst arguments
   - Explanation of how you weighed different risk perspectives
   - Addressing key concerns raised by each analyst
   - Integration of lessons from past experiences (if any)
3. **Refine the Trader's Plan**: Improve the original plan based on analyst insights:
   - Position sizing adjustments based on risk assessment
   - Enhanced entry/exit strategy with risk management
   - Stop-loss and take-profit refinements
   - Risk tolerance considerations
4. **Learn from Past Mistakes**: Use historical context to avoid repeating previous errors

**Critical Guidance**: Make a decisive Buy/Sell/Hold recommendation. Avoid defaulting to Hold unless there are compelling reasons. Your decision should balance opportunity with prudent risk management while prioritizing long-term portfolio health.`,
			state.CompanyOfInterest,
			state.TradeDate,
			state.MarketReport,
			state.SentimentReport,
			state.NewsReport,
			state.FundamentalsReport,
			state.InvestmentDebateState.JudgeDecision,
			state.TraderInvestmentPlan,
			riskyHistory,
			safeHistory,
			neutralHistory,
			fullDebateHistory,
			memoryContext)

		output = []*schema.Message{
			schema.SystemMessage(systemPrompt),
		}

		userMessage := fmt.Sprintf("As the Risk Management Judge and Debate Facilitator, make your definitive trading decision for %s. Synthesize all three risk analyst perspectives (Risky, Conservative, Neutral), provide comprehensive reasoning anchored in their specific arguments, and refine the original trading plan. Use the decision framework to structure your response.", state.CompanyOfInterest)
		output = append(output, schema.UserMessage(userMessage))

		return nil
	})
	return output, err
}

func NewRiskManagerNode[I, O any](ctx context.Context, cfg *config.Config) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()

	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadRiskManagerMessages))
	_ = g.AddChatModelNode("agent", agents.ChatModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(riskManagerRouter))

	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)

	return g
}
