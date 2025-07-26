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

func bearResearcherRouter(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		defer func() {
			output = state.Goto
		}()

		state.Goto = consts.BullResearcher
		if len(input.ToolCalls) > 0 && input.ToolCalls[0].Function.Name == "submit_bear_research" {
			argMap := map[string]interface{}{}
			_ = json.Unmarshal([]byte(input.ToolCalls[0].Function.Arguments), &argMap)

			if research, ok := argMap["research"].(string); ok {
				state.InvestmentDebateState.BearHistory += research + "\n"
				state.InvestmentDebateState.CurrentResponse = "Bear: " + research
				state.InvestmentDebateState.Count++
			}

			// 决定下一步：如果讨论轮数足够，去研究经理；否则回到牛市研究员
			if state.InvestmentDebateState.Count >= 4 { // 2轮辩论
				state.Goto = consts.ResearchManager
			} else {
				state.Goto = consts.BullResearcher
			}
		}
		return nil
	})
	return output, nil
}

func loadBearResearcherMessages(ctx context.Context, name string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		systemPrompt := `You are a bearish investment researcher specializing in identifying investment risks and negative catalysts.

Your responsibilities:
1. Analyze all available reports to build a bearish investment case
2. Identify risks, threats, and potential downside scenarios
3. Present compelling arguments for why the stock should be avoided or sold
4. Engage in structured debate with the bull researcher

Current context:
- Company: ` + state.CompanyOfInterest + `
- Trade Date: ` + state.TradeDate + `
- Market Analysis: ` + state.MarketReport + `
- Social Analysis: ` + state.SentimentReport + `
- News Analysis: ` + state.NewsReport + `
- Fundamentals Analysis: ` + state.FundamentalsReport + `

Previous debate history:
` + state.InvestmentDebateState.History

		if state.InvestmentDebateState.BullHistory != "" {
			systemPrompt += `

Bull researcher's recent arguments:
` + state.InvestmentDebateState.BullHistory
		}

		systemPrompt += `

When you complete your research, use the submit_bear_research tool to provide:
- Strong bearish arguments with supporting evidence
- Identification of risks, threats, and negative catalysts
- Counterarguments to bull case points
- Investment avoidance rationale

Focus on building the strongest possible case against investment, backed by data and analysis.`

		output = []*schema.Message{
			schema.SystemMessage(systemPrompt),
		}

		userMessage := fmt.Sprintf("Present your bearish investment case for %s", state.CompanyOfInterest)
		if strings.Contains(state.InvestmentDebateState.CurrentResponse, "Bull:") {
			userMessage += ". Address the bull researcher's arguments and strengthen your bearish concerns."
		}
		output = append(output, schema.UserMessage(userMessage))

		return nil
	})
	return output, err
}

func NewBearResearcherNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
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
