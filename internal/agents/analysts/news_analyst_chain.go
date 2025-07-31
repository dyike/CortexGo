package analysts

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/dyike/CortexGo/internal/models"
	"github.com/dyike/CortexGo/internal/utils"
)

// NewsAnalystChainInput represents the input for the news analyst chain
type NewsAnalystChainInput struct {
	State *models.TradingState `json:"state"`
}

// NewsAnalystChainOutput represents the output from the news analyst chain
type NewsAnalystChainOutput struct {
	NewsReport string                `json:"news_report"`
	Messages   []*schema.Message     `json:"messages"`
	State      *models.TradingState  `json:"state"`
}

// createPromptMessages creates the prompt messages for the news analyst
func createPromptMessages(ctx context.Context, input NewsAnalystChainInput, opts ...any) ([]*schema.Message, error) {
	state := input.State
	
	// Determine tools based on configuration (similar to Python version)
	var toolNames []string
	if state.Config != nil && state.Config.OnlineTools {
		toolNames = []string{"get_global_news_openai", "get_google_news"}
	} else {
		toolNames = []string{"get_finnhub_news", "get_reddit_news", "get_google_news"}
	}
	
	// System message matching Python version
	systemMessage := "You are a news researcher tasked with analyzing recent news and trends over the past week. Please write a comprehensive report of the current state of the world that is relevant for trading and macroeconomics. Look at news from EODHD, and finnhub to be comprehensive. Do not simply state the trends are mixed, provide detailed and finegrained analysis and insights that may help traders make decisions. Make sure to append a Markdown table at the end of the report to organize key points in the report, organized and easy to read."
	
	// Load system prompt with context variables
	context := map[string]string{
		"CompanyOfInterest": state.CompanyOfInterest,
		"TradeDate":         state.TradeDate,
		"ToolNames":         fmt.Sprintf("%v", toolNames),
		"OnlineTools":       fmt.Sprintf("%t", state.Config != nil && state.Config.OnlineTools),
	}
	
	systemPrompt, err := utils.LoadPromptWithContext("analysts/news_analyst", context)
	if err != nil {
		log.Printf("Failed to load news analyst prompt: %v", err)
		systemPrompt = systemMessage // Fallback to default
	}
	
	// Create collaborative system prompt (matching Python ChatPromptTemplate logic)
	collaborativePrompt := fmt.Sprintf(
		"You are a helpful AI assistant, collaborating with other assistants. "+
		"Use the provided tools to progress towards answering the question. "+
		"If you are unable to fully answer, that's OK; another assistant with different tools "+
		"will help where you left off. Execute what you can to make progress. "+
		"If you or any other assistant has the FINAL TRANSACTION PROPOSAL: **BUY/HOLD/SELL** or deliverable, "+
		"prefix your response with FINAL TRANSACTION PROPOSAL: **BUY/HOLD/SELL** so the team knows to stop. "+
		"You have access to the following tools: %s.\n%s"+
		"For your reference, the current date is %s. We are looking at the company %s",
		fmt.Sprintf("%v", toolNames), systemPrompt, state.TradeDate, state.CompanyOfInterest,
	)
	
	messages := []*schema.Message{
		schema.SystemMessage(collaborativePrompt),
	}
	
	// Add existing messages from state
	messages = append(messages, state.Messages...)
	
	return messages, nil
}

// processLLMResponse processes the response from the LLM and extracts the report
func processLLMResponse(ctx context.Context, response *schema.Message, opts ...any) (NewsAnalystChainOutput, error) {
	var output NewsAnalystChainOutput
	
	err := compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, state *models.TradingState) error {
		report := ""
		
		// Handle both tool calls and direct content (matching Python logic)
		if len(response.ToolCalls) == 0 {
			// No tool calls, use content directly
			report = response.Content
		} else {
			// Process tool calls if needed
			for _, toolCall := range response.ToolCalls {
				if toolCall.Function.Name == "submit_news_analysis" {
					argMap := map[string]interface{}{}
					if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &argMap); err == nil {
						if analysis, ok := argMap["analysis"].(string); ok {
							report = analysis
						}
					}
				}
			}
		}
		
		// Update state with the news report
		state.NewsReport = report
		state.NewsAnalystComplete = true
		
		// Prepare output
		output = NewsAnalystChainOutput{
			NewsReport: report,
			Messages:   []*schema.Message{response},
			State:      state,
		}
		
		if report != "" {
			log.Printf("News Analyst completed analysis for %s", state.CompanyOfInterest)
		}
		
		return nil
	})
	
	return output, err
}

// CreateNewsAnalystChain creates a news analyst chain that mimics the Python function behavior
func CreateNewsAnalystChain(ctx context.Context, chatModel model.ChatModel) (compose.Runnable[NewsAnalystChainInput, NewsAnalystChainOutput], error) {
	chain := compose.NewChain[NewsAnalystChainInput, NewsAnalystChainOutput]()
	
	// Step 1: Create prompt messages (equivalent to prompt template creation in Python)
	chain.AppendLambda(compose.InvokableLambdaWithOption(createPromptMessages))
	
	// Step 2: Invoke the chat model with tools bound (equivalent to chain.invoke() in Python)
	// Note: Tool binding would need to be implemented based on your tool system
	chain.AppendChatModel(chatModel)
	
	// Step 3: Process the response and extract the report
	chain.AppendLambda(compose.InvokableLambdaWithOption(processLLMResponse))
	
	// Compile the chain to get a Runnable
	runnable, err := chain.Compile(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to compile news analyst chain: %v", err)
	}
	
	return runnable, nil
}

// CreateNewsAnalystNode creates a news analyst node using the chain pattern
// This provides the same interface as the existing graph-based implementation
func CreateNewsAnalystNode[I, O any](ctx context.Context, chatModel model.ChatModel) (*compose.Graph[I, O], error) {
	// Create and compile the chain
	newsChain, err := CreateNewsAnalystChain(ctx, chatModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create news analyst chain: %v", err)
	}
	
	// Wrap the chain in a graph node for compatibility
	g := compose.NewGraph[I, O]()
	
	// Create adapter functions to match the expected input/output types
	chainAdapter := func(ctx context.Context, input I, opts ...any) (O, error) {
		// Extract state from input (assumes I contains or is a TradingState)
		var state *models.TradingState
		if tradingState, ok := any(input).(*models.TradingState); ok {
			state = tradingState
		} else {
			// Try to extract from context or opts
			err := compose.ProcessState[*models.TradingState](ctx, func(_ context.Context, s *models.TradingState) error {
				state = s
				return nil
			})
			if err != nil {
				var zero O
				return zero, fmt.Errorf("failed to extract trading state: %v", err)
			}
		}
		
		chainInput := NewsAnalystChainInput{State: state}
		
		// Invoke the compiled chain
		result, err := newsChain.Invoke(ctx, chainInput)
		if err != nil {
			var zero O
			return zero, err
		}
		
		// Convert result to expected output type
		if output, ok := any(result).(O); ok {
			return output, nil
		}
		
		var zero O
		return zero, fmt.Errorf("failed to convert chain output to expected type")
	}
	
	_ = g.AddLambdaNode("news_analyst_chain", compose.InvokableLambdaWithOption(chainAdapter))
	_ = g.AddEdge(compose.START, "news_analyst_chain")
	_ = g.AddEdge("news_analyst_chain", compose.END)
	
	return g, nil
}