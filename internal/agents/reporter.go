package agents

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func reporterRouter(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*TradingState](ctx, func(_ context.Context, state *TradingState) error {
		defer func() {
			output = state.Goto
		}()

		state.Goto = compose.END
		if len(input.ToolCalls) > 0 && input.ToolCalls[0].Function.Name == "generate_final_report" {
			argMap := map[string]interface{}{}
			_ = json.Unmarshal([]byte(input.ToolCalls[0].Function.Arguments), &argMap)

			if state.Metadata == nil {
				state.Metadata = make(map[string]interface{})
			}
			state.Metadata["final_report"] = argMap["report"]
			state.Metadata["executive_summary"] = argMap["executive_summary"]

			output = compose.END
		}
		return nil
	})
	return output, nil
}

func loadReporterMessages(ctx context.Context, name string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*TradingState](ctx, func(_ context.Context, state *TradingState) error {
		systemPrompt := `You are a financial report writer responsible for generating comprehensive trading reports.

Your responsibilities:
1. Summarize all analysis and research findings
2. Present the final trading decision with clear rationale
3. Highlight key risks and opportunities
4. Provide executive summary for stakeholders

Current context:
- Symbol: ` + state.CurrentSymbol + `
- Date: ` + state.CurrentDate.Format("2006-01-02") + `

Analysis Reports:`

		for _, report := range state.Reports {
			systemPrompt += fmt.Sprintf("\n- %s: %s (Recommendation: %s, Confidence: %.1f)",
				report.Analyst, report.Analysis, report.Rating, report.Confidence)
		}

		if state.Decision != nil {
			systemPrompt += fmt.Sprintf(`

Final Trading Decision:
- Action: %s
- Quantity: %.0f shares
- Price: %.2f
- Reasoning: %s
- Confidence: %.1f`,
				state.Decision.Action, state.Decision.Quantity,
				state.Decision.Price, state.Decision.Reason, state.Decision.Confidence)
		}

		systemPrompt += `

Generate a comprehensive final report using the generate_final_report tool.
Include:
- Executive summary
- Analysis overview
- Trading rationale
- Risk assessment
- Market outlook
- Implementation timeline

Make the report professional and suitable for stakeholders.`

		output = []*schema.Message{
			schema.SystemMessage(systemPrompt),
		}

		reportPrompt := fmt.Sprintf("Generate a comprehensive trading report for %s based on all the analysis and decisions made.", state.CurrentSymbol)
		output = append(output, schema.UserMessage(reportPrompt))

		return nil
	})
	return output, err
}

func NewReporterNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()

	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadReporterMessages))
	_ = g.AddChatModelNode("agent", ChatModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(reporterRouter))

	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)

	return g
}
