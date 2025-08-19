package analysts

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	t_utils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/consts"
	"github.com/dyike/CortexGo/internal/config"
	"github.com/dyike/CortexGo/internal/dataflows"
	"github.com/dyike/CortexGo/internal/models"
	"github.com/dyike/CortexGo/internal/utils"
)

// MarketAnalystState represents the state for market analyst ReAct agent
type MarketAnalystState struct {
	Messages                 []*schema.Message
	ReturnDirectlyToolCallID string
	MarketData              []*models.MarketData
	AnalysisComplete        bool
}

var registerMarketStateOnce sync.Once

// ReAct Agent constants
const (
	nodeKeyMarketTools = "market_tools"
	nodeKeyMarketModel = "market_chat"
)

// Router function for backwards compatibility
func router(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		defer func() {
			output = state.Goto
		}()
		// 处理工具返回的消息
		if input != nil {
			if input.Role == schema.Tool && input.ToolName == "get_market_data" {
				marketData := struct {
					Data []*models.MarketData `json:"data"`
				}{}
				_ = json.Unmarshal([]byte(input.Content), &marketData)
				fmt.Println("get_marked_data data: ", marketData.Data)
				state.MarketData = marketData.Data
				state.Goto = consts.MarketAnalyst
				return nil
			}
		}
		// Mark market analyst as complete and set sequential flow
		state.Goto = consts.NewsAnalyst
		return nil
	})
	return output, nil
}

func loadMsg(ctx context.Context, name string, opts ...any) ([]*schema.Message, error) {
	var (
		output []*schema.Message
		fErr   error
	)
	err := compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
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
			// 添加模板所需的新变量
			"tool_names":     strings.Join(getMarketDataTools(), ","), // 需要实现工具名称获取函数
			"current_date":   time.Now().Format("2006-01-02"),
			"ticker":         state.CompanyOfInterest,
			"system_message": systemPrompt,
		}

		output, fErr = promptTemp.Format(ctx, context)
		if fErr != nil {
			log.Printf("MarkteAnalyst failed to format prompt: %v", fErr)
			return fErr
		}

		marketDataStr := ""
		for _, data := range state.MarketData {
			marketContext := fmt.Sprintf(
				"Symbol(%s) market data on %s: Volume: %d, High: %.2f, Low: %.2f, Open: %.2f, Close: %.2f",
				data.Symbol, data.Date, data.Volume, data.High, data.Low, data.Open, data.Close)
			if marketDataStr != "" {
				marketDataStr += "\n"
			}
			marketDataStr += marketContext
		}
		if marketDataStr != "" {
			output = append(output, schema.UserMessage(marketDataStr))
		}
		return nil
	})
	return output, err
}

// ReAct pattern stream tool call checker
func reactStreamToolCallChecker(_ context.Context, sr *schema.StreamReader[*schema.Message]) (bool, error) {
	defer sr.Close()

	for {
		msg, err := sr.Recv()
		if err == io.EOF {
			return false, nil
		}
		if err != nil {
			return false, err
		}

		if len(msg.ToolCalls) > 0 {
			return true, nil
		}

		if len(msg.Content) == 0 {
			continue
		}

		return false, nil
	}
}

// Legacy function for backwards compatibility
func shoudContinueMarket(ctx context.Context, input *schema.Message) (next string, err error) {
	_ = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, st *models.TradingState) error {
		return nil
	})

	// Check if the assistant message contains tool calls
	if input.Role == schema.Assistant && len(input.ToolCalls) > 0 {
		return "tools", nil
	}

	// If it's a tool response, continue to agent
	if input.Role == schema.Tool {
		return "router", nil
	}

	// Default case - end the flow
	return "router", nil
}

// MarketAnalyst ReAct Agent
type MarketAnalyst struct {
	runnable         compose.Runnable[[]*schema.Message, *schema.Message]
	graph            *compose.Graph[[]*schema.Message, *schema.Message]
	graphAddNodeOpts []compose.GraphAddNodeOpt
}

// NewMarketAnalyst creates a ReAct-style market analyst
func NewMarketAnalyst(ctx context.Context, cfg *config.Config, companyOfInterest, tradeDate string) (*MarketAnalyst, error) {
	var err error
	
	// Register state type once
	registerMarketStateOnce.Do(func() {
		err = compose.RegisterSerializableType[MarketAnalystState]("_cortex_market_analyst_state")
	})
	if err != nil {
		return nil, err
	}
	
	// Create DeepSeek model
	chatModel, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		APIKey:    cfg.DeepSeekAPIKey,
		Model:     "deepseek-chat",
		MaxTokens: 2000,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create DeepSeek model: %v", err)
	}
	
	// Create Longport client for real market data
	longportClient, err := dataflows.NewLongportClient(cfg)
	if err != nil {
		log.Printf("Failed to create Longport client, using mock data: %v", err)
		longportClient = nil
	}
	
	// Create market data tool with real Longport integration
	getMarketDataTool := t_utils.NewTool(
		&schema.ToolInfo{
			Name: "get_market_data",
			Desc: "Get market data for a specific symbol and date range",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"symbol": {
					Type:     "string",
					Desc:     "The stock symbol",
					Required: true,
				},
				"count": {
					Type:     "integer",
					Desc:     "Number of days to retrieve (default: 30)",
					Required: false,
				},
			}),
		},
		func(ctx context.Context, input *schema.ToolCall) (*GetMarketDataOutput, error) {
			// Parse parameters from Arguments map
			argsBytes, err := json.Marshal(input.Function.Arguments)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal arguments: %v", err)
			}
			
			var args struct {
				Symbol string `json:"symbol"`
				Count  int    `json:"count"`
			}
			if err := json.Unmarshal(argsBytes, &args); err != nil {
				return nil, fmt.Errorf("failed to parse arguments: %v", err)
			}
			
			if args.Symbol == "" {
				return nil, fmt.Errorf("symbol parameter is required")
			}
			
			count := args.Count
			if count <= 0 {
				count = 30 // default
			}
			
			// Try to get real market data from Longport
			if longportClient != nil {
				sticks, err := longportClient.GetSticksWithDay(ctx, args.Symbol, count)
				if err == nil && len(sticks) > 0 {
					marketData := make([]*models.MarketData, 0, len(sticks))
					for _, stick := range sticks {
						// Convert Unix timestamp to date string
						date := time.Unix(stick.Timestamp, 0).Format("2006-01-02")
						
						// Convert decimal values to float64
						open, _ := stick.Open.Float64()
						high, _ := stick.High.Float64() 
						low, _ := stick.Low.Float64()
						close, _ := stick.Close.Float64()
						
						marketData = append(marketData, &models.MarketData{
							Symbol: args.Symbol,
							Date:   date,
							Open:   open,
							High:   high,
							Low:    low,
							Close:  close,
							Volume: stick.Volume,
						})
					}
					return &GetMarketDataOutput{Data: marketData}, nil
				}
				log.Printf("Failed to get real market data for %s: %v", args.Symbol, err)
			}
			
			// Fallback to mock data
			return &GetMarketDataOutput{
				Data: []*models.MarketData{
					{
						Symbol: args.Symbol,
						Date:   tradeDate,
						Open:   100.0,
						High:   101.0,
						Low:    99.0,
						Close:  100.5,
						Volume: int64(1000000),
					},
				},
			}, nil
		},
	)
	
	// Get tool info
	info, err := getMarketDataTool.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get market data tool info: %v", err)
	}
	
	// Bind tools to model
	chatModelWithTool, err := chatModel.WithTools([]*schema.ToolInfo{info})
	if err != nil {
		return nil, fmt.Errorf("failed to bind market data tool: %v", err)
	}
	
	// Create tools node
	toolsNode, err := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
		Tools: []tool.BaseTool{getMarketDataTool},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create tools node: %v", err)
	}
	
	// Create message modifier for system prompts
	messageModifier := func(ctx context.Context, input []*schema.Message) []*schema.Message {
		systemPrompt, _ := utils.LoadPrompt("analysts/market_analyst")
		
		systemTpl := `You are a helpful AI assistant, collaborating with other assistants.
Use the provided tools to progress towards answering the question.
If you are unable to fully answer, that's OK; another assistant with different tools
will help where you left off. Execute what you can to make progress.
If you or any other assistant has the FINAL TRANSACTION PROPOSAL: **BUY/HOLD/SELL** or deliverable,
prefix your response with FINAL TRANSACTION PROPOSAL: **BUY/HOLD/SELL** so the team knows to stop.

You have access to the following tools:
- get_market_data: Get market data for a specific symbol and date range.

{system_message}

For your reference, the current date is {current_date}. The company we want to look at is {ticker}`
		
		context := map[string]any{
			"current_date":   time.Now().Format("2006-01-02"),
			"ticker":         companyOfInterest,
			"system_message": systemPrompt,
		}
		
		promptTemp := prompt.FromMessages(schema.FString,
			schema.SystemMessage(systemTpl),
		)
		
		systemMessages, err := promptTemp.Format(ctx, context)
		if err != nil {
			log.Printf("MarketAnalyst failed to format system prompt: %v", err)
			return input
		}
		
		res := make([]*schema.Message, 0, len(input)+len(systemMessages))
		res = append(res, systemMessages...)
		res = append(res, input...)
		return res
	}
	
	// Create graph with local state
	graph := compose.NewGraph[[]*schema.Message, *schema.Message](
		compose.WithGenLocalState(func(ctx context.Context) *MarketAnalystState {
			return &MarketAnalystState{
				Messages:   make([]*schema.Message, 0, 15),
				MarketData: make([]*models.MarketData, 0),
			}
		}),
	)
	
	// Model pre-handler to manage messages and state
	modelPreHandler := func(ctx context.Context, input []*schema.Message, state *MarketAnalystState) ([]*schema.Message, error) {
		// For the first call, we need to apply the system message and store the initial input
		if len(state.Messages) == 0 {
			// Store the initial input messages in state (without system message)
			state.Messages = append(state.Messages, input...)
			
			// Apply message modifier to create the full message list with system prompt
			if messageModifier != nil {
				return messageModifier(ctx, input), nil
			}
			return input, nil
		}
		
		// For subsequent calls, send all accumulated messages
		// We need to apply the system message only once at the beginning
		if messageModifier != nil {
			// Extract user messages from state (skip system message)
			userMessages := make([]*schema.Message, 0)
			for _, msg := range state.Messages {
				if msg.Role != schema.System {
					userMessages = append(userMessages, msg)
				}
			}
			return messageModifier(ctx, userMessages), nil
		}
		
		return state.Messages, nil
	}
	
	// Model post-handler to update conversation state
	modelPostHandler := func(ctx context.Context, input *schema.Message, state *MarketAnalystState) (*schema.Message, error) {
		// Add the model's response to conversation history
		if input != nil {
			state.Messages = append(state.Messages, input)
		}
		return input, nil
	}
	
	// Add chat model node with both pre and post handlers
	err = graph.AddChatModelNode(nodeKeyMarketModel, chatModelWithTool, 
		compose.WithStatePreHandler(modelPreHandler),
		compose.WithStatePostHandler(modelPostHandler),
		compose.WithNodeName("MarketChatModel"))
	if err != nil {
		return nil, err
	}
	
	// Tools post-handler to process tool responses and update conversation state
	toolsPostHandler := func(ctx context.Context, input []*schema.Message, state *MarketAnalystState) ([]*schema.Message, error) {
		// Update the state with all tool response messages
		for _, msg := range input {
			if msg.Role == schema.Tool && msg.ToolName == "get_market_data" {
				var marketDataOutput GetMarketDataOutput
				if err := json.Unmarshal([]byte(msg.Content), &marketDataOutput); err == nil {
					state.MarketData = append(state.MarketData, marketDataOutput.Data...)
					log.Printf("Market data updated: %d records for analysis", len(marketDataOutput.Data))
				} else {
					log.Printf("Failed to parse market data response: %v", err)
				}
			}
		}
		
		// Append tool response messages to conversation history
		state.Messages = append(state.Messages, input...)
		
		return input, nil
	}
	
	// Add tools node with post-handler to manage conversation state
	err = graph.AddToolsNode(nodeKeyMarketTools, toolsNode,
		compose.WithStatePostHandler(toolsPostHandler),
		compose.WithNodeName("MarketTools"))
	if err != nil {
		return nil, err
	}
	
	// Add START -> model edge
	err = graph.AddEdge(compose.START, nodeKeyMarketModel)
	if err != nil {
		return nil, err
	}
	
	// Branch condition after model
	modelBranchCondition := func(ctx context.Context, sr *schema.StreamReader[*schema.Message]) (endNode string, err error) {
		if isToolCall, err := reactStreamToolCallChecker(ctx, sr); err != nil {
			return "", err
		} else if isToolCall {
			return nodeKeyMarketTools, nil
		}
		return compose.END, nil
	}
	
	// Add branch after model
	err = graph.AddBranch(nodeKeyMarketModel, compose.NewStreamGraphBranch(
		modelBranchCondition,
		map[string]bool{nodeKeyMarketTools: true, compose.END: true}))
	if err != nil {
		return nil, err
	}
	
	// Add direct edge from tools back to model (standard ReAct pattern)
	err = graph.AddEdge(nodeKeyMarketTools, nodeKeyMarketModel)
	if err != nil {
		return nil, err
	}
	
	// Compile graph
	compileOpts := []compose.GraphCompileOption{
		compose.WithMaxRunSteps(12),
		compose.WithNodeTriggerMode(compose.AnyPredecessor),
		compose.WithGraphName("MarketAnalyst"),
	}
	
	runnable, err := graph.Compile(ctx, compileOpts...)
	if err != nil {
		return nil, err
	}
	
	return &MarketAnalyst{
		runnable:         runnable,
		graph:            graph,
		graphAddNodeOpts: []compose.GraphAddNodeOpt{compose.WithGraphCompileOptions(compileOpts...)},
	}, nil
}

// Generate generates market analysis response using ReAct pattern
func (ma *MarketAnalyst) Generate(ctx context.Context, input []*schema.Message, opts ...agent.AgentOption) (*schema.Message, error) {
	return ma.runnable.Invoke(ctx, input, agent.GetComposeOptions(opts...)...)
}

// Stream calls market analyst and returns stream response
func (ma *MarketAnalyst) Stream(ctx context.Context, input []*schema.Message, opts ...agent.AgentOption) (*schema.StreamReader[*schema.Message], error) {
	return ma.runnable.Stream(ctx, input, agent.GetComposeOptions(opts...)...)
}

// ExportGraph exports the underlying graph for composition
func (ma *MarketAnalyst) ExportGraph() (compose.AnyGraph, []compose.GraphAddNodeOpt) {
	return ma.graph, ma.graphAddNodeOpts
}

// NewMarketAnalystNode creates a legacy graph node for backwards compatibility
func NewMarketAnalystNode[I, O any](ctx context.Context, cfg *config.Config) *compose.Graph[I, O] {
	// For backwards compatibility, create legacy graph structure
	// 创建 deepseek 模型
	apiKey := cfg.DeepSeekAPIKey
	chatModel, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		APIKey:    apiKey,
		Model:     "deepseek-chat",
		MaxTokens: 2000,
	})
	if err != nil {
		log.Printf("MarkteAnalyst failed to create DeepSeek model: %v", err)
	}
	getMarketDataTool := t_utils.NewTool(
		&schema.ToolInfo{
			Name: "get_market_data",
			Desc: "Get market data for a specific symbol and date range",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"symbol": {
					Type:     "string",
					Desc:     "The stock symbol",
					Required: true,
				},
				"start_date": {
					Type:     "string",
					Desc:     "The start date in YYYY-MM-DD format",
					Required: true,
				},
				"end_date": {
					Type:     "string",
					Desc:     "The end date in YYYY-MM-DD format",
					Required: true,
				},
			}),
		},
		func(ctx context.Context, input *schema.ToolCall) (*GetMarketDataOutput, error) {
			return &GetMarketDataOutput{
				Data: []*models.MarketData{
					{
						Symbol: "UI",
						Date:   "2025-08-06",
						Open:   100,
						High:   101,
						Low:    99,
						Close:  100.5,
						Volume: 1000000,
					},
				},
			}, nil
		},
	)

	info, err := getMarketDataTool.Info(ctx)
	if err != nil {
		log.Printf("MarkteAnalyst failed to get market data tool info: %v", err)
		return nil
	}

	chatModelWithTool, _ := chatModel.WithTools([]*schema.ToolInfo{info})
	if err != nil {
		log.Printf("MarkteAnalyst failed to bind market data tool: %v", err)
		return nil
	}

	toolsNode, err := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
		Tools: []tool.BaseTool{getMarketDataTool},
	})
	if err != nil {
		log.Printf("NewToolNode failed, err=%v", err)
		return nil
	}

	g := compose.NewGraph[I, O]()
	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadMsg))
	_ = g.AddChatModelNode("agent", chatModelWithTool)
	_ = g.AddToolsNode("tools", toolsNode)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(router))

	outMap := map[string]bool{
		"tools":  true,
		"router": true,
	}
	_ = g.AddBranch("agent", compose.NewGraphBranch(shoudContinueMarket, outMap))
	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("tools", "agent")
	_ = g.AddEdge("router", compose.END)

	return g
}

func getMarketDataTools() []string {
	return []string{"get_market_data"}
}

type GetMarketDataInput struct {
	Symbol    string `json:"symbol"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type GetMarketDataOutput struct {
	Data []*models.MarketData `json:"data"`
}
