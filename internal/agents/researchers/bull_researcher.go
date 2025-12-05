package researchers

import (
	"context"
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
		if input != nil && state.InvestmentDebateState != nil {
			argument := strings.TrimSpace(input.Content)
			if argument == "" {
				argument = "(no argument provided)"
			}
			labeledArgument := "Bull Analyst: " + argument
			investmentDebateState := state.InvestmentDebateState
			investmentDebateState.History = strings.TrimSpace(investmentDebateState.History + "\n" + labeledArgument)
			investmentDebateState.BullHistory = strings.TrimSpace(investmentDebateState.BullHistory + "\n" + labeledArgument)
			investmentDebateState.CurrentResponse = labeledArgument
			investmentDebateState.Count++
			state.Messages = append(state.Messages, input)

			filePath := fmt.Sprintf("results/%s/%s", state.CompanyOfInterest, state.TradeDate)
			fileName := "bull_researcher_report.md"
			if err := utils.WriteMarkdown(filePath, fileName, labeledArgument); err != nil {
				log.Printf("Failed to write bull researcher report: %v", err)
			}
		}

		if state.InvestmentDebateState != nil {
			next := consts.BearResearcher
			if state.InvestmentDebateState.Count >= 2 {
				next = consts.ResearchManager
			}
			state.Goto = next
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

		ptl, err := utils.LoadPrompt("researchers/bull_researcher")
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
			"history":                history,
			"current_response":       currentResponse,
			"past_memory_str":        pastMemoryStr,
		}

		output, err = promptTemp.Format(ctx, context)
		return nil
	})
	return output, err
}
