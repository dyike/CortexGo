package agents

import (
	"context"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/consts"
)

// 简化的占位符节点，用于快速编译测试
func SimpleRouter(nextNode string) func(context.Context, *schema.Message, ...any) (string, error) {
	return func(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
		err = compose.ProcessState[*TradingState](ctx, func(_ context.Context, state *TradingState) error {
			state.Goto = nextNode
			return nil
		})
		return nextNode, nil
	}
}

func SimpleLoader(prompt string) func(context.Context, string, ...any) ([]*schema.Message, error) {
	return func(ctx context.Context, name string, opts ...any) (output []*schema.Message, err error) {
		output = []*schema.Message{
			schema.SystemMessage(prompt),
			schema.UserMessage("Proceed with analysis"),
		}
		return output, nil
	}
}

func NewRiskyAnalystNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()
	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(SimpleLoader("You are a risky analyst.")))
	_ = g.AddChatModelNode("agent", ChatModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(SimpleRouter(consts.SafeAnalyst)))
	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)
	return g
}

func NewSafeAnalystNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()
	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(SimpleLoader("You are a safe analyst.")))
	_ = g.AddChatModelNode("agent", ChatModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(SimpleRouter(consts.NeutralAnalyst)))
	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)
	return g
}

func NewNeutralAnalystNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()
	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(SimpleLoader("You are a neutral analyst.")))
	_ = g.AddChatModelNode("agent", ChatModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(SimpleRouter(consts.RiskJudge)))
	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)
	return g
}

func NewRiskJudgeNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()
	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(SimpleLoader("You are the final risk judge.")))
	_ = g.AddChatModelNode("agent", ChatModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(SimpleRouter(compose.END)))
	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)
	return g
}
