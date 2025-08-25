package analysts

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/tool"
	t_utils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/consts"
	"github.com/dyike/CortexGo/internal/config"
	"github.com/dyike/CortexGo/internal/dataflows"
	"github.com/dyike/CortexGo/internal/models"
)

func NewMarketAnalyst[I, O any](ctx context.Context, cfg *config.Config) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()

	getMarketDataTool := createMarketDataTool(cfg)
	marketTools := []tool.BaseTool{
		getMarketDataTool,
	}

	log.Printf("Created %d market tools", len(marketTools))

	// Test tool info
	if toolInfo, err := getMarketDataTool.Info(ctx); err != nil {
		log.Printf("Failed to get tool info: %v", err)
	} else {
		log.Printf("Tool info - Name: %s, Desc: %s", toolInfo.Name, toolInfo.Desc)
	}

	chatModel, err := createMarketChatModel(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to create market chat model: %v", err)
	}

	// 添加工具调用检查器（来自参考实现）
	toolCallChecker := func(ctx context.Context, sr *schema.StreamReader[*schema.Message]) (bool, error) {
		defer sr.Close()
		for {
			msg, err := sr.Recv()
			if err != nil {
				if err.Error() == "EOF" {
					return false, nil
				}
				return false, err
			}
			if len(msg.ToolCalls) > 0 {
				log.Printf("StreamToolCallChecker detected tool calls: %d", len(msg.ToolCalls))
				return true, nil
			}
		}
	}

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		MaxStep:          40,  // 增加最大步数，参考实现用的是40
		ToolCallingModel: chatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: marketTools,
		},
		// 添加调试选项
		MessageModifier: func(ctx context.Context, input []*schema.Message) []*schema.Message {
			log.Printf("Agent processing %d messages", len(input))
			for i, msg := range input {
				log.Printf("Message %d: Role=%s, ToolCalls=%d, Content length=%d", i, msg.Role, len(msg.ToolCalls), len(msg.Content))
			}
			return input
		},
		// 添加流式工具调用检查器
		StreamToolCallChecker: toolCallChecker,
	})
	if err != nil {
		log.Fatalf("failed to create agent: %v", err)
	}

	log.Printf("Created ReAct agent with %d tools", len(marketTools))

	agentLambda, err := compose.AnyLambda(agent.Generate, agent.Stream, nil, nil)
	if err != nil {
		log.Fatalf("failed to create agent lambda: %v", err)
	}

	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadMarketMsg))
	_ = g.AddLambdaNode("agent", agentLambda)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(marketRouter))

	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)
	return g
}

func loadMarketMsg(ctx context.Context, name string, opts ...any) ([]*schema.Message, error) {
	var (
		output []*schema.Message
		err    error
	)
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		systemMsg := fmt.Sprintf(`You are a market analyst assistant. Your task is to analyze trading opportunities for %s on %s.

You must use the get_market_data tool to fetch the latest market data before providing any analysis.

Instructions:
1. First call get_market_data with symbol="%s" to get market data
2. Analyze the market data (price trends, volume, volatility)
3. Provide a comprehensive trading recommendation (BUY/HOLD/SELL)
4. Include specific reasoning based on the data

Current date: %s`,
			state.CompanyOfInterest,
			state.TradeDate,
			state.CompanyOfInterest,
			time.Now().Format("2006-01-02"))

		userMsg := fmt.Sprintf("Please analyze trading opportunities for %s. You MUST start by calling the get_market_data tool with symbol=%s to get recent market data, then provide analysis based on the data.", state.CompanyOfInterest, state.CompanyOfInterest)

		output = []*schema.Message{
			schema.SystemMessage(systemMsg),
			schema.UserMessage(userMsg),
		}

		return nil
	})
	return output, err
}

func marketRouter(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		defer func() {
			output = state.Goto
		}()

		// 调试：打印接收到的消息
		if input != nil {
			log.Printf("Market router received message - Role: %s, Content length: %d", input.Role, len(input.Content))
			if len(input.Content) > 200 {
				log.Printf("Message preview: %s...", input.Content[:200])
			} else {
				log.Printf("Full message: %s", input.Content)
			}

			// 检查是否有工具调用
			if len(input.ToolCalls) > 0 {
				log.Printf("WARNING: Message contains %d tool calls", len(input.ToolCalls))
			}
			
			// 存储市场分析报告（无论是否有工具调用）
			state.MarketReport = input.Content
			state.Messages = append(state.Messages, input)
			
			log.Printf("Market analysis step completed. Content length: %d", len(input.Content))
		}

		// 设置下一步流程
		state.Goto = consts.NewsAnalyst
		return nil
	})
	return output, nil
}

func createMarketChatModel(ctx context.Context, cfg *config.Config) (*openai.ChatModel, error) {
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
	return chatModel, nil
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
			log.Printf("Market data tool called with input: %+v", input)

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

type MarketDataInput struct {
	Symbol string `json:"symbol"`
	Count  int    `json:"count"`
}

type GetMarketDataOutput struct {
	Data []*models.MarketData `json:"data"`
}
