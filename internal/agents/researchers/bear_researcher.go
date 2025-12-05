package researchers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/config"
	"github.com/dyike/CortexGo/consts"
	"github.com/dyike/CortexGo/internal/agents"
	"github.com/dyike/CortexGo/internal/utils"
	"github.com/dyike/CortexGo/models"
)

func NewBearResearcherNode[I, O any](ctx context.Context, cfg *config.Config) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()
	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadBearResearcherMessages))
	_ = g.AddChatModelNode("agent", agents.ChatModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(bearResearcherRouter))

	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)

	return g
}

func loadBearResearcherMessages(ctx context.Context, name string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		ptl, err := utils.LoadPrompt("researchers/bear_researcher")
		if err != nil {
			return err
		}

		// 创建prompt模板
		promptTemp := prompt.FromMessages(schema.FString,
			schema.UserMessage(ptl),
			schema.MessagesPlaceholder("user_input", true),
		)
		// Load prompt from external markdown file with context
		context := map[string]any{
			"market_research_report": state.MarketReport,
			"social_media_report":    state.SocialReport,
			"news_report":            state.NewsReport,
			"fundamentals_report":    state.FundamentalsReport,
			"history":                "",
			"current_response":       "",
			"past_memory_str":        "",
		}

		output, err = promptTemp.Format(ctx, context)
		return nil
	})
	return output, err
}

func bearResearcherRouter(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		defer func() {
			output = state.Goto
		}()
		if input != nil && state.InvestmentDebateState != nil {
			argument := strings.TrimSpace(input.Content)
			if len(input.ToolCalls) > 0 && input.ToolCalls[0].Function.Name == "submit_bear_research" {
				argMap := map[string]any{}
				_ = json.Unmarshal([]byte(input.ToolCalls[0].Function.Arguments), &argMap)
				if research, ok := argMap["research"].(string); ok && strings.TrimSpace(research) != "" {
					argument = strings.TrimSpace(research)
				}
			}
			if argument == "" {
				argument = "(no argument provided)"
			}
			labeledArgument := "Bear Analyst: " + argument
			investmentDebateState := state.InvestmentDebateState
			investmentDebateState.History = strings.TrimSpace(investmentDebateState.History + "\n" + labeledArgument)
			investmentDebateState.BearHistory = strings.TrimSpace(investmentDebateState.BearHistory + "\n" + labeledArgument)
			investmentDebateState.CurrentResponse = labeledArgument
			investmentDebateState.Count++
			state.Messages = append(state.Messages, input)

			filePath := fmt.Sprintf("results/%s/%s", state.CompanyOfInterest, state.TradeDate)
			fileName := "bear_researcher_report.md"
			if err := utils.WriteMarkdown(filePath, fileName, labeledArgument); err != nil {
				log.Printf("Failed to write bear researcher report: %v", err)
			}
		}

		if state.InvestmentDebateState != nil {
			next := consts.BullResearcher
			if state.InvestmentDebateState.Count >= 2 {
				next = consts.ResearchManager
			}
			state.Goto = next
		}

		// Use conditional logic to determine next step based on debate rounds
		// if state.InvestmentDebateState.Count >= state.InvestmentDebateState.MaxRounds*2 {
		// 	state.DebatePhaseComplete = true
		// 	state.Phase = "trading"
		// 	state.Goto = consts.ResearchManager
		// } else {
		// 	state.Goto = consts.BullResearcher
		// }

		return nil
	})
	return output, nil
}
