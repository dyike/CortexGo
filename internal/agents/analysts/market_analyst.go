package analysts

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/consts"
	"github.com/dyike/CortexGo/internal/agents"
	"github.com/dyike/CortexGo/internal/config"
	"github.com/dyike/CortexGo/internal/models"
	"github.com/dyike/CortexGo/internal/tools"
	"github.com/dyike/CortexGo/internal/utils"
)

func NewMarketAnalyst[I, O any](ctx context.Context, cfg *config.Config) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()
	getMarketDataTool := tools.NewMarketool(cfg)
	marketTools := []tool.BaseTool{
		getMarketDataTool,
	}
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

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		MaxStep:          40, // 增加最大步数，参考实现用的是40
		ToolCallingModel: chatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: marketTools,
		},
		// 添加调试选项
		// MessageModifier: func(ctx context.Context, input []*schema.Message) []*schema.Message {
		// 	log.Printf("Agent processing %d messages", len(input))
		// 	for i, msg := range input {
		// 		log.Printf("Message %d: Role=%s, ToolCalls=%d, Content length=%d", i, msg.Role, len(msg.ToolCalls), len(msg.Content))
		// 	}
		// 	return input
		// },
		// 添加流式工具调用检查器
		StreamToolCallChecker: agents.ToolCallChecker,
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
			"current_date":      time.Now().Format("2006-01-02"),
			"ticker":            state.CompanyOfInterest,
			"system_message":    systemPrompt,
		}

		output, err = promptTemp.Format(ctx, context)
		return nil
	})
	return output, err
}

func marketRouter(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		defer func() {
			output = state.Goto
		}()
		if input != nil {
			// 存储市场分析报告（无论是否有工具调用）
			// TODO
			state.MarketReport = input.Content
			state.Messages = append(state.Messages, input)
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
