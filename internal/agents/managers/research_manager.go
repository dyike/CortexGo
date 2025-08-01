package managers

import (
	"context"
	"encoding/json"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/consts"
	"github.com/dyike/CortexGo/internal/agents"
	"github.com/dyike/CortexGo/internal/models"
)

func researchManagerRouter(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		defer func() {
			output = state.Goto
		}()

		// Mark debate phase as complete and transition to trading phase
		state.DebatePhaseComplete = true
		state.Phase = "trading"
		state.Goto = consts.Trader

		if len(input.ToolCalls) > 0 && input.ToolCalls[0].Function.Name == "submit_research_decision" {
			argMap := map[string]interface{}{}
			_ = json.Unmarshal([]byte(input.ToolCalls[0].Function.Arguments), &argMap)

			if decision, ok := argMap["decision"].(string); ok {
				state.InvestmentDebateState.JudgeDecision = decision
			}
		}
		return nil
	})
	return output, nil
}

func loadResearchManagerMessages(ctx context.Context, name string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		systemPrompt := `You are a senior research manager who makes final investment decisions based on debate between bull and bear researchers.

Your responsibilities:
1. Review arguments from both bull and bear researchers
2. Weigh the strength of evidence on both sides
3. Make a final investment decision with clear rationale
4. Provide structured decision for the trader to execute

Debate Summary:
Bull Arguments: ` + state.InvestmentDebateState.BullHistory + `
Bear Arguments: ` + state.InvestmentDebateState.BearHistory + `

When you complete your analysis, use the submit_research_decision tool to provide your final investment decision.`

		output = []*schema.Message{
			schema.SystemMessage(systemPrompt),
			schema.UserMessage("Based on the debate between bull and bear researchers, make your final investment decision."),
		}

		return nil
	})
	return output, err
}

func NewResearchManagerNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()

	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadResearchManagerMessages))
	_ = g.AddChatModelNode("agent", agents.ChatModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(researchManagerRouter))

	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)

	return g
}
