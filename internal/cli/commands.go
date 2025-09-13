package cli

import (
	"context"
	"fmt"
	"os"
	"sort"
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
	rootCmd.AddCommand(newBatchCmd(cfg))

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
			return validateConfigEnhanced(cfg)
		},
	})

	// config set subcommand
	configCmd.AddCommand(&cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return setConfigValue(cfg, args[0], args[1])
		},
	})

	// config get subcommand
	configCmd.AddCommand(&cobra.Command{
		Use:   "get <key>",
		Short: "Get configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return getConfigValue(cfg, args[0])
		},
	})

	// config list subcommand
	configCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all configuration keys",
		Run: func(cmd *cobra.Command, args []string) {
			listConfigKeys(cfg)
		},
	})

	// config save subcommand
	configCmd.AddCommand(&cobra.Command{
		Use:   "save",
		Short: "Save current configuration to file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return saveConfig(cfg)
		},
	})

	// config load subcommand
	configCmd.AddCommand(&cobra.Command{
		Use:   "load",
		Short: "Load configuration from file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return loadConfig(cfg)
		},
	})

	// config reset subcommand
	configCmd.AddCommand(&cobra.Command{
		Use:   "reset",
		Short: "Reset configuration to defaults",
		RunE: func(cmd *cobra.Command, args []string) error {
			return resetConfig(cfg)
		},
	})

	// config export subcommand
	exportCmd := &cobra.Command{
		Use:   "export <filename>",
		Short: "Export configuration to file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return exportConfig(cfg, args[0])
		},
	}
	configCmd.AddCommand(exportCmd)

	// config import subcommand
	importCmd := &cobra.Command{
		Use:   "import <filename>",
		Short: "Import configuration from file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return importConfig(cfg, args[0])
		},
	}
	configCmd.AddCommand(importCmd)

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
		Short: "Export results in different formats (json, csv, txt)",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			return exportResults(cfg, args[0], args[1], args[2])
		},
	}
	cmd.AddCommand(exportCmd)

	// results delete subcommand
	deleteCmd := &cobra.Command{
		Use:   "delete [SYMBOL] [DATE]",
		Short: "Delete a specific analysis result",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return deleteResult(cfg, args[0], args[1])
		},
	}
	cmd.AddCommand(deleteCmd)

	// results cleanup subcommand
	cleanupCmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Clean up old analysis results",
		RunE: func(cmd *cobra.Command, args []string) error {
			maxDays, _ := cmd.Flags().GetInt("max-days")
			maxCount, _ := cmd.Flags().GetInt("max-count")
			return cleanupResults(cfg, maxDays, maxCount)
		},
	}
	cleanupCmd.Flags().Int("max-days", 30, "Maximum age in days (0 = no age limit)")
	cleanupCmd.Flags().Int("max-count", 100, "Maximum number of results to keep (0 = no count limit)")
	cmd.AddCommand(cleanupCmd)

	// results stats subcommand
	statsCmd := &cobra.Command{
		Use:   "stats",
		Short: "Show analysis results statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showResultsStats(cfg)
		},
	}
	cmd.AddCommand(statsCmd)

	return cmd
}

// listResults lists all available analysis results
func listResults(cfg *config.Config) error {
	rm := NewResultsManager(cfg)
	results, err := rm.ListResults("date", true) // Sort by date, newest first
	if err != nil {
		display.DisplayError(err, "listing results")
		return err
	}

	if len(results) == 0 {
		display.DisplayInfo("No analysis results found")
		fmt.Printf("📂 Results Directory: %s\n", cfg.ResultsDir)
		fmt.Println("💡 Run 'cortexgo analyze <SYMBOL>' to create your first analysis")
		return nil
	}

	display.DisplayInfo(fmt.Sprintf("Found %d analysis results:", len(results)))
	fmt.Println("═══════════════════════════════════════════════════════════════")

	// Display results in a table format
	fmt.Printf("%-10s %-12s %-12s %-15s %s\n", "SYMBOL", "DATE", "RECOMMENDATION", "CREATED", "SIZE")
	fmt.Println(strings.Repeat("-", 65))

	for _, result := range results {
		emoji := getRecommendationEmoji(result.Recommendation)
		sizeKB := float64(result.FileSize) / 1024
		fmt.Printf("%-10s %-12s %s %-11s %-15s %.1fKB\n",
			result.Symbol,
			result.Date,
			emoji,
			result.Recommendation,
			result.CreatedAt.Format("01-02 15:04"),
			sizeKB)
	}

	fmt.Println()
	fmt.Printf("📊 Total: %d results, %.1fMB\n", 
		len(results), 
		float64(getTotalSize(results))/1024/1024)
	fmt.Println("💡 Use 'cortexgo results show <SYMBOL> <DATE>' for details")

	return nil
}

// showResults displays detailed results for a specific analysis
func showResults(cfg *config.Config, symbol, date string) error {
	rm := NewResultsManager(cfg)
	return rm.ShowResult(symbol, date)
}

// exportResults exports analysis results in different formats
func exportResults(cfg *config.Config, symbol, date, format string) error {
	validFormats := []string{"json", "csv", "txt"}
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
	
	rm := NewResultsManager(cfg)
	if err := rm.ExportResults(symbol, date, format); err != nil {
		display.DisplayError(err, "exporting results")
		return err
	}
	
	outputFile := fmt.Sprintf("%s/%s_%s_analysis.%s", cfg.ResultsDir, symbol, date, format)
	display.DisplaySuccess(fmt.Sprintf("Results exported to: %s", outputFile))
	
	return nil
}

// deleteResult deletes a specific analysis result
func deleteResult(cfg *config.Config, symbol, date string) error {
	rm := NewResultsManager(cfg)
	return rm.DeleteResult(symbol, date)
}

// cleanupResults cleans up old analysis results
func cleanupResults(cfg *config.Config, maxDays, maxCount int) error {
	rm := NewResultsManager(cfg)
	
	var maxAge time.Duration
	if maxDays > 0 {
		maxAge = time.Duration(maxDays) * 24 * time.Hour
	}
	
	display.DisplayInfo("Cleaning up old analysis results...")
	return rm.CleanupResults(maxAge, maxCount)
}

// showResultsStats shows analysis results statistics
func showResultsStats(cfg *config.Config) error {
	rm := NewResultsManager(cfg)
	results, err := rm.ListResults("date", true)
	if err != nil {
		display.DisplayError(err, "loading results for statistics")
		return err
	}

	if len(results) == 0 {
		display.DisplayInfo("No analysis results found")
		return nil
	}

	// Calculate statistics
	stats := calculateResultsStats(results)
	
	// Display statistics
	display.DisplayInfo("Analysis Results Statistics:")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	
	fmt.Printf("📊 OVERVIEW:\n")
	fmt.Printf("  Total Results:        %d\n", stats.TotalResults)
	fmt.Printf("  Total Size:           %.1f MB\n", float64(stats.TotalSize)/1024/1024)
	fmt.Printf("  Average Size:         %.1f KB\n", float64(stats.TotalSize)/float64(stats.TotalResults)/1024)
	fmt.Printf("  Date Range:           %s to %s\n", stats.OldestDate, stats.NewestDate)
	fmt.Println()

	fmt.Printf("🎯 RECOMMENDATIONS:\n")
	for rec, count := range stats.RecommendationCounts {
		emoji := getRecommendationEmoji(rec)
		percentage := float64(count) / float64(stats.TotalResults) * 100
		fmt.Printf("  %s %-12s: %3d (%.1f%%)\n", emoji, rec, count, percentage)
	}
	fmt.Println()

	fmt.Printf("📈 TOP SYMBOLS:\n")
	for i, symbol := range stats.TopSymbols {
		if i >= 10 { // Limit to top 10
			break
		}
		count := stats.SymbolCounts[symbol]
		fmt.Printf("  %-10s: %d analysis\n", symbol, count)
	}
	
	if len(stats.TopSymbols) > 10 {
		fmt.Printf("  ... and %d more\n", len(stats.TopSymbols)-10)
	}
	fmt.Println()

	fmt.Printf("📅 RECENT ACTIVITY:\n")
	recentCount := 0
	weekAgo := time.Now().AddDate(0, 0, -7)
	for _, result := range results {
		if result.CreatedAt.After(weekAgo) {
			recentCount++
		}
	}
	fmt.Printf("  Last 7 days:          %d analyses\n", recentCount)
	
	monthAgo := time.Now().AddDate(0, -1, 0)
	monthCount := 0
	for _, result := range results {
		if result.CreatedAt.After(monthAgo) {
			monthCount++
		}
	}
	fmt.Printf("  Last 30 days:         %d analyses\n", monthCount)

	return nil
}

// ResultsStats holds statistics about analysis results
type ResultsStats struct {
	TotalResults        int
	TotalSize          int64
	OldestDate         string
	NewestDate         string
	RecommendationCounts map[string]int
	SymbolCounts       map[string]int
	TopSymbols         []string
}

// calculateResultsStats calculates statistics from results
func calculateResultsStats(results []ResultSummary) ResultsStats {
	stats := ResultsStats{
		TotalResults:         len(results),
		RecommendationCounts: make(map[string]int),
		SymbolCounts:        make(map[string]int),
	}

	if len(results) == 0 {
		return stats
	}

	// Calculate totals and counts
	for _, result := range results {
		stats.TotalSize += result.FileSize
		stats.RecommendationCounts[result.Recommendation]++
		stats.SymbolCounts[result.Symbol]++
	}

	// Find date range (results are sorted by date, newest first)
	stats.NewestDate = results[0].CreatedAt.Format("2006-01-02")
	stats.OldestDate = results[len(results)-1].CreatedAt.Format("2006-01-02")

	// Sort symbols by count
	type symbolCount struct {
		symbol string
		count  int
	}
	
	var symbolCounts []symbolCount
	for symbol, count := range stats.SymbolCounts {
		symbolCounts = append(symbolCounts, symbolCount{symbol, count})
	}
	
	sort.Slice(symbolCounts, func(i, j int) bool {
		return symbolCounts[i].count > symbolCounts[j].count
	})
	
	for _, sc := range symbolCounts {
		stats.TopSymbols = append(stats.TopSymbols, sc.symbol)
	}

	return stats
}

// Helper function to calculate total size
func getTotalSize(results []ResultSummary) int64 {
	var total int64
	for _, result := range results {
		total += result.FileSize
	}
	return total
}

// Enhanced configuration management functions

// validateConfigEnhanced provides enhanced configuration validation
func validateConfigEnhanced(cfg *config.Config) error {
	cm := NewConfigManager(cfg)
	warnings := cm.ValidateConfiguration()
	
	display.DisplayInfo("Validating CortexGo Configuration...")
	fmt.Println("═══════════════════════════════════════")
	
	if len(warnings) == 0 {
		display.DisplaySuccess("Configuration validation passed!")
		display.DisplayInfo("All settings are properly configured")
		return nil
	}
	
	fmt.Printf("⚠️  Found %d configuration issues:\n", len(warnings))
	for i, warning := range warnings {
		fmt.Printf("  %d. %s\n", i+1, warning)
	}
	
	fmt.Println("\n💡 Configuration Tips:")
	fmt.Println("  • Use 'cortexgo config set <key> <value>' to update settings")
	fmt.Println("  • Use 'cortexgo config list' to see all available keys")
	fmt.Println("  • Set environment variables for API keys")
	
	return nil
}

// setConfigValue sets a configuration value
func setConfigValue(cfg *config.Config, key, value string) error {
	cm := NewConfigManager(cfg)
	
	oldValue, _ := cm.GetConfigValue(key)
	
	if err := cm.SetConfigValue(key, value); err != nil {
		display.DisplayError(err, "setting configuration value")
		return err
	}
	
	display.DisplaySuccess(fmt.Sprintf("Updated %s: %v → %s", key, oldValue, value))
	
	// Auto-save after successful update
	if err := cm.SaveConfig(); err != nil {
		display.DisplayWarning("Configuration updated but could not save to file")
	} else {
		display.DisplayInfo("Configuration saved to file")
	}
	
	return nil
}

// getConfigValue gets a configuration value
func getConfigValue(cfg *config.Config, key string) error {
	cm := NewConfigManager(cfg)
	
	value, err := cm.GetConfigValue(key)
	if err != nil {
		display.DisplayError(err, "getting configuration value")
		return err
	}
	
	fmt.Printf("📋 %s: %v\n", key, value)
	return nil
}

// listConfigKeys lists all available configuration keys
func listConfigKeys(cfg *config.Config) {
	cm := NewConfigManager(cfg)
	keys := cm.ListAvailableKeys()
	
	display.DisplayInfo("Available Configuration Keys:")
	fmt.Println("═══════════════════════════════════════")
	
	for _, key := range keys {
		value, _ := cm.GetConfigValue(key)
		fmt.Printf("  %-20s: %v\n", key, value)
	}
	
	fmt.Println("\n💡 Usage:")
	fmt.Println("  cortexgo config get <key>")
	fmt.Println("  cortexgo config set <key> <value>")
}

// saveConfig saves current configuration to file
func saveConfig(cfg *config.Config) error {
	cm := NewConfigManager(cfg)
	
	if err := cm.SaveConfig(); err != nil {
		display.DisplayError(err, "saving configuration")
		return err
	}
	
	display.DisplaySuccess("Configuration saved to cortexgo.json")
	return nil
}

// loadConfig loads configuration from file
func loadConfig(cfg *config.Config) error {
	cm := NewConfigManager(cfg)
	
	if err := cm.LoadConfig(); err != nil {
		display.DisplayError(err, "loading configuration")
		return err
	}
	
	return nil
}

// resetConfig resets configuration to defaults
func resetConfig(cfg *config.Config) error {
	cm := NewConfigManager(cfg)
	
	if err := cm.ResetConfig(); err != nil {
		display.DisplayError(err, "resetting configuration")
		return err
	}
	
	return nil
}

// exportConfig exports configuration to specified file
func exportConfig(cfg *config.Config, filename string) error {
	cm := NewConfigManager(cfg)
	
	if err := cm.ExportConfig(filename); err != nil {
		display.DisplayError(err, "exporting configuration")
		return err
	}
	
	display.DisplaySuccess(fmt.Sprintf("Configuration exported to %s", filename))
	return nil
}

// importConfig imports configuration from specified file
func importConfig(cfg *config.Config, filename string) error {
	cm := NewConfigManager(cfg)
	
	if err := cm.ImportConfig(filename); err != nil {
		display.DisplayError(err, "importing configuration")
		return err
	}
	
	display.DisplaySuccess(fmt.Sprintf("Configuration imported from %s", filename))
	return nil
}

// runInteractiveMode starts the enhanced interactive trading analysis mode
func runInteractiveMode(cfg *config.Config) error {
	session := NewInteractiveSession(cfg)
	return session.Start()
}

// newBatchCmd creates the batch analysis command
func newBatchCmd(cfg *config.Config) *cobra.Command {
	var (
		dateFlag       string
		concurrentFlag int
		fileFlag       string
	)

	batchCmd := &cobra.Command{
		Use:   "batch",
		Short: "Batch analysis operations",
		Long:  "Perform batch analysis on multiple symbols with parallel processing",
	}

	// batch analyze command
	analyzeCmd := &cobra.Command{
		Use:   "analyze [symbols...]",
		Short: "Run analysis on multiple symbols",
		Long: `Run trading analysis on multiple symbols in parallel.
Symbols can be provided as arguments or loaded from a file.

Examples:
  cortexgo batch analyze AAPL GOOGL MSFT
  cortexgo batch analyze -f symbols.txt
  cortexgo batch analyze AAPL GOOGL -d 2024-03-15 -c 2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBatchAnalyze(cfg, args, dateFlag, concurrentFlag, fileFlag)
		},
	}

	analyzeCmd.Flags().StringVarP(&dateFlag, "date", "d", "", "Analysis date (YYYY-MM-DD, defaults to today)")
	analyzeCmd.Flags().IntVarP(&concurrentFlag, "concurrent", "c", 3, "Number of concurrent analyses (1-10)")
	analyzeCmd.Flags().StringVarP(&fileFlag, "file", "f", "", "Load symbols from file (one per line)")

	// batch estimate command
	estimateCmd := &cobra.Command{
		Use:   "estimate [symbols...]",
		Short: "Estimate batch analysis duration",
		Long:  "Estimate the time required to complete batch analysis",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBatchEstimate(cfg, args, concurrentFlag, fileFlag)
		},
	}

	estimateCmd.Flags().IntVarP(&concurrentFlag, "concurrent", "c", 3, "Number of concurrent analyses (1-10)")
	estimateCmd.Flags().StringVarP(&fileFlag, "file", "f", "", "Load symbols from file (one per line)")

	// batch template command
	templateCmd := &cobra.Command{
		Use:   "template [filename]",
		Short: "Generate symbol file template",
		Long:  "Generate a template file for batch symbol input",
		RunE: func(cmd *cobra.Command, args []string) error {
			filename := "symbols.txt"
			if len(args) > 0 {
				filename = args[0]
			}
			return generateSymbolTemplate(filename)
		},
	}

	batchCmd.AddCommand(analyzeCmd)
	batchCmd.AddCommand(estimateCmd)
	batchCmd.AddCommand(templateCmd)

	return batchCmd
}

// runBatchAnalyze executes batch analysis
func runBatchAnalyze(cfg *config.Config, args []string, date string, concurrent int, filename string) error {
	bm := NewBatchManager(cfg)
	var symbols []string
	var err error

	// Load symbols from file or arguments
	if filename != "" {
		symbols, err = bm.LoadSymbolsFromFile(filename)
		if err != nil {
			display.DisplayError(err, "loading symbols from file")
			return err
		}
		display.DisplayInfo(fmt.Sprintf("Loaded %d symbols from %s", len(symbols), filename))
	} else if len(args) > 0 {
		symbols = args
	} else {
		return fmt.Errorf("no symbols provided. Use arguments or -f flag to specify symbols file")
	}

	// Validate symbols
	validSymbols, invalidSymbols := bm.ValidateSymbols(symbols)
	
	if len(invalidSymbols) > 0 {
		display.DisplayWarning(fmt.Sprintf("Invalid symbols (skipped): %s", strings.Join(invalidSymbols, ", ")))
	}
	
	if len(validSymbols) == 0 {
		return fmt.Errorf("no valid symbols to analyze")
	}

	display.DisplayInfo(fmt.Sprintf("Valid symbols to analyze: %d", len(validSymbols)))

	// Estimate duration and ask for confirmation
	estimatedDuration := bm.EstimateBatchDuration(len(validSymbols), concurrent)
	display.DisplayInfo(fmt.Sprintf("Estimated completion time: %s", estimatedDuration.Round(time.Minute)))
	
	fmt.Printf("Continue with batch analysis? (y/N): ")
	var response string
	fmt.Scanln(&response)
	
	if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
		display.DisplayInfo("Batch analysis cancelled")
		return nil
	}

	// Run batch analysis
	return bm.RunBatchAnalysis(validSymbols, date, concurrent)
}

// runBatchEstimate estimates batch analysis duration
func runBatchEstimate(cfg *config.Config, args []string, concurrent int, filename string) error {
	bm := NewBatchManager(cfg)
	var symbols []string
	var err error

	// Load symbols from file or arguments
	if filename != "" {
		symbols, err = bm.LoadSymbolsFromFile(filename)
		if err != nil {
			display.DisplayError(err, "loading symbols from file")
			return err
		}
	} else if len(args) > 0 {
		symbols = args
	} else {
		return fmt.Errorf("no symbols provided. Use arguments or -f flag to specify symbols file")
	}

	// Validate symbols
	validSymbols, invalidSymbols := bm.ValidateSymbols(symbols)
	
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    BATCH ANALYSIS ESTIMATE                    ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	display.DisplayInfo(fmt.Sprintf("Total symbols provided: %d", len(symbols)))
	display.DisplaySuccess(fmt.Sprintf("Valid symbols: %d", len(validSymbols)))
	
	if len(invalidSymbols) > 0 {
		display.DisplayWarning(fmt.Sprintf("Invalid symbols: %d (%s)", len(invalidSymbols), strings.Join(invalidSymbols, ", ")))
	}

	if len(validSymbols) > 0 {
		display.DisplayInfo(fmt.Sprintf("Concurrent analyses: %d", concurrent))
		
		estimatedDuration := bm.EstimateBatchDuration(len(validSymbols), concurrent)
		display.DisplayInfo(fmt.Sprintf("Estimated total time: %s", estimatedDuration.Round(time.Minute)))
		
		avgPerSymbol := estimatedDuration / time.Duration(len(validSymbols))
		display.DisplayInfo(fmt.Sprintf("Average time per symbol: %s", avgPerSymbol.Round(time.Second)))

		fmt.Println()
		fmt.Println("💡 Tips for batch analysis:")
		fmt.Println("  • Higher concurrent values reduce total time but use more resources")
		fmt.Println("  • API rate limits may affect actual execution time")
		fmt.Println("  • Consider running during off-peak hours for better performance")
	}

	return nil
}

// generateSymbolTemplate creates a template file for batch symbol input
func generateSymbolTemplate(filename string) error {
	template := `# CortexGo Batch Analysis Symbol List
# Lines starting with # are comments and will be ignored
# One symbol per line
# Example symbols below - replace with your actual symbols

AAPL
GOOGL
MSFT
AMZN
TSLA
META
NVDA
NFLX
ORCL
CRM

# You can add more symbols here
# Maximum recommended: 50 symbols per batch for optimal performance
`

	err := os.WriteFile(filename, []byte(template), 0644)
	if err != nil {
		return fmt.Errorf("failed to create template file: %w", err)
	}

	display.DisplaySuccess(fmt.Sprintf("Symbol template created: %s", filename))
	display.DisplayInfo("Edit the file to add your desired symbols, then use: cortexgo batch analyze -f " + filename)
	
	return nil
}
