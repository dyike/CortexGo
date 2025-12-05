package analysts

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/config"
	"github.com/dyike/CortexGo/consts"
	"github.com/dyike/CortexGo/internal/agents"
	"github.com/dyike/CortexGo/internal/utils"
	"github.com/dyike/CortexGo/models"
)

func NewFundamentalsAnalystNode[I, O any](ctx context.Context, cfg *config.Config) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()

	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadFundamentalsAnalystMessages))
	_ = g.AddChatModelNode("agent", agents.ChatModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(fundamentalsAnalystRouter))

	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)

	return g
}

func fundamentalsAnalystRouter(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		defer func() {
			output = state.Goto
		}()

		// Mark fundamentals analyst as complete and transition to debate phase
		state.FundamentalsAnalystComplete = true
		state.AnalysisPhaseComplete = true
		state.Phase = "debate"
		state.Goto = consts.BullResearcher

		var reportContent string
		if input != nil && len(input.ToolCalls) > 0 && input.ToolCalls[0].Function.Name == "submit_fundamentals_analysis" {
			argMap := map[string]interface{}{}
			_ = json.Unmarshal([]byte(input.ToolCalls[0].Function.Arguments), &argMap)

			if analysis, ok := argMap["analysis"].(string); ok {
				state.FundamentalsReport = analysis
				reportContent = analysis
			}
		}

		if reportContent == "" && input != nil {
			reportContent = input.Content
			if state.FundamentalsReport == "" {
				state.FundamentalsReport = reportContent
			}
		}

		if input != nil {
			state.Messages = append(state.Messages, input)
		}

		if reportContent != "" {
			filePath := fmt.Sprintf("results/%s/%s", state.CompanyOfInterest, state.TradeDate)
			fileName := "fundamentals_analyst_report.md"
			if err := utils.WriteMarkdown(filePath, fileName, reportContent); err != nil {
				log.Printf("Failed to write fundamentals report to file: %v", err)
			}
		}
		return nil
	})
	return output, nil
}

func loadFundamentalsAnalystMessages(ctx context.Context, name string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		systemTpl := `You are a helpful AI assistant, collaborating with other assistants.
Use the provided tools to progress towards answering the question.
If you are unable to fully answer, that's OK; another assistant with different tools
will help where you left off. Execute what you can to make progress.
If you or any other assistant has the FINAL TRANSACTION PROPOSAL: **BUY/HOLD/SELL** or deliverable,
prefix your response with FINAL TRANSACTION PROPOSAL: **BUY/HOLD/SELL** so the team knows to stop.

You have access to the following tools:


{system_message}

For your reference, the current date is {current_date}. The company we want to look at is {ticker} .

The output content should be in Chinese.
`
		systemPrompt, _ := utils.LoadPrompt("analysts/fundamentals_analyst")
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
