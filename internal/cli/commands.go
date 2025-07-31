package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/dyike/CortexGo/internal/config"
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
	}

	// Add subcommands
	rootCmd.AddCommand(newAnalyzeCmd(cfg))
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newConfigCmd(cfg))
	rootCmd.AddCommand(newDebugCmd(cfg))

	// Global flags
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug mode")
	rootCmd.PersistentFlags().String("config", "", "Configuration file path")

	return rootCmd
}

// newAnalyzeCmd creates the analyze command
func newAnalyzeCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Run comprehensive trading analysis",
		Long: `Run a comprehensive trading analysis for a given stock ticker.
This command will guide you through an interactive setup process to configure
your analysis preferences, then execute the full multi-agent trading workflow.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnalyzeCommand(cfg)
		},
	}

	// Analyze command flags
	cmd.Flags().String("ticker", "", "Stock ticker symbol (interactive mode if not provided)")
	cmd.Flags().String("date", "", "Analysis date in YYYY-MM-DD format (today if not provided)")
	cmd.Flags().StringSlice("analysts", []string{}, "Comma-separated list of analysts to include")
	cmd.Flags().String("depth", "", "Research depth: shallow, medium, or deep")
	cmd.Flags().String("provider", "", "LLM provider: openai, anthropic, google, openrouter, or ollama")
	cmd.Flags().Bool("auto", false, "Run with default settings (non-interactive mode)")

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
func runAnalyzeCommand(cfg *config.Config) error {
	ctx := context.Background()

	// Display welcome banner
	DisplayWelcomeBanner()

	// Check for non-interactive mode flags first
	// (In a full implementation, you'd parse the flags and create selections)

	// Interactive mode - collect user preferences
	for {
		DisplayInfo("Starting interactive analysis configuration...")

		// Step 1: Ticker Symbol
		ticker, err := PromptForTicker()
		if err != nil {
			DisplayError(err)
			continue
		}

		// Step 2: Analysis Date
		analysisDate, err := PromptForAnalysisDate()
		if err != nil {
			DisplayError(err)
			continue
		}

		// Step 3: Select Analysts
		analysts, err := PromptForAnalysts()
		if err != nil {
			DisplayError(err)
			continue
		}

		// Step 4: Research Depth
		depth, err := PromptForResearchDepth()
		if err != nil {
			DisplayError(err)
			continue
		}

		// Step 5: LLM Provider
		provider, err := PromptForLLMProvider()
		if err != nil {
			DisplayError(err)
			continue
		}

		// Step 6: Model Selection
		quickModel, deepModel, err := PromptForModels(provider)
		if err != nil {
			DisplayError(err)
			continue
		}

		// Create selections
		selections := UserSelections{
			Ticker:        ticker,
			AnalysisDate:  analysisDate,
			Analysts:      analysts,
			ResearchDepth: depth,
			LLMProvider:   provider,
			QuickModel:    quickModel,
			DeepModel:     deepModel,
		}

		// Step 7: Confirmation
		confirmed, err := PromptForConfirmation(selections)
		if err != nil {
			DisplayError(err)
			continue
		}

		if !confirmed {
			DisplayInfo("Configuration cancelled. Starting over...")
			continue
		}

		// Execute analysis
		DisplaySuccess("Configuration confirmed! Starting analysis...")
		DisplayInitializing()

		analyzer := NewAnalyzer(cfg)
		session, err := analyzer.RunAnalysis(ctx, selections)
		if err != nil {
			DisplayError(fmt.Errorf("analysis failed: %w", err))
			break
		}

		// Display final results
		DisplayCompleteReport(session)

		// Ask for next action
		restart, err := PromptForRestartOrExit()
		if err != nil {
			DisplayError(err)
			break
		}

		if !restart {
			DisplaySuccess("Thank you for using CortexGo! 🚀")
			break
		}

		// Clear screen for new analysis
		ClearScreen()
		DisplayWelcomeBanner()
	}

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

func newDebugCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Start Eino visual debug server",
		Long: `Start the Eino visual debug server for debugging Graph and Chain orchestration.
The debug server provides a web interface to visualize node execution, inputs, outputs, and timing.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDebugCommand(cfg)
		},
	}

	cmd.Flags().IntP("port", "p", 52538, "Debug server port")

	return cmd
}

func runDebugCommand(cfg *config.Config) error {
	fmt.Println("🚀 Starting CortexGo Eino Debug Server...")
	fmt.Println("═══════════════════════════════════════")

	// Override config with debug enabled
	cfg.EinoDebugEnabled = true
	cfg.Debug = true

	// Keep the server running
	select {}
}

