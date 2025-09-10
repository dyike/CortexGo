package managers

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/config"
	"github.com/dyike/CortexGo/internal/agents"
	"github.com/dyike/CortexGo/internal/models"
	"github.com/dyike/CortexGo/internal/utils"
)

func riskManagerRouter(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		defer func() {
			output = state.Goto
		}()

		if input != nil && state.RiskDebateState != nil {
			// Update the risk debate state following the Python pattern
			riskDebateState := state.RiskDebateState
			
			// Set judge decision and final trade decision
			riskDebateState.JudgeDecision = input.Content
			state.FinalTradeDecision = input.Content
			
			// Update latest speaker to Judge
			riskDebateState.LatestSpeaker = "Judge"

			// Add the response to the state messages
			state.Messages = append(state.Messages, input)

			// Mark risk phase and workflow as complete
			state.RiskPhaseComplete = true
			state.WorkflowComplete = true
		}

		// Set next step - typically end of workflow
		state.Goto = compose.END
		
		return nil
	})
	return output, err
}

func loadRiskManagerMessages(ctx context.Context, name string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		// Extract risk debate state data
		riskDebateState := state.RiskDebateState
		history := ""
		if riskDebateState != nil {
			history = riskDebateState.History
		}

		// Get memory context for learning from past mistakes  
		pastMemoryStr := ""
		if len(state.PreviousDecisions) > 0 {
			for i, decision := range state.PreviousDecisions {
				pastMemoryStr += fmt.Sprintf("Decision %d: %+v\n\n", i+1, decision)
			}
		}

		// Construct current situation for memory context (matching Python implementation)
		currSituation := fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s",
			state.MarketReport,
			state.SocialReport,
			state.NewsReport,
			state.FundamentalsReport)
		_ = currSituation // For future memory integration

		// Load prompt from external markdown file
		systemPrompt, _ := utils.LoadPrompt("managers/risk_manager")
		
		// Create prompt template
		promptTemp := prompt.FromMessages(schema.FString,
			schema.SystemMessage("{system_message}"),
			schema.MessagesPlaceholder("user_input", true),
		)
		
		// Load prompt context
		context := map[string]any{
			"system_message":        systemPrompt,
			"trader_plan":           state.InvestmentPlan,
			"past_memory_str":       pastMemoryStr,
			"history":               history,
		}

		output, err = promptTemp.Format(ctx, context)
		return err
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
