package analysts

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/consts"
	"github.com/dyike/CortexGo/internal/agents"
	"github.com/dyike/CortexGo/internal/models"
)

func socialAnalystRouter(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		defer func() {
			output = state.Goto
		}()

		// Mark social analyst as complete and set sequential flow
		state.SocialAnalystComplete = true
		state.Goto = consts.NewsAnalyst
		
		if len(input.ToolCalls) > 0 && input.ToolCalls[0].Function.Name == "submit_social_analysis" {
			argMap := map[string]interface{}{}
			_ = json.Unmarshal([]byte(input.ToolCalls[0].Function.Arguments), &argMap)

			if analysis, ok := argMap["analysis"].(string); ok {
				state.SentimentReport = analysis
			}
		}
		return nil
	})
	return output, nil
}

func loadSocialAnalystMessages(ctx context.Context, name string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		systemPrompt := `You are a social media sentiment analyst specializing in financial market sentiment analysis.

Your responsibilities:
1. Analyze social media sentiment from Reddit, Twitter, and other platforms
2. Evaluate public opinion and sentiment trends around the stock
3. Assess social media impact on stock price movements
4. Determine next analysis step in the workflow

Current context:
- Company: ` + state.CompanyOfInterest + `
- Trade Date: ` + state.TradeDate + `
- Previous Market Analysis: ` + state.MarketReport + `

When you complete your analysis, use the submit_social_analysis tool to provide:
- Social media sentiment analysis and trends
- Public opinion assessment and sentiment scoring
- Social media impact evaluation on stock movement
- Next analysis step (news/fundamentals/bull_researcher)

Focus on sentiment trends, discussion volume, and social media influence on market behavior.`

		output = []*schema.Message{
			schema.SystemMessage(systemPrompt),
		}

		userMessage := fmt.Sprintf("Analyze social media sentiment for %s on %s",
			state.CompanyOfInterest, state.TradeDate)
		output = append(output, schema.UserMessage(userMessage))

		return nil
	})
	return output, err
}

func NewSocialMediaAnalystNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()

	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadSocialAnalystMessages))
	_ = g.AddChatModelNode("agent", agents.ChatModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(socialAnalystRouter))

	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)

	return g
}
