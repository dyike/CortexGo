package managers

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/config"
	"github.com/dyike/CortexGo/consts"
	"github.com/dyike/CortexGo/internal/agents"
	"github.com/dyike/CortexGo/internal/models"
	"github.com/dyike/CortexGo/internal/utils"
)

func researchManagerRouter(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		defer func() {
			output = state.Goto
		}()

		if input != nil && state.InvestmentDebateState != nil {
			// Update the investment debate state following the Python pattern
			investmentDebateState := state.InvestmentDebateState

			// Set judge decision and investment plan
			investmentDebateState.JudgeDecision = input.Content
			investmentDebateState.CurrentResponse = input.Content
			state.InvestmentPlan = input.Content

			// Add the response to the state messages
			state.Messages = append(state.Messages, input)

			filePath := fmt.Sprintf("results/%s/%s", state.CompanyOfInterest, state.TradeDate)
			fileName := "research_manager_report.md"
			if err := utils.WriteMarkdown(filePath, fileName, input.Content); err != nil {
				log.Printf("Failed to write research manager report: %v", err)
			}

			// Mark debate phase as complete and transition to trading phase
			state.DebatePhaseComplete = true
			state.Phase = "trading"
		}

		// Set next step to trader
		state.Goto = consts.Trader

		return nil
	})
	return output, err
}

func loadResearchManagerMessages(ctx context.Context, name string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		// Extract investment debate state data
		investmentDebateState := state.InvestmentDebateState
		history := ""
		if investmentDebateState != nil {
			history = investmentDebateState.History
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
		systemPrompt, _ := utils.LoadPrompt("managers/research_manager")

		// Create prompt template
		promptTemp := prompt.FromMessages(schema.FString,
			schema.SystemMessage("{system_message}"),
			schema.MessagesPlaceholder("user_input", true),
		)

		// Load prompt context
		context := map[string]any{
			"system_message":  systemPrompt,
			"past_memory_str": pastMemoryStr,
			"history":         history,
		}

		output, err = promptTemp.Format(ctx, context)
		return err
	})
	return output, err
}

func NewResearchManagerNode[I, O any](ctx context.Context, cfg *config.Config) *compose.Graph[I, O] {
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
