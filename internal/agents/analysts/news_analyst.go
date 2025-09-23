package analysts

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/config"
	"github.com/dyike/CortexGo/consts"
	"github.com/dyike/CortexGo/internal/agents"
	"github.com/dyike/CortexGo/internal/models"
	"github.com/dyike/CortexGo/internal/tools"
	"github.com/dyike/CortexGo/internal/utils"
)

func NewNewsAnalyst[I, O any](ctx context.Context, cfg *config.Config) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()
	googleFinanceNewsTool := tools.NewGoogleFinanceNewsTool(cfg)
	googleNewsSearchTool := tools.NewGoogleNewsSearchTool(cfg)
	googleStockNewsTool := tools.NewGoogleStockNewsTool(cfg)

	newsTools := []tool.BaseTool{
		googleFinanceNewsTool,
		googleNewsSearchTool,
		googleStockNewsTool,
	}

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		MaxStep:          40, // 增加最大步数，参考实现用的是40
		ToolCallingModel: agents.ChatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: newsTools,
		},
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

	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadNewsAnalystMsg))
	_ = g.AddLambdaNode("agent", agentLambda)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(newsAnalystRouter))

	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)
	return g
}

func loadNewsAnalystMsg(ctx context.Context, name string, opts ...any) ([]*schema.Message, error) {
	var (
		output []*schema.Message
		err    error
	)
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		systemTpl := `You are a helpful AI assistant, collaborating with other assistants.
Use the provided tools to progress towards answering the question.
If you are unable to fully answer, that's OK; another assistant with different tools will help where you left off. Execute what you can to make progress.
If you or any other assistant has the FINAL TRANSACTION PROPOSAL: **BUY/HOLD/SELL** or deliverable, prefix your response with FINAL TRANSACTION PROPOSAL: **BUY/HOLD/SELL** so the team knows to stop.

You have access to the following tools:
- get_google_finance_news: Pull the latest macro and sector headlines from Google Finance news to understand market-moving narratives.
- search_google_news: Run an advanced Google News query with language, country, and recency filters to collect context-rich coverage.
- get_google_stock_news: Retrieve Google News articles for the target ticker to monitor company announcements, sentiment, and reactions.

{system_message}

For your reference, the current date is {current_date}. The current company we want to analyze is {ticker}.

The output content should be in Chinese.
`
		systemPrompt, _ := utils.LoadPrompt("analysts/news_analyst")
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

func newsAnalystRouter(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		defer func() {
			output = state.Goto
		}()
		if input != nil {
			state.NewsReport = input.Content
			state.Messages = append(state.Messages, input)

			filePath := fmt.Sprintf("results/%s/%s", state.CompanyOfInterest, state.TradeDate)
			fileName := "news_analyst_report.md"
			if err := utils.WriteMarkdown(filePath, fileName, input.Content); err != nil {
				log.Printf("Failed to write news report to file: %v", err)
			}
		}
		// 设置下一步流程
		state.Goto = consts.NewsAnalyst
		return nil
	})
	return output, nil
}
