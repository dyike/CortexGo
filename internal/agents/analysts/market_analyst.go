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

func marketAnalystRouter(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	fmt.Printf("++=====+++++, %+v  \n", input)
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

func loadMarketAnalystMessages(ctx context.Context, name string, opts ...any) ([]*schema.Message, error) {
	var (
		output []*schema.Message
		fErr   error
	)
	err := compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		// Load prompt from external markdown file with context
		context := map[string]any{
			"CompanyOfInterest": state.CompanyOfInterest,
			"TradeDate":         state.TradeDate,
			// 添加模板所需的新变量
			"ToolNames":   strings.Join(getMarketDataTools(), ","), // 需要实现工具名称获取函数
			"CurrentDate": time.Now().Format("2006-01-02"),
			"Ticker":      state.CompanyOfInterest,
		}

		systemTpl, _ := utils.LoadPrompt("analysts/market_analyst")

		// 创建prompt模板
		promptTemp := prompt.FromMessages(schema.FString,
			schema.SystemMessage(systemTpl),
			schema.MessagesPlaceholder("user_input", true),
		)

		output, fErr = promptTemp.Format(ctx, context)
		if fErr != nil {
			log.Printf("MarkteAnalyst failed to format prompt: %v", fErr)
			return fErr
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
		Model:     "deepseek-chat",
		MaxTokens: 2000,
	})
	if err != nil {
		log.Printf("MarkteAnalyst failed to create DeepSeek model: %v", err)
	}
	getMarketDataTool := t_utils.NewTool(
		&schema.ToolInfo{
			Name: "get_market_data",
			Desc: "Get market data",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"symbol": {
					Type: "String",
					Desc: "The stock symbol",
				},
				"start_date": {
					Type: "String",
					Desc: "The start date",
				},
				"end_date": {
					Type: "String",
					Desc: "The end date",
				},
			}),
		},
		func(ctx context.Context, input *schema.ToolCall) (*GetMarketDataOutput, error) {
			return &GetMarketDataOutput{
				Data: []*MarketData{
					{
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

	err = chatModel.BindTools([]*schema.ToolInfo{info})
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

	// Create a typed loader that accepts the correct input type but ignores it
	typedLoader := func(ctx context.Context, input I, opts ...any) ([]*schema.Message, error) {
		return loadMarketAnalystMessages(ctx, "", opts...)
	}

	// Create a typed router that accepts the tools node's output type ([]*schema.Message)
	typedRouter := func(ctx context.Context, input []*schema.Message, opts ...any) (O, error) {
		// Take the first message from tools output
		if len(input) == 0 {
			var zero O
			return zero, fmt.Errorf("no messages from tools node")
		}

		nextNode, err := marketAnalystRouter(ctx, input[0], opts...) // Pass first message
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
	_ = g.AddToolsNode("tools", toolsNode)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(typedRouter))

	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "tools")
	_ = g.AddEdge("tools", "router")
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
	Data []*MarketData `json:"data"`
}

type MarketData struct {
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume int64   `json:"volume"`
}
