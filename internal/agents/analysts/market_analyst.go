package analysts

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	t_utils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/consts"
	"github.com/dyike/CortexGo/internal/config"
	"github.com/dyike/CortexGo/internal/models"
	"github.com/dyike/CortexGo/internal/utils"
)

func router(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
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

func loadMsg(ctx context.Context, name string, opts ...any) ([]*schema.Message, error) {
	var (
		output []*schema.Message
		fErr   error
	)
	err := compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
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
		systemPrompt, _ := utils.LoadPrompt("analysts/market_analyst")
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

		output, fErr = promptTemp.Format(ctx, context)
		if fErr != nil {
			log.Printf("MarkteAnalyst failed to format prompt: %v", fErr)
			return fErr
		}

		marketDataStr := ""
		for _, data := range state.MarketData {
			marketContext := fmt.Sprintf(
				"Symbol(%s) market data on %s: Volume: %d, High: %.2f, Low: %.2f, Open: %.2f, Close: %.2f",
				data.Symbol, data.Date, data.Volume, data.High, data.Low, data.Open, data.Close)
			if marketDataStr != "" {
				marketDataStr += "\n"
			}
			marketDataStr += marketContext
		}
		if marketDataStr != "" {
			output = append(output, schema.UserMessage(marketDataStr))
		}
		return nil
	})
	return output, err
}

func shoudContinueMarket(ctx context.Context, input *schema.Message) (next string, err error) {
	_ = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, st *models.TradingState) error {
		return nil
	})

	// Check if the assistant message contains tool calls
	if input.Role == schema.Assistant && len(input.ToolCalls) > 0 {
		return "tools", nil
	}

	// If it's a tool response, continue to agent
	if input.Role == schema.Tool {
		return "router", nil
	}

	// Default case - end the flow
	return "router", nil
}

func NewMarketAnalystNode[I, O any](ctx context.Context, cfg *config.Config) *compose.Graph[I, O] {
	// 创建 deepseek 模型
	apiKey := cfg.DeepSeekAPIKey
	chatModel, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		APIKey:    apiKey,
		Model:     "deepseek-chat",
		MaxTokens: 2000,
	})
	if err != nil {
		log.Printf("MarkteAnalyst failed to create DeepSeek model: %v", err)
	}
	getMarketDataTool := t_utils.NewTool(
		&schema.ToolInfo{
			Name: "get_market_data",
			Desc: "Get market data for a specific symbol and date range",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"symbol": {
					Type:     "string",
					Desc:     "The stock symbol",
					Required: true,
				},
				"start_date": {
					Type:     "string",
					Desc:     "The start date in YYYY-MM-DD format",
					Required: true,
				},
				"end_date": {
					Type:     "string",
					Desc:     "The end date in YYYY-MM-DD format",
					Required: true,
				},
			}),
		},
		func(ctx context.Context, input *schema.ToolCall) (*GetMarketDataOutput, error) {
			return &GetMarketDataOutput{
				Data: []*models.MarketData{
					{
						Symbol: "UI",
						Date:   "2025-08-06",
						Open:   100,
						High:   101,
						Low:    99,
						Close:  100.5,
						Volume: 1000000,
					},
				},
			}, nil
		},
	)

	info, err := getMarketDataTool.Info(ctx)
	if err != nil {
		log.Printf("MarkteAnalyst failed to get market data tool info: %v", err)
		return nil
	}

	chatModelWithTool, _ := chatModel.WithTools([]*schema.ToolInfo{info})
	if err != nil {
		log.Printf("MarkteAnalyst failed to bind market data tool: %v", err)
		return nil
	}

	toolsNode, err := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
		Tools: []tool.BaseTool{getMarketDataTool},
	})
	if err != nil {
		log.Printf("NewToolNode failed, err=%v", err)
		return nil
	}

	g := compose.NewGraph[I, O]()
	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadMsg))
	_ = g.AddChatModelNode("agent", chatModelWithTool)
	_ = g.AddToolsNode("tools", toolsNode)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(router))

	outMap := map[string]bool{
		"tools":  true,
		"router": true,
	}
	_ = g.AddBranch("agent", compose.NewGraphBranch(shoudContinueMarket, outMap))
	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("tools", "agent")
	_ = g.AddEdge("router", compose.END)

	return g
}

func getMarketDataTools() []string {
	return []string{"get_market_data"}
}

type GetMarketDataInput struct {
	Symbol    string `json:"symbol"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type GetMarketDataOutput struct {
	Data []*models.MarketData `json:"data"`
}
