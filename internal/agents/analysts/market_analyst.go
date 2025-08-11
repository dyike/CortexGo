package analysts

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/consts"
	"github.com/dyike/CortexGo/internal/config"
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

		systemPrompt, err1 := utils.LoadPromptWithContext("analysts/market_analyst", context)
		if err1 != nil {
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

func NewMarketAnalystNode[I, O any](ctx context.Context, cfg *config.Config) *compose.Graph[I, O] {
	// 创建 deepseek 模型
	apiKey := cfg.DeepSeekAPIKey
	chatModel, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		APIKey:    apiKey,
		Model:     "deepseek-reasoner",
		MaxTokens: 2000,
	})
	if err != nil {
		log.Printf("MarkteAnalyst failed to create DeepSeek model: %v", err)
	}
	// TODO
	// chatModel.BindTools()

	g := compose.NewGraph[I, O]()

	// Create a typed loader that accepts the correct input type but ignores it
	typedLoader := func(ctx context.Context, input I, opts ...any) ([]*schema.Message, error) {
		return loadMarketAnalystMessages(ctx, "", opts...)
	}

	// Create a typed router that accepts the correct message type
	typedRouter := func(ctx context.Context, input *schema.Message, opts ...any) (O, error) {
		nextNode, err := marketAnalystRouter(ctx, input, opts...)
		if err != nil {
			var zero O
			return zero, err
		}
		// Convert string to O type
		if result, ok := any(nextNode).(O); ok {
			return result, nil
		}
		var zero O
		return zero, nil
	}

	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(typedLoader))
	_ = g.AddChatModelNode("agent", chatModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(typedRouter))

	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)

	return g
}
