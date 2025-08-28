package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dyike/CortexGo/internal/config"
)

// SessionManager manages analysis sessions
type SessionManager struct {
	config *config.Config
}

// NewSessionManager creates a new session manager
func NewSessionManager(cfg *config.Config) *SessionManager {
	return &SessionManager{
		config: cfg,
	}
}

// CreateSession creates a new analysis session
func (sm *SessionManager) CreateSession(selections UserSelections) (*AnalysisSession, error) {
	// Generate session ID
	sessionID := fmt.Sprintf("%s_%s_%d",
		selections.Ticker,
		selections.AnalysisDate.Format("20060102"),
		time.Now().Unix(),
	)

	// Create results directory
	resultsDir := filepath.Join(sm.config.ResultsDir, selections.Ticker, selections.AnalysisDate.Format("2006-01-02"))
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create results directory: %w", err)
	}

	// Initialize session
	session := &AnalysisSession{
		Selections: selections,
		Progress:   AnalysisProgress{}, // All agents start as pending
		Stats: AnalysisStats{
			StartTime:    time.Now(),
			CurrentPhase: "Initialization",
		},
		Messages:   []LogMessage{},
		Reports:    []ReportSection{},
		ResultsDir: resultsDir,
		SessionID:  sessionID,
	}

	// Initialize all agent statuses to pending
	session.Progress = AnalysisProgress{
		MarketAnalyst:       StatusPending,
		SocialAnalyst:       StatusPending,
		NewsAnalyst:         StatusPending,
		FundamentalsAnalyst: StatusPending,
		BullResearcher:      StatusPending,
		BearResearcher:      StatusPending,
		ResearchManager:     StatusPending,
		Trader:              StatusPending,
		RiskyAnalyst:        StatusPending,
		SafeAnalyst:         StatusPending,
		NeutralAnalyst:      StatusPending,
		RiskManager:         StatusPending,
	}

	// Add initial log message
	session.AddLogMessage("System", "info", "session_start", "Analysis session initialized")

	// Save session to file
	if err := sm.SaveSession(session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return session, nil
}

// SaveSession saves the session to a file
func (sm *SessionManager) SaveSession(session *AnalysisSession) error {
	sessionFile := filepath.Join(session.ResultsDir, "session.json")

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	return os.WriteFile(sessionFile, data, 0644)
}

// LoadSession loads a session from a file
func (sm *SessionManager) LoadSession(sessionDir string) (*AnalysisSession, error) {
	sessionFile := filepath.Join(sessionDir, "session.json")

	data, err := os.ReadFile(sessionFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	var session AnalysisSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// AddLogMessage adds a log message to the session
func (session *AnalysisSession) AddLogMessage(agent, level, msgType, content string) {
	message := LogMessage{
		Timestamp: time.Now(),
		Agent:     agent,
		Type:      msgType,
		Content:   content,
		Level:     level,
	}

	session.Messages = append(session.Messages, message)

	// Update stats based on message type
	switch msgType {
	case "tool_call":
		session.Stats.ToolCallsCount++
	case "reasoning":
		session.Stats.LLMCallsCount++
	case "report":
		session.Stats.ReportsGenerated++
	}

	// Update elapsed time
	session.Stats.ElapsedTime = time.Since(session.Stats.StartTime)
}

// AddReport adds a report section to the session
func (session *AnalysisSession) AddReport(agent, title, content, filePath string) {
	report := ReportSection{
		Title:     title,
		Content:   content,
		Agent:     agent,
		Timestamp: time.Now(),
		FilePath:  filePath,
	}

	session.Reports = append(session.Reports, report)
	session.AddLogMessage(agent, "info", "report", fmt.Sprintf("Generated report: %s", title))
}

// UpdateAgentStatus updates the status of a specific agent
func (session *AnalysisSession) UpdateAgentStatus(agentName string, status AgentStatus) {
	switch agentName {
	case "market_analyst":
		session.Progress.MarketAnalyst = status
	case "social_analyst":
		session.Progress.SocialAnalyst = status
	case "news_analyst":
		session.Progress.NewsAnalyst = status
	case "fundamentals_analyst":
		session.Progress.FundamentalsAnalyst = status
	case "bull_researcher":
		session.Progress.BullResearcher = status
	case "bear_researcher":
		session.Progress.BearResearcher = status
	case "research_manager":
		session.Progress.ResearchManager = status
	case "trader":
		session.Progress.Trader = status
	case "risky_analyst":
		session.Progress.RiskyAnalyst = status
	case "safe_analyst":
		session.Progress.SafeAnalyst = status
	case "neutral_analyst":
		session.Progress.NeutralAnalyst = status
	case "risk_manager":
		session.Progress.RiskManager = status
	}

	// Log status change
	session.AddLogMessage(agentName, "info", "status_change", fmt.Sprintf("Status changed to %s", status))
}

// UpdateCurrentPhase updates the current phase of analysis
func (session *AnalysisSession) UpdateCurrentPhase(phase string) {
	session.Stats.CurrentPhase = phase
	session.AddLogMessage("System", "info", "phase_change", fmt.Sprintf("Entered phase: %s", phase))
}

// SaveReportToFile saves a report to a markdown file
func (session *AnalysisSession) SaveReportToFile(agent, title, content string) (string, error) {
	// Create filename
	timestamp := time.Now().Format("150405")
	filename := fmt.Sprintf("%s_%s_%s.md",
		agent,
		timestamp,
		sanitizeFilename(title),
	)

	filePath := filepath.Join(session.ResultsDir, filename)

	// Format content as markdown
	markdownContent := fmt.Sprintf("# %s\n\n**Agent:** %s  \n**Generated:** %s  \n**Ticker:** %s  \n**Analysis Date:** %s\n\n---\n\n%s",
		title,
		agent,
		time.Now().Format("2006-01-02 15:04:05"),
		session.Selections.Ticker,
		session.Selections.AnalysisDate.Format("2006-01-02"),
		content,
	)

	// Write to file
	if err := os.WriteFile(filePath, []byte(markdownContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write report file: %w", err)
	}

	return filePath, nil
}

// SaveFinalReport saves a comprehensive final report
func (session *AnalysisSession) SaveFinalReport() (string, error) {
	filename := fmt.Sprintf("final_report_%s_%s.md",
		session.Selections.Ticker,
		session.Selections.AnalysisDate.Format("20060102"),
	)

	filePath := filepath.Join(session.ResultsDir, filename)

	// Build comprehensive report
	content := fmt.Sprintf(`# Trading Analysis Report: %s

## Analysis Overview

**Ticker Symbol:** %s  
**Analysis Date:** %s  
**Analysis Duration:** %s  
**Research Depth:** %s (%d rounds)  
**LLM Provider:** %s  
**Session ID:** %s  

## Configuration Details

**Selected Analysts:**
%s

**Models Used:**
- Quick Thinking: %s
- Deep Thinking: %s

## Analysis Statistics

- **Total Agents:** %d
- **Completed Agents:** %d
- **Tool Calls Made:** %d
- **LLM Calls Made:** %d
- **Reports Generated:** %d
- **Start Time:** %s
- **End Time:** %s
- **Total Duration:** %s

## Agent Completion Status

### 👥 Analyst Team
- Market Analyst: %s
- Social Media Analyst: %s
- News Analyst: %s
- Fundamentals Analyst: %s

### 🔬 Research Team
- Bull Researcher: %s
- Bear Researcher: %s
- Research Manager: %s

### 💼 Trading Team
- Trader: %s

### ⚖️ Risk Management Team
- Risky Analyst: %s
- Safe Analyst: %s
- Neutral Analyst: %s
- Risk Manager: %s

## Generated Reports

`,
		session.Selections.Ticker,
		session.Selections.Ticker,
		session.Selections.AnalysisDate.Format("2006-01-02"),
		session.Stats.ElapsedTime.Round(time.Second),
		session.Selections.ResearchDepth,
		session.Selections.ResearchDepth.GetResearchRounds(),
		session.Selections.LLMProvider,
		session.SessionID,
		formatAnalystList(session.Selections.Analysts),
		session.Selections.QuickModel,
		session.Selections.DeepModel,
		session.Progress.GetTotalAgents(),
		session.Progress.GetCompletedCount(),
		session.Stats.ToolCallsCount,
		session.Stats.LLMCallsCount,
		session.Stats.ReportsGenerated,
		session.Stats.StartTime.Format("2006-01-02 15:04:05"),
		time.Now().Format("2006-01-02 15:04:05"),
		session.Stats.ElapsedTime.Round(time.Second),
		session.Progress.MarketAnalyst,
		session.Progress.SocialAnalyst,
		session.Progress.NewsAnalyst,
		session.Progress.FundamentalsAnalyst,
		session.Progress.BullResearcher,
		session.Progress.BearResearcher,
		session.Progress.ResearchManager,
		session.Progress.Trader,
		session.Progress.RiskyAnalyst,
		session.Progress.SafeAnalyst,
		session.Progress.NeutralAnalyst,
		session.Progress.RiskManager,
	)

	// Add individual reports
	for i, report := range session.Reports {
		content += fmt.Sprintf("### %d. %s - %s\n\n", i+1, report.Agent, report.Title)
		if report.FilePath != "" {
			content += fmt.Sprintf("**File:** `%s`\n\n", filepath.Base(report.FilePath))
		}
		content += fmt.Sprintf("**Generated:** %s\n\n", report.Timestamp.Format("2006-01-02 15:04:05"))

		// Add truncated content preview
		preview := report.Content
		if len(preview) > 500 {
			preview = preview[:500] + "...\n\n*[Full content available in individual report file]*"
		}
		content += preview + "\n\n---\n\n"
	}

	// Add activity log summary
	content += "## Activity Log Summary\n\n"

	// Count message types
	msgCounts := map[string]int{}
	for _, msg := range session.Messages {
		msgCounts[msg.Type]++
	}

	for msgType, count := range msgCounts {
		content += fmt.Sprintf("- %s: %d\n", msgType, count)
	}

	content += "\n## Files Generated\n\n"
	content += fmt.Sprintf("All analysis files are saved in: `%s`\n\n", session.ResultsDir)
	for _, report := range session.Reports {
		if report.FilePath != "" {
			content += fmt.Sprintf("- `%s`\n", filepath.Base(report.FilePath))
		}
	}

	content += "\n---\n\n*Generated by CortexGo - AI-Powered Trading Analysis*"

	// Write to file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write final report: %w", err)
	}

	return filePath, nil
}

// Helper functions

func sanitizeFilename(filename string) string {
	// Replace invalid characters with underscores
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", " "}
	result := filename

	// Simple replacement approach
	for _, invalidChar := range invalidChars {
		for i := 0; i < len(result); i++ {
			if string(result[i]) == invalidChar {
				result = result[:i] + "_" + result[i+1:]
			}
		}
	}

	// Limit length
	if len(result) > 50 {
		result = result[:50]
	}

	return result
}

func formatAnalystList(analysts []AnalystType) string {
	var result string
	for _, analyst := range analysts {
		result += fmt.Sprintf("- %s\n", analyst.GetDisplayName())
	}
	return result
}
