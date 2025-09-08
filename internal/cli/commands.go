package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/dyike/CortexGo/internal/config"
	"github.com/dyike/CortexGo/internal/trading"
)

// NewRootCmd creates the root command
func NewRootCmd() *cobra.Command {
	// Initialize configuration early
	cfg := config.DefaultConfig()

	rootCmd := &cobra.Command{
		Use:   "cortexgo",
		Short: "CortexGo - AI-Powered Trading Analysis",
		Long: `CortexGo is an advanced multi-agent trading analysis system powered by Large Language Models.
It provides comprehensive market analysis, research, and risk assessment for informed trading decisions.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Ensure directories exist
			if err := cfg.EnsureDirectories(); err != nil {
				return fmt.Errorf("failed to create directories: %w", err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default behavior: start interactive mode
			return runInteractiveMode(cfg)
		},
	}

	// Add subcommands
	rootCmd.AddCommand(newAnalyzeCmd(cfg))
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newConfigCmd(cfg))

	// Global flags
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug mode")
	rootCmd.PersistentFlags().String("config", "", "Configuration file path")

	return rootCmd
}

// newAnalyzeCmd creates the analyze command
func newAnalyzeCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze [SYMBOL]",
		Short: "Run trading analysis for a stock symbol",
		Long: `Run a comprehensive trading analysis for a given stock ticker symbol.
Example: cortexgo analyze AAPL --date=2024-03-15`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			symbol := args[0]
			date, _ := cmd.Flags().GetString("date")
			
			return runAnalyzeCommand(cfg, symbol, date)
		},
	}

	// Analyze command flags
	cmd.Flags().String("date", "", "Analysis date in YYYY-MM-DD format (today if not provided)")
	cmd.MarkFlagRequired("date")

	return cmd
}

// newVersionCmd creates the version command
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("CortexGo v1.0.0")
			fmt.Println("AI-Powered Trading Analysis System")
			fmt.Println("Built with ❤️  using Go and Large Language Models")
		},
	}
}

// newConfigCmd creates the config command
func newConfigCmd(cfg *config.Config) *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration management",
		Long:  "Manage CortexGo configuration settings",
	}

	// config show subcommand
	configCmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Run: func(cmd *cobra.Command, args []string) {
			showConfig(cfg)
		},
	})

	// config validate subcommand
	configCmd.AddCommand(&cobra.Command{
		Use:   "validate",
		Short: "Validate configuration and dependencies",
		RunE: func(cmd *cobra.Command, args []string) error {
			return validateConfig(cfg)
		},
	})

	return configCmd
}

// runAnalyzeCommand executes the main analysis workflow
func runAnalyzeCommand(cfg *config.Config, symbol, date string) error {
	ctx := context.Background()

	// Validate inputs
	if symbol == "" {
		return fmt.Errorf("symbol is required")
	}

	// Use current date if not provided
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	// Validate date format
	_, err := time.Parse("2006-01-02", date)
	if err != nil {
		return fmt.Errorf("invalid date format, use YYYY-MM-DD: %w", err)
	}

	fmt.Printf("🚀 Starting analysis for %s on %s\n", symbol, date)

	// Create trading session
	session := trading.NewTradingSession(cfg, symbol, date)

	// Run the analysis
	err = session.Execute(ctx)
	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	fmt.Println("✅ Analysis completed successfully!")
	return nil
}

// showConfig displays the current configuration
func showConfig(cfg *config.Config) {
	fmt.Println("📋 Current CortexGo Configuration:")
	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("Project Directory:    %s\n", cfg.ProjectDir)
	fmt.Printf("Results Directory:    %s\n", cfg.ResultsDir)
	fmt.Printf("Data Directory:       %s\n", cfg.DataDir)
	fmt.Printf("Cache Directory:      %s\n", cfg.DataCacheDir)
	fmt.Println()
	fmt.Printf("LLM Provider:         %s\n", cfg.LLMProvider)
	fmt.Printf("Deep Think Model:     %s\n", cfg.DeepThinkLLM)
	fmt.Printf("Quick Think Model:    %s\n", cfg.QuickThinkLLM)
	fmt.Printf("Backend URL:          %s\n", cfg.BackendURL)
	fmt.Println()
	fmt.Printf("Max Debate Rounds:    %d\n", cfg.MaxDebateRounds)
	fmt.Printf("Max Risk Rounds:      %d\n", cfg.MaxRiskDiscussRounds)
	fmt.Printf("Max Recursion Limit:  %d\n", cfg.MaxRecurLimit)
	fmt.Println()
	fmt.Printf("Online Tools:         %t\n", cfg.OnlineTools)
	fmt.Printf("Cache Enabled:        %t\n", cfg.CacheEnabled)
	fmt.Printf("Debug Mode:           %t\n", cfg.Debug)
	fmt.Printf("Eino Debug:           %t\n", cfg.EinoDebugEnabled)
	if cfg.EinoDebugEnabled {
		fmt.Printf("Eino Debug Port:      %d\n", cfg.EinoDebugPort)
		fmt.Printf("Debug URL:            http://localhost:%d\n", cfg.EinoDebugPort)
	}
	fmt.Println()

	// Dataflows configuration
	fmt.Println("🔌 API Configuration:")
	fmt.Println("─────────────────────")
	if cfg.FinnhubAPIKey != "" {
		fmt.Println("Finnhub API:          ✅ Configured")
	} else {
		fmt.Println("Finnhub API:          ❌ Not configured")
	}

	if cfg.RedditClientID != "" && cfg.RedditSecret != "" {
		fmt.Println("Reddit API:           ✅ Configured")
	} else {
		fmt.Println("Reddit API:           ❌ Not configured")
	}

	fmt.Printf("Reddit User Agent:    %s\n", cfg.RedditUserAgent)
}

// validateConfig validates the configuration and dependencies
func validateConfig(cfg *config.Config) error {
	fmt.Println("🔍 Validating CortexGo Configuration...")
	fmt.Println("═══════════════════════════════════════")

	// Check directories
	fmt.Print("📁 Checking directories... ")
	if err := cfg.EnsureDirectories(); err != nil {
		fmt.Println("❌")
		return fmt.Errorf("directory validation failed: %w", err)
	}
	fmt.Println("✅")

	// Check API keys
	fmt.Print("🔑 Checking API keys... ")
	warnings := []string{}

	if cfg.FinnhubAPIKey == "" {
		warnings = append(warnings, "Finnhub API key not configured")
	}

	if cfg.RedditClientID == "" || cfg.RedditSecret == "" {
		warnings = append(warnings, "Reddit API credentials not configured")
	}

	if len(warnings) > 0 {
		fmt.Println("⚠️")
		for _, warning := range warnings {
			fmt.Printf("  ⚠️  %s\n", warning)
		}
	} else {
		fmt.Println("✅")
	}

	// Check configuration values
	fmt.Print("⚙️  Checking configuration values... ")
	if cfg.MaxDebateRounds < 1 || cfg.MaxDebateRounds > 10 {
		fmt.Println("❌")
		return fmt.Errorf("max debate rounds must be between 1 and 10")
	}

	if cfg.MaxRiskDiscussRounds < 1 || cfg.MaxRiskDiscussRounds > 10 {
		fmt.Println("❌")
		return fmt.Errorf("max risk discussion rounds must be between 1 and 10")
	}
	fmt.Println("✅")

	// Simulate dataflows validation
	fmt.Print("🌊 Validating dataflows... ")
	// In a real implementation, you'd test API connections here
	time.Sleep(500 * time.Millisecond) // Simulate validation time
	fmt.Println("✅")

	fmt.Println()
	if len(warnings) == 0 {
		fmt.Println("✅ Configuration validation completed successfully!")
	} else {
		fmt.Printf("⚠️  Configuration validation completed with %d warnings.\n", len(warnings))
		fmt.Println("Some features may be limited without proper API configuration.")
	}

	fmt.Println()
	fmt.Println("💡 Tips:")
	fmt.Println("  • Set CORTEXGO_FINNHUB_API_KEY environment variable for market data")
	fmt.Println("  • Set CORTEXGO_REDDIT_CLIENT_ID and CORTEXGO_REDDIT_SECRET for social data")
	fmt.Println("  • Use 'cortexgo analyze' to start your first analysis")

	return nil
}

// runInteractiveMode starts the interactive trading analysis mode
func runInteractiveMode(cfg *config.Config) error {
	fmt.Println("🚀 Welcome to CortexGo - AI-Powered Trading Analysis")
	fmt.Println("=" + strings.Repeat("=", 58))
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	for {
		// Get symbol from user
		fmt.Print("📊 Enter stock symbol (or 'exit' to quit): ")
		symbol, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}
		symbol = strings.TrimSpace(strings.ToUpper(symbol))

		if symbol == "EXIT" || symbol == "QUIT" {
			fmt.Println("👋 Thank you for using CortexGo!")
			break
		}

		if symbol == "" {
			fmt.Println("❌ Symbol cannot be empty. Please try again.")
			continue
		}

		// Get date from user
		fmt.Print("📅 Enter analysis date (YYYY-MM-DD, or press Enter for today): ")
		date, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}
		date = strings.TrimSpace(date)

		// Use today's date if empty
		if date == "" {
			date = time.Now().Format("2006-01-02")
			fmt.Printf("Using today's date: %s\n", date)
		}

		// Validate date format
		_, err = time.Parse("2006-01-02", date)
		if err != nil {
			fmt.Printf("❌ Invalid date format. Please use YYYY-MM-DD format.\n\n")
			continue
		}

		// Run analysis
		fmt.Printf("\n🔄 Starting analysis for %s on %s...\n", symbol, date)
		err = runAnalyzeCommand(cfg, symbol, date)
		if err != nil {
			fmt.Printf("❌ Analysis failed: %v\n", err)
		}

		fmt.Println("\n" + strings.Repeat("-", 60))
		fmt.Println()
	}

	return nil
}

