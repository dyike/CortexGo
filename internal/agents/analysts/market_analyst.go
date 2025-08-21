package analysts

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	t_utils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/consts"
	"github.com/dyike/CortexGo/internal/config"
	"github.com/dyike/CortexGo/internal/dataflows"
	"github.com/dyike/CortexGo/internal/models"
)

// createMarketAnalystGraph creates a simplified graph for the MarketAnalyst using standard chat model
// instead of the nested ReAct agent to avoid conflicts
func CreateMarketAnalystGraph[I, O any](ctx context.Context, cfg *config.Config) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()

	// Create a simple loader that creates a market analysis request
	typedLoader := func(ctx context.Context, input I, opts ...any) ([]*schema.Message, error) {
		var messages []*schema.Message

		// Try to process state if it's available
		err := compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
			if state != nil {
				// Create system and user messages for market analysis
				systemMsg := schema.SystemMessage(`You are a market analyst. Use the get_market_data tool to analyze trading opportunities and provide recommendations. Be specific and actionable in your analysis.`)
				userMsg := schema.UserMessage(fmt.Sprintf("Analyze trading opportunities for %s on %s. Use the get_market_data tool to fetch the latest market data and provide a detailed analysis with recommendations.",
					state.CompanyOfInterest, state.TradeDate))
				messages = []*schema.Message{systemMsg, userMsg}
			} else {
				// Fallback messages
				messages = []*schema.Message{
					schema.SystemMessage("You are a market analyst."),
					schema.UserMessage("Analyze current market conditions"),
				}
			}
			return nil
		})

		if err != nil {
			// Fallback if state processing fails
			messages = []*schema.Message{
				schema.SystemMessage("You are a market analyst."),
				schema.UserMessage("Analyze current market conditions"),
			}
		}

		return messages, nil
	}

	// Create a router that processes the result and updates state
	typedRouter := func(ctx context.Context, input *schema.Message, opts ...any) (O, error) {
		// Update trading state based on the analysis result
		err := compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
			if state != nil {
				// Track all messages flowing through the router (including tool results)
				if input != nil {
					state.Messages = append(state.Messages, input)
					log.Printf("Router: Added message to state - Role: %s, Content length: %d", input.Role, len(input.Content))
				}

				// Store the market analysis result
				state.Goto = consts.SocialMediaAnalyst // Set next node
				if input != nil {
					state.MarketReport = input.Content
					maxLen := 100
					if len(input.Content) < maxLen {
						maxLen = len(input.Content)
					}
					log.Printf("Market analysis completed, result stored: %s", input.Content[:maxLen])
				}
			}
			return nil
		})

		if err != nil {
			log.Printf("Failed to update trading state: %v", err)
		}

		// Convert result to expected output type
		nextNode := consts.SocialMediaAnalyst
		if result, ok := any(nextNode).(O); ok {
			return result, nil
		}
		var zero O
		return zero, nil
	}

	// Create a chat model with the market data tool
	chatModel, err := createMarketAnalystChatModel(ctx, cfg)
	if err != nil {
		log.Printf("Failed to create market analyst chat model: %v", err)
		// Return empty graph as fallback
		return g
	}

	// Create tools node with the market data tool
	toolsNode, err := createMarketAnalystTools(ctx, cfg)
	if err != nil {
		log.Printf("Failed to create market analyst tools: %v", err)
		// Return empty graph as fallback
		return g
	}

	// Build the graph with standard ReAct pattern
	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(typedLoader))
	_ = g.AddChatModelNode("agent", chatModel)
	_ = g.AddToolsNode("tools", toolsNode)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(typedRouter))

	// Add edges
	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")

	// Add conditional branch from agent
	agentBranchFunc := func(ctx context.Context, input *schema.Message) (string, error) {
		// Check if the assistant message contains tool calls
		if input.Role == schema.Assistant && len(input.ToolCalls) > 0 {
			return "tools", nil
		}
		return "router", nil
	}

	outMap := map[string]bool{
		"tools":  true,
		"router": true,
	}
	_ = g.AddBranch("agent", compose.NewGraphBranch(agentBranchFunc, outMap))
	_ = g.AddEdge("tools", "agent")
	_ = g.AddEdge("router", compose.END)

	return g
}

// createMarketAnalystChatModel creates a chat model with market data tools
func createMarketAnalystChatModel(ctx context.Context, cfg *config.Config) (model.ToolCallingChatModel, error) {
	// Create DeepSeek model
	maxTokens := 8192
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL:   "https://api.deepseek.com/v1",
		APIKey:    cfg.DeepSeekAPIKey,
		Model:     "deepseek-chat",
		MaxTokens: &maxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create DeepSeek model: %v", err)
	}

	// Get market data tool info
	marketDataTool := createMarketDataTool(cfg)
	info, err := marketDataTool.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get market data tool info: %v", err)
	}

	// Bind tools to model
	chatModelWithTool, err := chatModel.WithTools([]*schema.ToolInfo{info})
	if err != nil {
		return nil, fmt.Errorf("failed to bind market data tool: %v", err)
	}

	return chatModelWithTool, nil
}

// createMarketAnalystTools creates the tools node for market analysis
func createMarketAnalystTools(ctx context.Context, cfg *config.Config) (*compose.ToolsNode, error) {
	marketDataTool := createMarketDataTool(cfg)

	return compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
		Tools: []tool.BaseTool{marketDataTool},
	})
}

// MarketDataInput represents the input parameters for the market data tool
type MarketDataInput struct {
	Symbol string `json:"symbol"`
	Count  int    `json:"count"`
}

// createMarketDataTool creates the market data tool using proper generic types
func createMarketDataTool(cfg *config.Config) tool.BaseTool {
	return t_utils.NewTool[MarketDataInput, *GetMarketDataOutput](
		&schema.ToolInfo{
			Name: "get_market_data",
			Desc: "Get market data for a specific symbol and date range",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"symbol": {
					Type:     "string",
					Desc:     "The stock symbol",
					Required: true,
				},
				"count": {
					Type:     "integer",
					Desc:     "Number of days to retrieve (default: 30)",
					Required: false,
				},
			}),
		},
		func(ctx context.Context, input MarketDataInput) (*GetMarketDataOutput, error) {
			// Debug: Log the input
			// log.Printf("Market data tool called with input: %+v", input)

			if input.Symbol == "" {
				return nil, fmt.Errorf("symbol parameter is required")
			}

			count := input.Count
			if count <= 0 {
				count = 30 // default
			}

			// Create Longport client for real market data
			longportClient, err := dataflows.NewLongportClient(cfg)
			if err != nil {
				log.Printf("Failed to create Longport client, using mock data: %v", err)
				longportClient = nil
			}

			// Try to get real market data from Longport
			if longportClient != nil {
				sticks, err := longportClient.GetSticksWithDay(ctx, input.Symbol, count)
				if err == nil && len(sticks) > 0 {
					marketData := make([]*models.MarketData, 0, len(sticks))
					for _, stick := range sticks {
						// Convert Unix timestamp to date string
						date := time.Unix(stick.Timestamp, 0).Format("2006-01-02")

						// Convert decimal values to float64
						open, _ := stick.Open.Float64()
						high, _ := stick.High.Float64()
						low, _ := stick.Low.Float64()
						close, _ := stick.Close.Float64()

						marketData = append(marketData, &models.MarketData{
							Symbol: input.Symbol,
							Date:   date,
							Open:   open,
							High:   high,
							Low:    low,
							Close:  close,
							Volume: stick.Volume,
						})
					}
					// log.Printf("Successfully retrieved %d market data records for %s", len(marketData), input.Symbol)
					return &GetMarketDataOutput{Data: marketData}, nil
				}
				log.Printf("Failed to get real market data for %s: %v", input.Symbol, err)
			}

			// Fallback to mock data
			log.Printf("Using mock market data for %s", input.Symbol)
			return &GetMarketDataOutput{
				Data: []*models.MarketData{
					{
						Symbol: input.Symbol,
						Date:   time.Now().Format("2006-01-02"),
						Open:   100.0,
						High:   101.0,
						Low:    99.0,
						Close:  100.5,
						Volume: int64(1000000),
					},
				},
			}, nil
		},
	)
}

type GetMarketDataInput struct {
	Symbol    string `json:"symbol"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type GetMarketDataOutput struct {
	Data []*models.MarketData `json:"data"`
}
