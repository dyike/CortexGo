package eino

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/consts"
	"github.com/dyike/CortexGo/pkg/models"
)

func analystRouter(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*TradingState](ctx, func(_ context.Context, state *TradingState) error {
		defer func() {
			output = state.Goto
		}()

		state.Goto = consts.Coordinator
		if len(input.ToolCalls) > 0 && input.ToolCalls[0].Function.Name == "submit_analysis" {
			argMap := map[string]interface{}{}
			_ = json.Unmarshal([]byte(input.ToolCalls[0].Function.Arguments), &argMap)
			
			report := models.AnalysisReport{
				AnalystName: "TechnicalAnalyst",
				Symbol:      state.CurrentSymbol,
				Timestamp:   state.CurrentDate,
				Analysis:    fmt.Sprintf("%v", argMap["analysis"]),
				Confidence:  0.8,
				Recommendation: fmt.Sprintf("%v", argMap["recommendation"]),
			}
			
			state.Reports = append(state.Reports, report)
			
			if next, ok := argMap["next_agent"].(string); ok && next != "" {
				switch next {
				case "researcher":
					state.Goto = consts.Researcher
				case "trader":
					state.Goto = consts.Trader
				case "risk_manager":
					state.Goto = consts.RiskManager
				case "reporter":
					state.Goto = consts.Reporter
				default:
					state.Goto = consts.Coordinator
				}
			}
		}
		return nil
	})
	return output, nil
}

func loadAnalystMessages(ctx context.Context, name string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*TradingState](ctx, func(_ context.Context, state *TradingState) error {
		systemPrompt := `You are a senior financial analyst specializing in technical and fundamental analysis.

Your responsibilities:
1. Analyze market data for the given symbol
2. Provide technical analysis insights
3. Make trading recommendations
4. Determine which agent should process next

Current context:
- Symbol: ` + state.CurrentSymbol + `
- Date: ` + state.CurrentDate.Format("2006-01-02") + `
- Market Data: Price trends, volume, technical indicators

When you complete your analysis, use the submit_analysis tool to provide:
- Detailed technical analysis
- Trading recommendation (buy/sell/hold)
- Next agent to activate (researcher/trader/risk_manager/reporter)

Focus on technical patterns, support/resistance levels, and momentum indicators.`

		output = []*schema.Message{
			schema.SystemMessage(systemPrompt),
		}
		
		if state.MarketData != nil {
			marketContext := fmt.Sprintf("Current market data for %s: Price: %.2f, Volume: %d, High: %.2f, Low: %.2f", 
				state.MarketData.Symbol, state.MarketData.Price, state.MarketData.Volume,
				state.MarketData.High, state.MarketData.Low)
			output = append(output, schema.UserMessage(marketContext))
		}
		
		return nil
	})
	return output, err
}

func NewAnalystNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()

	submitAnalysisTool := &schema.ToolInfo{
		Name: "submit_analysis",
		Desc: "Submit the completed technical analysis",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"analysis": {
				Type:     schema.String,
				Desc:     "Detailed technical analysis of the symbol",
				Required: true,
			},
			"recommendation": {
				Type:     schema.String,
				Desc:     "Trading recommendation: buy, sell, or hold",
				Required: true,
			},
			"confidence": {
				Type:     schema.Number,
				Desc:     "Confidence level in the analysis (0-1)",
				Required: true,
			},
			"next_agent": {
				Type:     schema.String,
				Desc:     "Next agent to activate: researcher, trader, risk_manager, or reporter",
				Required: true,
			},
		}),
	}

	modelWithTools, _ := ChatModel.WithTools([]*schema.ToolInfo{submitAnalysisTool})

	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadAnalystMessages))
	_ = g.AddChatModelNode("agent", modelWithTools)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(analystRouter))

	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)

	return g
}