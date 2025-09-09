package researchers

import (
	"context"
	"strings"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/config"
	"github.com/dyike/CortexGo/internal/agents"
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
		if input != nil {
			argument := "Bull Analyst: " + input.Content
			investmentDebateState := state.InvestmentDebateState
			history := investmentDebateState.History
			bullHistory := investmentDebateState.BullHistory
			state.InvestmentDebateState.History = history + "\n" + argument
			state.InvestmentDebateState.BullHistory = bullHistory + "\n" + argument
			state.InvestmentDebateState.CurrentResponse = argument
			state.InvestmentDebateState.Count += 1
		}
		return nil
	})
	return output, nil
}

func loadBullResearcherMessages(ctx context.Context, name string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		investmentDebateState := state.InvestmentDebateState
		history := investmentDebateState.History
		currentResponse := investmentDebateState.CurrentResponse

		// TODO
		// marketResearchReport := state.MarketReport
		// sentimentReport := state.SentimentReport
		// newsReport := state.NewsReport
		// fundamentalsReport := state.FundamentalsReport
		// Create current situation summary
		// currSituation := fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s",
		// marketResearchReport, sentimentReport, newsReport, fundamentalsReport)
		// Get past memories

		// pastMemories := memory.GetMemories(currSituation, 2)
		var pastMemoryStr strings.Builder
		// for i, rec := range pastMemories {
		// 	pastMemoryStr.WriteString(fmt.Sprintf("%d. %s\n\n", i+1, rec.Recommendation))
		// }

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
			"history":                history,
			"current_response":       currentResponse,
			"past_memory_str":        pastMemoryStr,
		}

		output, err = promptTemp.Format(ctx, context)
		return nil
	})
	return output, err
}
