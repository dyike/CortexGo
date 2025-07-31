package risk_mgmt

import (
	"context"

	"github.com/cloudwego/eino/compose"
)

func NewRiskyAnalystNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()
	// _ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(agents.SimpleLoader("You are a risky analyst who identifies high-risk scenarios and aggressive trading strategies.")))
	// _ = g.AddChatModelNode("agent", agents.ChatModel)
	// _ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(agents.SimpleRouter(consts.SafeAnalyst)))
	// _ = g.AddEdge(compose.START, "load")
	// _ = g.AddEdge("load", "agent")
	// _ = g.AddEdge("agent", "router")
	// _ = g.AddEdge("router", compose.END)
	return g
}
