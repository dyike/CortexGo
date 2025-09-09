package risk_mgmt

import (
	"context"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/config"
	"github.com/dyike/CortexGo/internal/agents"
)

func NewSafeAnalystNode[I, O any](ctx context.Context, cfg *config.Config) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()
	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadSafeMsg))
	_ = g.AddChatModelNode("agent", agents.ChatModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(safeRouter))
	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)
	return g
}

func loadSafeMsg(ctx context.Context, name string, opts ...any) (output []*schema.Message, err error) {
	return output, err
}

func safeRouter(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	return output, err
}
