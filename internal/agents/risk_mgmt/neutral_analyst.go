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
	"github.com/dyike/CortexGo/internal/models"
	"github.com/dyike/CortexGo/internal/utils"
)

func NewNeutralAnalystNode[I, O any](ctx context.Context, cfg *config.Config) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()
	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadNeutralMsg))
	_ = g.AddChatModelNode("agent", agents.ChatModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(neutralRouter))

	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)
	return g
}

func loadNeutralMsg(ctx context.Context, name string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		// Extract risk debate state data
		riskDebateState := state.RiskDebateState
		history := ""
		currentRiskyResponse := ""
		currentSafeResponse := ""

		if riskDebateState != nil {
			history = riskDebateState.History
			currentRiskyResponse = riskDebateState.CurrentRiskyResponse
			currentSafeResponse = riskDebateState.CurrentSafeResponse
		}

		// Load prompt from external markdown file
		systemPrompt, _ := utils.LoadPrompt("risk_mgmt/neutral_debate")

		// Create prompt template
		promptTemp := prompt.FromMessages(schema.FString,
			schema.SystemMessage("{system_message}"),
			schema.MessagesPlaceholder("user_input", true),
		)

		// Load prompt context
		context := map[string]any{
			"system_message":         systemPrompt,
			"trader_decision":        state.TraderInvestmentPlan,
			"market_research_report": state.MarketReport,
			"social_media_report":    state.SocialReport,
			"news_report":            state.NewsReport,
			"fundamentals_report":    state.FundamentalsReport,
			"history":                history,
			"current_risky_response": currentRiskyResponse,
			"current_safe_response":  currentSafeResponse,
		}

		output, err = promptTemp.Format(ctx, context)
		return err
	})
	return output, err
}

func neutralRouter(ctx context.Context, input *schema.Message, opts ...any) (string, error) {
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
			argument := fmt.Sprintf("Neutral Analyst: %s", input.Content)

			// Update the risk debate state with new data following Python logic
			riskDebateState := state.RiskDebateState

			// Update history fields
			riskDebateState.History = riskDebateState.History + "\n" + argument
			riskDebateState.NeutralHistory = riskDebateState.NeutralHistory + "\n" + argument

			// Update latest speaker and response tracking
			riskDebateState.LatestSpeaker = "Neutral"
			riskDebateState.CurrentNeutralResponse = argument

			// Increment count
			riskDebateState.Count = riskDebateState.Count + 1

			// Add the response to the state messages
			state.Messages = append(state.Messages, input)

			filePath := fmt.Sprintf("results/%s/%s", state.CompanyOfInterest, state.TradeDate)
			fileName := "neutral_analyst_report.md"
			if err := utils.WriteMarkdown(filePath, fileName, argument); err != nil {
				log.Printf("Failed to write neutral analyst report: %v", err)
			}
		}

		next := consts.RiskyAnalyst
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
