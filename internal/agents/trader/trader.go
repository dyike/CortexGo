package trader

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/config"
	"github.com/dyike/CortexGo/consts"
	"github.com/dyike/CortexGo/internal/agents"
	"github.com/dyike/CortexGo/internal/utils"
	"github.com/dyike/CortexGo/models"
)

func traderRouter(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		defer func() {
			output = state.Goto
		}()

		if input != nil {
			// Set trader investment plan following the Python pattern
			state.TraderInvestmentPlan = input.Content

			// Add the response to the state messages
			state.Messages = append(state.Messages, input)

			filePath := fmt.Sprintf("results/%s/%s", state.CompanyOfInterest, state.TradeDate)
			fileName := "trader_report.md"
			if err := utils.WriteMarkdown(filePath, fileName, input.Content); err != nil {
				log.Printf("Failed to write trader report: %v", err)
			}

			// Mark trading phase as complete and transition to risk phase
			state.TradingPhaseComplete = true
			state.Phase = "risk"
		}

		// Set next step to risky analyst to start risk debate
		state.Goto = consts.RiskyAnalyst

		return nil
	})
	return output, err
}

func loadTraderMessages(ctx context.Context, name string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		// Get memory context for learning from past mistakes
		pastMemoryStr := ""
		if len(state.PreviousDecisions) > 0 {
			for i, decision := range state.PreviousDecisions {
				pastMemoryStr += fmt.Sprintf("Decision %d: %+v\n\n", i+1, decision)
			}
		} else {
			pastMemoryStr = "No past memories found."
		}

		// TODO
		currSituation := fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s",
			state.MarketReport,
			state.SocialReport,
			state.NewsReport,
			state.FundamentalsReport)
		_ = currSituation // For future memory integration

		// Load prompt from external markdown file
		systemPrompt, _ := utils.LoadPrompt("trader/trader")

		// Create system prompt with past memory context using string replacement
		systemPromptWithContext := strings.ReplaceAll(systemPrompt, "{past_memory_str}", pastMemoryStr)

		// Create user context message matching Python implementation
		userContextMessage := fmt.Sprintf(`Based on a comprehensive analysis by a team of analysts, here is an investment plan tailored for %s. This plan incorporates insights from current technical market trends, macroeconomic indicators, and social media sentiment. Use this plan as a foundation for evaluating your next trading decision.\n\nProposed Investment Plan: %s\n\nLeverage these insights to make an informed and strategic decision.

The output content should be in Chinese.
`,
			state.CompanyOfInterest,
			state.InvestmentPlan)

		// Create messages following Python structure
		output = []*schema.Message{
			schema.SystemMessage(systemPromptWithContext),
			schema.UserMessage(userContextMessage),
		}
		return err
	})
	return output, err
}

func NewTraderNode[I, O any](ctx context.Context, cfg *config.Config) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()

	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadTraderMessages))
	_ = g.AddChatModelNode("agent", agents.ChatModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(traderRouter))

	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)

	return g
}
