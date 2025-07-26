package risk_mgmt

import (
	"context"

	"github.com/cloudwego/eino/compose"
	"github.com/dyike/CortexGo/internal/agents"
)

func NewRiskJudgeNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()
	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(agents.SimpleLoader("You are the final risk judge who makes ultimate risk management decisions.")))
	_ = g.AddChatModelNode("agent", agents.ChatModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(agents.SimpleRouter(compose.END)))
	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)
	return g
}

