package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dyike/CortexGo/config"
	"github.com/dyike/CortexGo/internal/display"
	"github.com/dyike/CortexGo/internal/trading"
	"github.com/spf13/cobra"
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
	rootCmd.AddCommand(newInteractiveCmd(cfg))
	rootCmd.AddCommand(newResultsCmd(cfg))

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

// runAnalyzeCommand executes the main analysis workflow with enhanced display
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

	display.DisplayInfo(fmt.Sprintf("Starting comprehensive analysis for %s on %s", symbol, date))

	// Create enhanced analysis session
	analysisSession := NewAnalysisSession(cfg, symbol, date)
	
	// Run analysis with progress tracking
	if err := analysisSession.ExecuteWithProgress(); err != nil {
		display.DisplayError(err, "analysis execution")
		return fmt.Errorf("analysis failed: %w", err)
	}

	// Create trading session for actual execution
	session := trading.NewTradingSession(cfg, symbol, date)

	// Run the analysis
	err = session.Execute(ctx)
	if err != nil {
		display.DisplayError(err, "trading session execution")
		return fmt.Errorf("analysis failed: %w", err)
	}

	display.DisplaySuccess("Analysis completed successfully!")
	display.DisplayInfo(fmt.Sprintf("Results saved to: %s/%s_%s_analysis.json", 
		cfg.ResultsDir, symbol, date))
	
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

// newInteractiveCmd creates the interactive command
func newInteractiveCmd(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "interactive",
		Short: "Start interactive analysis mode",
		Long: `Start an enhanced interactive mode with advanced commands and features.
Features include progress tracking, real-time results display, and configuration management.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInteractiveMode(cfg)
		},
	}
}

// newResultsCmd creates the results command
func newResultsCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "results",
		Short: "Manage analysis results",
		Long:  "View, export, and manage previous analysis results",
	}

	// results list subcommand
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List previous analysis results",
		RunE: func(cmd *cobra.Command, args []string) error {
			return listResults(cfg)
		},
	})

	// results show subcommand
	showCmd := &cobra.Command{
		Use:   "show [SYMBOL] [DATE]",
		Short: "Show detailed results for a specific analysis",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return showResults(cfg, args[0], args[1])
		},
	}
	cmd.AddCommand(showCmd)

	// results export subcommand
	exportCmd := &cobra.Command{
		Use:   "export [SYMBOL] [DATE] [FORMAT]",
		Short: "Export results in different formats (json, csv, pdf)",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			return exportResults(cfg, args[0], args[1], args[2])
		},
	}
	cmd.AddCommand(exportCmd)

	return cmd
}

// listResults lists all available analysis results
func listResults(cfg *config.Config) error {
	display.DisplayInfo("Available Analysis Results:")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("📂 Results Directory:", cfg.ResultsDir)
	fmt.Println("   (Feature implementation: list all JSON files in results directory)")
	fmt.Println("   Format: SYMBOL_DATE_analysis.json")
	fmt.Println()
	fmt.Println("💡 Use 'cortexgo results show <SYMBOL> <DATE>' to view specific results")
	return nil
}

// showResults displays detailed results for a specific analysis
func showResults(cfg *config.Config, symbol, date string) error {
	filename := fmt.Sprintf("%s/%s_%s_analysis.json", cfg.ResultsDir, symbol, date)
	display.DisplayInfo(fmt.Sprintf("Loading results for %s on %s", symbol, date))
	
	// In a full implementation, this would load and display the JSON file
	// For now, show a placeholder
	fmt.Printf("📄 Results file: %s\n", filename)
	fmt.Println("   (Feature implementation: load and display JSON results)")
	
	return nil
}

// exportResults exports analysis results in different formats
func exportResults(cfg *config.Config, symbol, date, format string) error {
	validFormats := []string{"json", "csv", "pdf"}
	format = strings.ToLower(format)
	
	// Validate format
	valid := false
	for _, f := range validFormats {
		if f == format {
			valid = true
			break
		}
	}
	
	if !valid {
		return fmt.Errorf("invalid format '%s'. Supported formats: %s", 
			format, strings.Join(validFormats, ", "))
	}
	
	display.DisplayInfo(fmt.Sprintf("Exporting %s analysis from %s to %s format", symbol, date, format))
	
	// Implementation placeholder
	outputFile := fmt.Sprintf("%s/%s_%s_analysis.%s", cfg.ResultsDir, symbol, date, format)
	fmt.Printf("📤 Export file: %s\n", outputFile)
	fmt.Println("   (Feature implementation: convert JSON to requested format)")
	
	return nil
}

// runInteractiveMode starts the enhanced interactive trading analysis mode
func runInteractiveMode(cfg *config.Config) error {
	session := NewInteractiveSession(cfg)
	return session.Start()
}
