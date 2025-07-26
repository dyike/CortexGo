package eino

import (
	"context"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/consts"
)

func coordinatorRouter(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*TradingState](ctx, func(_ context.Context, state *TradingState) error {
		defer func() {
			output = state.Goto
		}()

		state.Goto = compose.END
		if len(input.ToolCalls) > 0 {
			switch input.ToolCalls[0].Function.Name {
			case "start_analysis":
				state.Goto = consts.Analyst
			case "start_research":
				state.Goto = consts.Researcher
			case "make_trade":
				state.Goto = consts.Trader
			case "assess_risk":
				state.Goto = consts.RiskManager
			case "generate_report":
				state.Goto = consts.Reporter
			}
		}
		return nil
	})
	return output, nil
}

func loadCoordinatorMessages(ctx context.Context, name string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*TradingState](ctx, func(_ context.Context, state *TradingState) error {
		systemPrompt := `You are a trading system coordinator. Your role is to analyze the user's request and determine which trading agents should be activated.

Available tools:
- start_analysis: Activate analyst team for market analysis
- start_research: Activate researcher for deeper market research  
- make_trade: Activate trader for executing trades
- assess_risk: Activate risk manager for risk assessment
- generate_report: Generate final trading report

Current trading context:
- Symbol: ` + state.CurrentSymbol + `
- Date: ` + state.CurrentDate.Format("2006-01-02") + `

Analyze the user's request and decide which agent should handle it next.`

		output = []*schema.Message{
			schema.SystemMessage(systemPrompt),
		}
		output = append(output, state.Messages...)
		return nil
	})
	return output, err
}

func NewCoordinatorNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()

	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadCoordinatorMessages))
	_ = g.AddChatModelNode("agent", ChatModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(coordinatorRouter))

	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)

	return g
}
