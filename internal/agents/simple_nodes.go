package agents

// import (
// 	"context"

// 	"github.com/cloudwego/eino/compose"
// 	"github.com/dyike/CortexGo/consts"
// )

// // 简化版本的节点实现，替换有问题的旧节点
// func NewRiskyAnalystNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
// 	g := compose.NewGraph[I, O]()
// 	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(SimpleLoader("You are a risky analyst.")))
// 	_ = g.AddChatModelNode("agent", ChatModel)
// 	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(SimpleRouter(consts.SafeAnalyst)))
// 	_ = g.AddEdge(compose.START, "load")
// 	_ = g.AddEdge("load", "agent")
// 	_ = g.AddEdge("agent", "router")
// 	_ = g.AddEdge("router", compose.END)
// 	return g
// }

// func NewSafeAnalystNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
// 	g := compose.NewGraph[I, O]()
// 	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(SimpleLoader("You are a safe analyst.")))
// 	_ = g.AddChatModelNode("agent", ChatModel)
// 	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(SimpleRouter(consts.NeutralAnalyst)))
// 	_ = g.AddEdge(compose.START, "load")
// 	_ = g.AddEdge("load", "agent")
// 	_ = g.AddEdge("agent", "router")
// 	_ = g.AddEdge("router", compose.END)
// 	return g
// }

// func NewNeutralAnalystNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
// 	g := compose.NewGraph[I, O]()
// 	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(SimpleLoader("You are a neutral analyst.")))
// 	_ = g.AddChatModelNode("agent", ChatModel)
// 	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(SimpleRouter(consts.RiskJudge)))
// 	_ = g.AddEdge(compose.START, "load")
// 	_ = g.AddEdge("load", "agent")
// 	_ = g.AddEdge("agent", "router")
// 	_ = g.AddEdge("router", compose.END)
// 	return g
// }

// func NewRiskJudgeNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
// 	g := compose.NewGraph[I, O]()
// 	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(SimpleLoader("You are the final risk judge.")))
// 	_ = g.AddChatModelNode("agent", ChatModel)
// 	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(SimpleRouter(compose.END)))
// 	_ = g.AddEdge(compose.START, "load")
// 	_ = g.AddEdge("load", "agent")
// 	_ = g.AddEdge("agent", "router")
// 	_ = g.AddEdge("router", compose.END)
// 	return g
// }

// func NewTraderNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
// 	g := compose.NewGraph[I, O]()
// 	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(SimpleLoader("You are a trader. Make trading decisions.")))
// 	_ = g.AddChatModelNode("agent", ChatModel)
// 	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(SimpleRouter(consts.RiskyAnalyst)))
// 	_ = g.AddEdge(compose.START, "load")
// 	_ = g.AddEdge("load", "agent")
// 	_ = g.AddEdge("agent", "router")
// 	_ = g.AddEdge("router", compose.END)
// 	return g
// }

// func NewAnalystNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
// 	g := compose.NewGraph[I, O]()
// 	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(SimpleLoader("You are an analyst.")))
// 	_ = g.AddChatModelNode("agent", ChatModel)
// 	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(SimpleRouter(consts.Researcher)))
// 	_ = g.AddEdge(compose.START, "load")
// 	_ = g.AddEdge("load", "agent")
// 	_ = g.AddEdge("agent", "router")
// 	_ = g.AddEdge("router", compose.END)
// 	return g
// }

// func NewResearcherNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
// 	g := compose.NewGraph[I, O]()
// 	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(SimpleLoader("You are a researcher.")))
// 	_ = g.AddChatModelNode("agent", ChatModel)
// 	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(SimpleRouter(consts.Trader)))
// 	_ = g.AddEdge(compose.START, "load")
// 	_ = g.AddEdge("load", "agent")
// 	_ = g.AddEdge("agent", "router")
// 	_ = g.AddEdge("router", compose.END)
// 	return g
// }

// func NewRiskManagerNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
// 	g := compose.NewGraph[I, O]()
// 	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(SimpleLoader("You are a risk manager.")))
// 	_ = g.AddChatModelNode("agent", ChatModel)
// 	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(SimpleRouter(consts.Reporter)))
// 	_ = g.AddEdge(compose.START, "load")
// 	_ = g.AddEdge("load", "agent")
// 	_ = g.AddEdge("agent", "router")
// 	_ = g.AddEdge("router", compose.END)
// 	return g
// }

// func NewReporterNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
// 	g := compose.NewGraph[I, O]()
// 	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(SimpleLoader("You are a reporter.")))
// 	_ = g.AddChatModelNode("agent", ChatModel)
// 	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(SimpleRouter(compose.END)))
// 	_ = g.AddEdge(compose.START, "load")
// 	_ = g.AddEdge("load", "agent")
// 	_ = g.AddEdge("agent", "router")
// 	_ = g.AddEdge("router", compose.END)
// 	return g
// }

// func NewCoordinatorNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
// 	g := compose.NewGraph[I, O]()
// 	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(SimpleLoader("You are a coordinator.")))
// 	_ = g.AddChatModelNode("agent", ChatModel)
// 	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(SimpleRouter(consts.MarketAnalyst)))
// 	_ = g.AddEdge(compose.START, "load")
// 	_ = g.AddEdge("load", "agent")
// 	_ = g.AddEdge("agent", "router")
// 	_ = g.AddEdge("router", compose.END)
// 	return g
// }
