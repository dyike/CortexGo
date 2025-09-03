package researchers

import (
	"context"
	"encoding/json"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/internal/agents"
	"github.com/dyike/CortexGo/internal/config"
	"github.com/dyike/CortexGo/internal/models"
	"github.com/dyike/CortexGo/internal/utils"
)

func NewBullResearcherNode[I, O any](ctx context.Context, cfg *config.Config) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()

	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadBullResearcherMessages))
	_ = g.AddChatModelNode("agent", agents.ChatModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(bullResearcherRouter))

	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)

	return g
}

func bullResearcherRouter(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		defer func() {
			output = state.Goto
		}()

		if len(input.ToolCalls) > 0 && input.ToolCalls[0].Function.Name == "submit_bull_research" {
			argMap := map[string]interface{}{}
			_ = json.Unmarshal([]byte(input.ToolCalls[0].Function.Arguments), &argMap)

			if research, ok := argMap["research"].(string); ok {
				state.InvestmentDebateState.BullHistory += research + "\n"
				state.InvestmentDebateState.CurrentResponse = "Bull: " + research
				state.InvestmentDebateState.Count++
			}
		}

		// Use conditional logic to determine next step based on debate rounds
		// if state.InvestmentDebateState.Count >= state.InvestmentDebateState.MaxRounds*2 {
		// 	state.DebatePhaseComplete = true
		// 	state.Phase = "trading"
		// 	state.Goto = consts.ResearchManager
		// } else {
		// 	state.Goto = consts.BearResearcher
		// }

		return nil
	})
	return output, nil
}

func loadBullResearcherMessages(ctx context.Context, name string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		ptl, _ := utils.LoadPrompt("researchers/bull_resarcher")
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
