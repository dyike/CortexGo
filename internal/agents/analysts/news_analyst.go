package analysts

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/consts"
	"github.com/dyike/CortexGo/internal/models"
	"github.com/dyike/CortexGo/internal/utils"
)

func newsAnalystRouter(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		defer func() {
			output = state.Goto
		}()

		// Mark news analyst as complete and set sequential flow
		state.NewsAnalystComplete = true
		state.Goto = consts.FundamentalsAnalyst
		
		// Handle both tool calls and direct content (matching TradingAgents logic)
		var report string
		if len(input.ToolCalls) > 0 && input.ToolCalls[0].Function.Name == "submit_news_analysis" {
			// Handle tool call - extract analysis from arguments
			argMap := map[string]interface{}{}
			_ = json.Unmarshal([]byte(input.ToolCalls[0].Function.Arguments), &argMap)

			if analysis, ok := argMap["analysis"].(string); ok {
				report = analysis
			}
		} else {
			// If no tool calls, use the message content directly (TradingAgents pattern)
			if len(input.ToolCalls) == 0 && input.Content != "" {
				report = input.Content
			}
		}
		
		// Store the news report
		if report != "" {
			state.NewsReport = report
			log.Printf("News Analyst completed analysis for %s", state.CompanyOfInterest)
		}
		
		return nil
	})
	return output, nil
}

func loadNewsAnalystMessages(ctx context.Context, name string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		// Simulate dynamic tool selection logic
		var toolNames []string
		if state.Config != nil && state.Config.OnlineTools {
			toolNames = []string{"get_global_news_openai", "get_google_news"}
		} else {
			toolNames = []string{"get_finnhub_news", "get_reddit_news", "get_google_news"}
		}
		
		// Prepare context with all dynamic variables
		context := map[string]string{
			"CompanyOfInterest": state.CompanyOfInterest,
			"TradeDate":         state.TradeDate,
			"ToolNames":         fmt.Sprintf("%v", toolNames),
			"OnlineTools":       fmt.Sprintf("%t", state.Config != nil && state.Config.OnlineTools),
		}
		
		systemPrompt, err := utils.LoadPromptWithContext("analysts/news_analyst", context)
		if err != nil {
			log.Printf("Failed to load news analyst prompt: %v", err)
			// Fallback to TradingAgents-style prompt
			systemPrompt = "You are a news researcher tasked with analyzing recent news and trends over the past week. Please write a comprehensive report of the current state of the world that is relevant for trading and macroeconomics. Look at news from EODHD, and finnhub to be comprehensive. Do not simply state the trends are mixed, provide detailed and finegrained analysis and insights that may help traders make decisions. Make sure to append a Markdown table at the end of the report to organize key points in the report, organized and easy to read."
		}

		output = []*schema.Message{
			schema.SystemMessage(systemPrompt),
		}

		// Build collaborative context message (similar to TradingAgents approach)
		contextParts := []string{
			fmt.Sprintf("Analyze recent news and corporate announcements for %s on %s", state.CompanyOfInterest, state.TradeDate),
		}
		
		// Add context from previous analysts if available
		if state.MarketReport != "" {
			contextParts = append(contextParts, fmt.Sprintf("Market Analysis Context: %s", state.MarketReport))
		}
		if state.SentimentReport != "" {
			contextParts = append(contextParts, fmt.Sprintf("Social Media Analysis Context: %s", state.SentimentReport))
		}
		
		// Join all context into a comprehensive user message
		userMessage := fmt.Sprintf("%s\n\nPlease provide comprehensive news analysis building upon the previous context.", 
			contextParts[0])
		if len(contextParts) > 1 {
			userMessage = fmt.Sprintf("%s\n\nPrevious Analysis Context:\n%s", 
				contextParts[0], 
				fmt.Sprintf("- %s", contextParts[1:]))
		}
		
		output = append(output, schema.UserMessage(userMessage))

		return nil
	})
	return output, err
}

func NewNewsAnalystNode[I, O any](ctx context.Context, chatModel model.ChatModel) *compose.Graph[I, O] {
	g := compose.NewGraph[I, O]()

	// Create a typed loader that accepts the correct input type but ignores it
	typedLoader := func(ctx context.Context, input I, opts ...any) ([]*schema.Message, error) {
		return loadNewsAnalystMessages(ctx, "", opts...)
	}

	// Create a typed router that accepts the correct message type
	typedRouter := func(ctx context.Context, input *schema.Message, opts ...any) (O, error) {
		nextNode, err := newsAnalystRouter(ctx, input, opts...)
		if err != nil {
			var zero O
			return zero, err
		}
		// Convert string to O type
		if result, ok := any(nextNode).(O); ok {
			return result, nil
		}
		var zero O
		return zero, nil
	}

	_ = g.AddLambdaNode("load", compose.InvokableLambdaWithOption(typedLoader))
	_ = g.AddChatModelNode("agent", chatModel)
	_ = g.AddLambdaNode("router", compose.InvokableLambdaWithOption(typedRouter))

	_ = g.AddEdge(compose.START, "load")
	_ = g.AddEdge("load", "agent")
	_ = g.AddEdge("agent", "router")
	_ = g.AddEdge("router", compose.END)

	return g
}
