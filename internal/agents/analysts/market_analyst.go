package analysts

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/consts"
	"github.com/dyike/CortexGo/internal/config"
	"github.com/dyike/CortexGo/internal/dataflows"
	"github.com/dyike/CortexGo/internal/models"
	t_utils "github.com/dyike/CortexGo/internal/utils"
)

func NewMarketAnalyst[I, O any](ctx context.Context, cfg *config.Config) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()

	getMarketDataTool := createMarketDataTool(cfg)
	marketTools := []tool.BaseTool{
		getMarketDataTool,
	}

	chatModel, err := createMarketChatModel(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to create market chat model: %v", err)
	}

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		MaxStep:          6,
		ToolCallingModel: chatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: marketTools,
		},
	})
	if err != nil {
		log.Fatalf("failed to create agent: %v", err)
	}

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
		systemTpl := `You are a helpful AI assistant, collaborating with other assistants.
Use the provided tools to progress towards answering the question.
If you are unable to fully answer, that's OK; another assistant with different tools
will help where you left off. Execute what you can to make progress.
If you or any other assistant has the FINAL TRANSACTION PROPOSAL: **BUY/HOLD/SELL** or deliverable,
prefix your response with FINAL TRANSACTION PROPOSAL: **BUY/HOLD/SELL** so the team knows to stop.

You have access to the following tools:
- get_market_data: Get market data for a specific symbol and date range.

{system_message}

For your reference, the current date is {current_date}. The company we want to look at is {ticker}
`
		systemPrompt, _ := t_utils.LoadPrompt("analysts/market_analyst")
		// 创建prompt模板
		promptTemp := prompt.FromMessages(schema.FString,
			schema.SystemMessage(systemTpl),
			schema.MessagesPlaceholder("user_input", true),
		)
		// Load prompt from external markdown file with context
		context := map[string]any{
			"CompanyOfInterest": state.CompanyOfInterest,
			"trade_date":        state.TradeDate,
			// 添加模板所需的新变量
			"tool_names":     strings.Join(getMarketDataTools(), ","), // 需要实现工具名称获取函数
			"current_date":   time.Now().Format("2006-01-02"),
			"ticker":         state.CompanyOfInterest,
			"system_message": systemPrompt,
		}

		output, err = promptTemp.Format(ctx, context)
		if err != nil {
			log.Printf("MarkteAnalyst failed to format prompt: %v", err)
			return err
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
		// 处理工具返回的消息
		if input != nil {
			if input.Role == schema.Tool && input.ToolName == "get_market_data" {
				marketData := struct {
					Data []*models.MarketData `json:"data"`
				}{}
				_ = json.Unmarshal([]byte(input.Content), &marketData)
				fmt.Println("get_marked_data data: ", marketData.Data)
				state.MarketData = marketData.Data
				state.Goto = consts.MarketAnalyst
				return nil
			}
		}
		// Mark market analyst as complete and set sequential flow
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

func getMarketDataTools() []string {
	return []string{"get_market_data"}
}

// createMarketDataTool creates the market data tool using proper generic types
func createMarketDataTool(cfg *config.Config) tool.BaseTool {
	return utils.NewTool[MarketDataInput, *GetMarketDataOutput](
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

type MarketDataInput struct {
	Symbol string `json:"symbol"`
	Count  int    `json:"count"`
}

type GetMarketDataOutput struct {
	Data []*models.MarketData `json:"data"`
}
