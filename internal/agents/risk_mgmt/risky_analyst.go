package risk_mgmt

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
	"github.com/dyike/CortexGo/internal/utils"
	"github.com/dyike/CortexGo/models"
)

func NewRiskyAnalystNode[I, O any](ctx context.Context, cfg *config.Config) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()
	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadRiskyMsg))
	_ = g.AddChatModelNode("agent", agents.ChatModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(riskyRouter))
	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)
	return g
}

func loadRiskyMsg(ctx context.Context, name string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		// Extract risk debate state data
		riskDebateState := state.RiskDebateState
		history := ""
		currentSafeResponse := ""
		currentNeutralResponse := ""

		if riskDebateState != nil {
			history = riskDebateState.History
			currentSafeResponse = riskDebateState.CurrentSafeResponse
			currentNeutralResponse = riskDebateState.CurrentNeutralResponse
		}

		// Load prompt from external markdown file
		systemPrompt, _ := utils.LoadPrompt("risk_mgmt/risky_debate")

		// Create prompt template
		promptTemp := prompt.FromMessages(schema.FString,
			schema.SystemMessage("{system_message}"),
			schema.MessagesPlaceholder("user_input", true),
		)

		// Load prompt context
		context := map[string]any{
			"system_message":           systemPrompt,
			"trader_decision":          state.TraderInvestmentPlan,
			"market_research_report":   state.MarketReport,
			"social_media_report":      state.SocialReport,
			"news_report":              state.NewsReport,
			"fundamentals_report":      state.FundamentalsReport,
			"history":                  history,
			"current_safe_response":    currentSafeResponse,
			"current_neutral_response": currentNeutralResponse,
		}

		output, err = promptTemp.Format(ctx, context)
		return err
	})
	return output, err
}

func riskyRouter(ctx context.Context, input *schema.Message, opts ...any) (string, error) {
	var (
		output string
		err    error
	)

	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		defer func() {
			output = state.Goto
		}()

		if input != nil && state.RiskDebateState != nil {
			// Create the argument string following the Python pattern
			argument := fmt.Sprintf("Risky Analyst: %s", input.Content)

			// Update the risk debate state with new data following Python logic
			riskDebateState := state.RiskDebateState

			// Update history fields
			riskDebateState.History = riskDebateState.History + "\n" + argument
			riskDebateState.RiskyHistory = riskDebateState.RiskyHistory + "\n" + argument

			// Update latest speaker and response tracking
			riskDebateState.LatestSpeaker = "Risky"
			riskDebateState.CurrentRiskyResponse = argument

			// Increment count
			riskDebateState.Count = riskDebateState.Count + 1

			// Add the response to the state messages
			state.Messages = append(state.Messages, input)

			filePath := fmt.Sprintf("results/%s/%s", state.CompanyOfInterest, state.TradeDate)
			fileName := "risky_analyst_report.md"
			if err := utils.WriteMarkdown(filePath, fileName, argument); err != nil {
				log.Printf("Failed to write risky analyst report: %v", err)
			}
		}

		next := consts.SafeAnalyst
		if state.RiskDebateState != nil {
			if state.RiskDebateState.Count >= 3 {
				next = consts.RiskJudge
			}
		}
		state.Goto = next

		return nil
	})

	return output, err
}
