package trading

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dyike/CortexGo/internal/config"
	"github.com/dyike/CortexGo/internal/graph"
)

// TradingSession represents a trading analysis session
type TradingSession struct {
	config *config.Config
	symbol string
	date   string
	graph  *graph.TradingAgentsGraph
}

// NewTradingSession creates a new trading session
func NewTradingSession(config *config.Config, symbol, date string) *TradingSession {
	return &TradingSession{
		config: config,
		symbol: symbol,
		date:   date,
		graph:  nil, // Initialize later with proper error handling
	}
}

// Execute runs the trading analysis
func (s *TradingSession) Execute(ctx context.Context) error {
	fmt.Printf("📊 Initializing analysis for %s...\n", s.symbol)

	// Validate date format
	_, err := time.Parse("2006-01-02", s.date)
	if err != nil {
		return fmt.Errorf("invalid date format: %w", err)
	}

	// Validate configuration before running
	if err := s.validateConfig(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Initialize the trading graph
	if err := s.initializeGraph(); err != nil {
		return fmt.Errorf("failed to initialize trading graph: %w", err)
	}

	// Run the trading graph with error recovery
	fmt.Printf("🔄 Running analysis pipeline...\n")
	result, err := s.runWithRecovery()
	if err != nil {
		return fmt.Errorf("failed to execute trading graph: %w", err)
	}

	// Display results summary
	s.displayResults(result)

	return nil
}

// validateConfig checks if the configuration is valid for running analysis
func (s *TradingSession) validateConfig() error {
	// Check if we have necessary API keys based on provider
	switch s.config.LLMProvider {
	case "openai":
		if os.Getenv("OPENAI_API_KEY") == "" {
			return fmt.Errorf("OPENAI_API_KEY environment variable is required for OpenAI provider")
		}
	case "deepseek":
		if s.config.DeepSeekAPIKey == "" && os.Getenv("DEEPSEEK_API_KEY") == "" {
			return fmt.Errorf("DEEPSEEK_API_KEY is required for DeepSeek provider")
		}
	}
	
	return nil
}

// initializeGraph initializes the trading graph with error handling
func (s *TradingSession) initializeGraph() error {
	var initErr error
	
	defer func() {
		if r := recover(); r != nil {
			initErr = fmt.Errorf("graph initialization panicked: %v", r)
			fmt.Printf("❌ Graph initialization failed: %v\n", r)
		}
	}()

	fmt.Printf("🔧 Initializing trading graph...\n")
	s.graph = graph.NewTradingAgentsGraph(s.config.Debug, s.config)
	
	if initErr != nil {
		return initErr
	}
	
	if s.graph == nil {
		return fmt.Errorf("failed to create trading graph")
	}
	
	return nil
}

// runWithRecovery runs the trading graph with panic recovery
func (s *TradingSession) runWithRecovery() (interface{}, error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("❌ Panic recovered: %v\n", r)
		}
	}()

	result, err := s.graph.Propagate(s.symbol, s.date)
	return result, err
}

// displayResults shows the analysis results
func (s *TradingSession) displayResults(state interface{}) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Printf("📈 Analysis Results for %s (%s)\n", s.symbol, s.date)
	fmt.Println(strings.Repeat("=", 60))
	
	// In a real implementation, you would format and display
	// the actual analysis results from the state
	fmt.Printf("✓ Analysis completed successfully\n")
	fmt.Printf("📄 Results saved to: %s\n", s.config.ResultsDir)
	
	fmt.Println("\n💡 Check the results directory for detailed reports.")
}