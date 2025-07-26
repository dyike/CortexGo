package researchers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/consts"
	"github.com/dyike/CortexGo/internal/agents"
	"github.com/dyike/CortexGo/internal/models"
)

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
		if state.InvestmentDebateState.Count >= state.InvestmentDebateState.MaxRounds*2 {
			state.DebatePhaseComplete = true
			state.Phase = "trading"
			state.Goto = consts.ResearchManager
		} else {
			state.Goto = consts.BearResearcher
		}
		
		return nil
	})
	return output, nil
}

func loadBullResearcherMessages(ctx context.Context, name string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		systemPrompt := `You are a bullish investment researcher specializing in identifying investment opportunities and positive catalysts.

Your responsibilities:
1. Analyze all available reports to build a bullish investment case
2. Identify positive catalysts, growth opportunities, and upside potential
3. Present compelling arguments for why the stock should be bought
4. Engage in structured debate with the bear researcher

Current context:
- Company: ` + state.CompanyOfInterest + `
- Trade Date: ` + state.TradeDate + `
- Market Analysis: ` + state.MarketReport + `
- Social Analysis: ` + state.SentimentReport + `
- News Analysis: ` + state.NewsReport + `
- Fundamentals Analysis: ` + state.FundamentalsReport + `

Previous debate history:
` + state.InvestmentDebateState.History

		if state.InvestmentDebateState.BearHistory != "" {
			systemPrompt += `

Bear researcher's recent arguments:
` + state.InvestmentDebateState.BearHistory
		}

		systemPrompt += `

When you complete your research, use the submit_bull_research tool to provide:
- Strong bullish arguments with supporting evidence
- Identification of positive catalysts and growth drivers
- Counterarguments to bear case concerns
- Investment recommendation rationale

Focus on building the strongest possible case for investment, backed by data and analysis.`

		output = []*schema.Message{
			schema.SystemMessage(systemPrompt),
		}

		userMessage := fmt.Sprintf("Present your bullish investment case for %s", state.CompanyOfInterest)
		if strings.Contains(state.InvestmentDebateState.CurrentResponse, "Bear:") {
			userMessage += ". Address the bear researcher's concerns and strengthen your bullish arguments."
		}
		output = append(output, schema.UserMessage(userMessage))

		return nil
	})
	return output, err
}

func NewBullResearcherNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
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
