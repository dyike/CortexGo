package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dyike/CortexGo/config"
)

// InteractiveSession handles interactive CLI sessions
type InteractiveSession struct {
	config *config.Config
	reader *bufio.Reader
}

// NewInteractiveSession creates a new interactive session
func NewInteractiveSession(cfg *config.Config) *InteractiveSession {
	return &InteractiveSession{
		config: cfg,
		reader: bufio.NewReader(os.Stdin),
	}
}

// Start begins the interactive session
func (s *InteractiveSession) Start() error {
	s.showWelcome()
	return s.runMainLoop()
}

// showWelcome displays the welcome screen
func (s *InteractiveSession) showWelcome() {
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    🚀 CortexGo v1.0.0                         ║")
	fmt.Println("║              AI-Powered Trading Analysis System                ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("🤖 Multi-Agent Analysis Pipeline:")
	fmt.Println("   • Market Analyst     • Social Sentiment Analyst")
	fmt.Println("   • News Analyst       • Fundamentals Analyst")
	fmt.Println("   • Bull Researcher    • Bear Researcher")
	fmt.Println("   • Risk Analyst       • Portfolio Manager")
	fmt.Println()
	fmt.Println("💡 Commands:")
	fmt.Println("   analyze <SYMBOL> [date] - Run analysis for a stock")
	fmt.Println("   config              - Show/edit configuration")
	fmt.Println("   history             - View analysis history")
	fmt.Println("   help                - Show detailed help")
	fmt.Println("   exit                - Exit CortexGo")
	fmt.Println()
}

// runMainLoop runs the main interactive loop
func (s *InteractiveSession) runMainLoop() error {
	for {
		fmt.Print("📊 CortexGo> ")
		
		input, err := s.reader.ReadString('\n')
		if err != nil {
			fmt.Printf("❌ Error reading input: %v\n", err)
			continue
		}
		
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		
		// Parse command
		parts := strings.Fields(input)
		command := strings.ToLower(parts[0])
		
		switch command {
		case "exit", "quit", "q":
			fmt.Println("👋 Thank you for using CortexGo!")
			return nil
			
		case "help", "h", "?":
			s.showHelp()
			
		case "analyze", "a":
			if len(parts) < 2 {
				fmt.Println("❌ Usage: analyze <SYMBOL> [YYYY-MM-DD]")
				continue
			}
			symbol := strings.ToUpper(parts[1])
			date := ""
			if len(parts) >= 3 {
				date = parts[2]
			}
			s.runAnalysis(symbol, date)
			
		case "config", "cfg":
			s.handleConfigCommand(parts[1:])
			
		case "history", "hist":
			s.showHistory()
			
		case "clear", "cls":
			s.clearScreen()
			
		default:
			fmt.Printf("❌ Unknown command: %s. Type 'help' for available commands.\n", command)
		}
		
		fmt.Println()
	}
}

// showHelp displays detailed help information
func (s *InteractiveSession) showHelp() {
	fmt.Println("📚 CortexGo Help")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("🔍 ANALYSIS COMMANDS:")
	fmt.Println("  analyze <SYMBOL> [date]    - Run comprehensive trading analysis")
	fmt.Println("                               Example: analyze AAPL 2024-03-15")
	fmt.Println("  history                    - View previous analysis results")
	fmt.Println()
	fmt.Println("⚙️  CONFIGURATION COMMANDS:")
	fmt.Println("  config show                - Display current configuration")
	fmt.Println("  config validate           - Validate configuration and APIs")
	fmt.Println("  config set <key> <value>  - Update configuration value")
	fmt.Println()
	fmt.Println("📊 ANALYSIS WORKFLOW:")
	fmt.Println("  1. Market Data Collection  - Technical indicators, price data")
	fmt.Println("  2. Multi-Source Analysis   - News, social media, fundamentals")
	fmt.Println("  3. Research Debate         - Bull vs Bear arguments")
	fmt.Println("  4. Risk Assessment         - Conservative, Risky, Neutral views")
	fmt.Println("  5. Final Recommendation    - BUY/SELL/HOLD with reasoning")
	fmt.Println()
	fmt.Println("🔧 OTHER COMMANDS:")
	fmt.Println("  clear                      - Clear screen")
	fmt.Println("  help                       - Show this help")
	fmt.Println("  exit                       - Exit CortexGo")
	fmt.Println()
	fmt.Println("💡 Tips:")
	fmt.Println("  • Set up API keys in environment variables for full functionality")
	fmt.Println("  • Analysis typically takes 2-5 minutes depending on complexity")
	fmt.Println("  • Results are saved in the results directory for future reference")
}

// runAnalysis executes the trading analysis with enhanced UI
func (s *InteractiveSession) runAnalysis(symbol, date string) {
	// Validate date
	if date == "" {
		date = time.Now().Format("2006-01-02")
	} else {
		if _, err := time.Parse("2006-01-02", date); err != nil {
			fmt.Printf("❌ Invalid date format. Use YYYY-MM-DD format.\n")
			return
		}
	}
	
	fmt.Printf("🎯 Starting analysis for %s on %s\n", symbol, date)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	// Create analysis session with progress tracking
	session := NewAnalysisSession(s.config, symbol, date)
	
	if err := session.ExecuteWithProgress(); err != nil {
		fmt.Printf("❌ Analysis failed: %v\n", err)
		return
	}
	
	fmt.Println("✅ Analysis completed successfully!")
	fmt.Printf("📋 Results saved to: %s/%s_%s_analysis.json\n", 
		s.config.ResultsDir, symbol, date)
}

// handleConfigCommand handles configuration subcommands
func (s *InteractiveSession) handleConfigCommand(args []string) {
	if len(args) == 0 {
		showConfig(s.config)
		return
	}
	
	switch strings.ToLower(args[0]) {
	case "show", "s":
		showConfig(s.config)
		
	case "validate", "v":
		if err := validateConfig(s.config); err != nil {
			fmt.Printf("❌ Validation failed: %v\n", err)
		}
		
	case "set":
		if len(args) < 3 {
			fmt.Println("❌ Usage: config set <key> <value>")
			return
		}
		s.setConfigValue(args[1], args[2])
		
	case "edit", "e":
		s.interactiveConfigEdit()
		
	default:
		fmt.Printf("❌ Unknown config command: %s\n", args[0])
		fmt.Println("Available: show, validate, set, edit")
	}
}

// setConfigValue updates a configuration value
func (s *InteractiveSession) setConfigValue(key, value string) {
	switch strings.ToLower(key) {
	case "debug":
		if b, err := strconv.ParseBool(value); err == nil {
			s.config.Debug = b
			fmt.Printf("✅ Debug mode set to: %t\n", b)
		} else {
			fmt.Printf("❌ Invalid boolean value: %s\n", value)
		}
		
	case "max_debate_rounds":
		if i, err := strconv.Atoi(value); err == nil && i >= 1 && i <= 10 {
			s.config.MaxDebateRounds = i
			fmt.Printf("✅ Max debate rounds set to: %d\n", i)
		} else {
			fmt.Printf("❌ Invalid value. Must be between 1-10: %s\n", value)
		}
		
	case "max_risk_rounds":
		if i, err := strconv.Atoi(value); err == nil && i >= 1 && i <= 10 {
			s.config.MaxRiskDiscussRounds = i
			fmt.Printf("✅ Max risk rounds set to: %d\n", i)
		} else {
			fmt.Printf("❌ Invalid value. Must be between 1-10: %s\n", value)
		}
		
	default:
		fmt.Printf("❌ Unknown configuration key: %s\n", key)
		fmt.Println("Available keys: debug, max_debate_rounds, max_risk_rounds")
	}
}

// interactiveConfigEdit provides interactive configuration editing
func (s *InteractiveSession) interactiveConfigEdit() {
	fmt.Println("⚙️  Interactive Configuration Editor")
	fmt.Println("───────────────────────────────────")
	
	// Debug mode
	fmt.Printf("Current debug mode: %t\n", s.config.Debug)
	fmt.Print("Enable debug mode? (y/n): ")
	if input, err := s.reader.ReadString('\n'); err == nil {
		input = strings.TrimSpace(strings.ToLower(input))
		if input == "y" || input == "yes" {
			s.config.Debug = true
		} else if input == "n" || input == "no" {
			s.config.Debug = false
		}
	}
	
	// Max debate rounds
	fmt.Printf("Current max debate rounds: %d\n", s.config.MaxDebateRounds)
	fmt.Print("Set max debate rounds (1-10): ")
	if input, err := s.reader.ReadString('\n'); err == nil {
		if i, err := strconv.Atoi(strings.TrimSpace(input)); err == nil && i >= 1 && i <= 10 {
			s.config.MaxDebateRounds = i
		}
	}
	
	fmt.Println("✅ Configuration updated!")
}

// showHistory displays analysis history
func (s *InteractiveSession) showHistory() {
	fmt.Println("📈 Analysis History")
	fmt.Println("═══════════════════")
	fmt.Println("(Feature coming soon - will show previous analysis results)")
}

// clearScreen clears the terminal screen
func (s *InteractiveSession) clearScreen() {
	fmt.Print("\033[2J\033[H")
	s.showWelcome()
}

// AnalysisSession handles analysis execution with progress tracking
type AnalysisSession struct {
	config *config.Config
	symbol string
	date   string
	phases []AnalysisPhase
}

// AnalysisPhase represents a phase in the analysis pipeline
type AnalysisPhase struct {
	Name        string
	Description string
	Status      PhaseStatus
	StartTime   time.Time
	EndTime     time.Time
}

// PhaseStatus represents the status of an analysis phase
type PhaseStatus int

const (
	PhasePending PhaseStatus = iota
	PhaseRunning
	PhaseCompleted
	PhaseFailed
)

// String returns the string representation of PhaseStatus
func (ps PhaseStatus) String() string {
	switch ps {
	case PhasePending:
		return "⏳ Pending"
	case PhaseRunning:
		return "🔄 Running"
	case PhaseCompleted:
		return "✅ Completed"
	case PhaseFailed:
		return "❌ Failed"
	default:
		return "❓ Unknown"
	}
}

// NewAnalysisSession creates a new analysis session
func NewAnalysisSession(cfg *config.Config, symbol, date string) *AnalysisSession {
	return &AnalysisSession{
		config: cfg,
		symbol: symbol,
		date:   date,
		phases: []AnalysisPhase{
			{Name: "initialization", Description: "Initializing analysis pipeline"},
			{Name: "market_analysis", Description: "Analyzing market data and indicators"},
			{Name: "news_analysis", Description: "Processing latest news and events"},
			{Name: "social_analysis", Description: "Analyzing social media sentiment"},
			{Name: "fundamentals", Description: "Evaluating company fundamentals"},
			{Name: "research_debate", Description: "Bull vs Bear research debate"},
			{Name: "trading_plan", Description: "Generating trading recommendations"},
			{Name: "risk_assessment", Description: "Risk management evaluation"},
			{Name: "final_decision", Description: "Final recommendation synthesis"},
		},
	}
}

// ExecuteWithProgress runs analysis with progress tracking
func (s *AnalysisSession) ExecuteWithProgress() error {
	// This is a placeholder - integrate with actual trading session
	fmt.Println("📊 Analysis Pipeline Progress:")
	fmt.Println("─────────────────────────────────")
	
	for i := range s.phases {
		s.phases[i].Status = PhaseRunning
		s.phases[i].StartTime = time.Now()
		
		fmt.Printf("%s %s...\n", s.phases[i].Status, s.phases[i].Description)
		
		// Simulate work - replace with actual analysis logic
		time.Sleep(time.Duration(500+i*200) * time.Millisecond)
		
		s.phases[i].Status = PhaseCompleted
		s.phases[i].EndTime = time.Now()
		
		duration := s.phases[i].EndTime.Sub(s.phases[i].StartTime)
		fmt.Printf("%s %s (%.1fs)\n", 
			s.phases[i].Status, 
			s.phases[i].Description, 
			duration.Seconds())
	}
	
	fmt.Println("─────────────────────────────────")
	totalTime := s.phases[len(s.phases)-1].EndTime.Sub(s.phases[0].StartTime)
	fmt.Printf("⏱️  Total analysis time: %.1fs\n", totalTime.Seconds())
	
	return nil
}