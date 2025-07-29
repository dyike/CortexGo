package analysts

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/consts"
	"github.com/dyike/CortexGo/internal/agents"
	"github.com/dyike/CortexGo/internal/models"
	"github.com/dyike/CortexGo/internal/utils"
)

func marketAnalystRouter(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		defer func() {
			output = state.Goto
		}()

		// Mark market analyst as complete and set sequential flow
		state.MarketAnalystComplete = true
		state.Goto = consts.SocialMediaAnalyst

		if len(input.ToolCalls) > 0 && input.ToolCalls[0].Function.Name == "submit_market_analysis" {
			argMap := map[string]interface{}{}
			_ = json.Unmarshal([]byte(input.ToolCalls[0].Function.Arguments), &argMap)

			if analysis, ok := argMap["analysis"].(string); ok {
				state.MarketReport = analysis
			}
		}
		return nil
	})
	return output, nil
}

func loadMarketAnalystMessages(ctx context.Context, name string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		// Load prompt from external markdown file with context
		context := map[string]string{
			"CompanyOfInterest": state.CompanyOfInterest,
			"TradeDate":         state.TradeDate,
		}
		
		systemPrompt, err := utils.LoadPromptWithContext("analysts/market_analyst", context)
		if err != nil {
			log.Printf("Failed to load market analyst prompt: %v", err)
			// Fallback to basic prompt if file loading fails
			systemPrompt = "You are a market analyst. Analyze the given market data and provide insights."
		}

		output = []*schema.Message{
			schema.SystemMessage(systemPrompt),
		}

		if state.MarketData != nil {
			marketContext := fmt.Sprintf("Current market data for %s: Price: %.2f, Volume: %d, High: %.2f, Low: %.2f",
				state.MarketData.Symbol, state.MarketData.Price, state.MarketData.Volume,
				state.MarketData.High, state.MarketData.Low)
			output = append(output, schema.UserMessage(marketContext))
		}

		return nil
	})
	return output, err
}

func NewMarketAnalystNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()

	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadMarketAnalystMessages))
	_ = g.AddChatModelNode("agent", agents.ChatModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(marketAnalystRouter))

	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)

	return g
}
