package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// UI styles
var (
	// Base styles
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7C3AED")).
		Background(lipgloss.Color("#1F2937")).
		Padding(0, 1).
		MarginBottom(1)

	headerStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#3B82F6")).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3B82F6")).
		Padding(1, 2).
		Width(80)

	progressStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#10B981")).
		Padding(1, 2).
		Width(80).
		Height(12)

	messagesStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#F59E0B")).
		Padding(1, 2).
		Width(80).
		Height(15)

	reportsStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#EF4444")).
		Padding(1, 2).
		Width(80).
		Height(20)

	// Status styles
	pendingStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280"))

	inProgressStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F59E0B")).
		Bold(true)

	completedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#10B981")).
		Bold(true)

	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#EF4444")).
		Bold(true)

	// Message type styles
	toolCallStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8B5CF6"))

	reasoningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3B82F6"))

	reportStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#10B981")).
		Bold(true)

	logErrorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#EF4444"))
)

// DisplayWelcomeBanner shows the welcome banner
func DisplayWelcomeBanner() {
	banner := `
 ██████╗ ██████╗ ██████╗ ████████╗███████╗██╗  ██╗ ██████╗  ██████╗ 
██╔════╝██╔═══██╗██╔══██╗╚══██╔══╝██╔════╝╚██╗██╔╝██╔════╝ ██╔═══██╗
██║     ██║   ██║██████╔╝   ██║   █████╗   ╚███╔╝ ██║  ███╗██║   ██║
██║     ██║   ██║██╔══██╗   ██║   ██╔══╝   ██╔██╗ ██║   ██║██║   ██║
╚██████╗╚██████╔╝██║  ██║   ██║   ███████╗██╔╝ ██╗╚██████╔╝╚██████╔╝
 ╚═════╝ ╚═════╝ ╚═╝  ╚═╝   ╚═╝   ╚══════╝╚═╝  ╚═╝ ╚═════╝  ╚═════╝ 
                                                                     
           🚀 AI-Powered Trading Analysis & Decision Making 🚀
`

	welcomeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true).
		Align(lipgloss.Center).
		Width(80).
		MarginBottom(2)

	taglineStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3B82F6")).
		Italic(true).
		Align(lipgloss.Center).
		Width(80).
		MarginBottom(2)

	fmt.Print(welcomeStyle.Render(banner))
	fmt.Print(taglineStyle.Render("Advanced multi-agent trading analysis powered by Large Language Models"))
	fmt.Println()
}

// ClearScreen clears the terminal screen
func ClearScreen() {
	fmt.Print("\033[2J\033[H")
}

// DisplayAnalysisHeader shows the analysis header
func DisplayAnalysisHeader(session *AnalysisSession) {
	header := fmt.Sprintf("📊 Analysis: %s | 📅 Date: %s | 🎯 Phase: %s",
		session.Selections.Ticker,
		session.Selections.AnalysisDate.Format("2006-01-02"),
		session.Stats.CurrentPhase,
	)
	
	fmt.Println(headerStyle.Render(header))
}

// DisplayProgressPanel shows the agent progress panel
func DisplayProgressPanel(progress *AnalysisProgress, stats *AnalysisStats) {
	var content strings.Builder
	
	// Title
	content.WriteString("🔄 Agent Progress\n\n")
	
	// Analyst Team
	content.WriteString("👥 Analyst Team:\n")
	content.WriteString(formatAgentStatus("  Market Analyst", progress.MarketAnalyst))
	content.WriteString(formatAgentStatus("  Social Analyst", progress.SocialAnalyst))
	content.WriteString(formatAgentStatus("  News Analyst", progress.NewsAnalyst))
	content.WriteString(formatAgentStatus("  Fundamentals Analyst", progress.FundamentalsAnalyst))
	content.WriteString("\n")
	
	// Research Team
	content.WriteString("🔬 Research Team:\n")
	content.WriteString(formatAgentStatus("  Bull Researcher", progress.BullResearcher))
	content.WriteString(formatAgentStatus("  Bear Researcher", progress.BearResearcher))
	content.WriteString(formatAgentStatus("  Research Manager", progress.ResearchManager))
	content.WriteString("\n")
	
	// Trading & Risk Teams
	content.WriteString("💼 Trading Team:\n")
	content.WriteString(formatAgentStatus("  Trader", progress.Trader))
	content.WriteString("\n")
	content.WriteString("⚖️  Risk Management:\n")
	content.WriteString(formatAgentStatus("  Risk Manager", progress.RiskManager))
	
	// Statistics
	content.WriteString(fmt.Sprintf("\n📊 Stats: %d/%d completed | ⚡ %d tool calls | 🧠 %d LLM calls | 📝 %d reports",
		progress.GetCompletedCount(),
		progress.GetTotalAgents(),
		stats.ToolCallsCount,
		stats.LLMCallsCount,
		stats.ReportsGenerated,
	))
	
	fmt.Println(progressStyle.Render(content.String()))
}

// DisplayMessagesPanel shows the messages panel
func DisplayMessagesPanel(messages []LogMessage, maxMessages int) {
	var content strings.Builder
	
	content.WriteString("💬 Activity Log\n\n")
	
	if len(messages) == 0 {
		content.WriteString("No messages yet...")
		fmt.Println(messagesStyle.Render(content.String()))
		return
	}
	
	// Show only the last maxMessages
	start := 0
	if len(messages) > maxMessages {
		start = len(messages) - maxMessages
	}
	
	for i := start; i < len(messages); i++ {
		msg := messages[i]
		timestamp := msg.Timestamp.Format("15:04:05")
		
		var style lipgloss.Style
		var icon string
		
		switch msg.Type {
		case "tool_call":
			style = toolCallStyle
			icon = "🔧"
		case "reasoning":
			style = reasoningStyle
			icon = "🧠"
		case "report":
			style = reportStyle
			icon = "📝"
		case "error":
			style = logErrorStyle
			icon = "❌"
		default:
			style = lipgloss.NewStyle()
			icon = "ℹ️"
		}
		
		line := fmt.Sprintf("[%s] %s %s: %s",
			timestamp,
			icon,
			msg.Agent,
			truncateString(msg.Content, 60),
		)
		
		content.WriteString(style.Render(line) + "\n")
	}
	
	fmt.Println(messagesStyle.Render(content.String()))
}

// DisplayReportsPanel shows the reports panel
func DisplayReportsPanel(reports []ReportSection) {
	var content strings.Builder
	
	content.WriteString("📄 Generated Reports\n\n")
	
	if len(reports) == 0 {
		content.WriteString("No reports generated yet...")
		fmt.Println(reportsStyle.Render(content.String()))
		return
	}
	
	// Group reports by team
	teams := map[string][]ReportSection{
		"Analysis": {},
		"Research": {},
		"Trading": {},
		"Risk": {},
	}
	
	for _, report := range reports {
		switch {
		case strings.Contains(strings.ToLower(report.Agent), "analyst"):
			teams["Analysis"] = append(teams["Analysis"], report)
		case strings.Contains(strings.ToLower(report.Agent), "research"):
			teams["Research"] = append(teams["Research"], report)
		case strings.Contains(strings.ToLower(report.Agent), "trader"):
			teams["Trading"] = append(teams["Trading"], report)
		case strings.Contains(strings.ToLower(report.Agent), "risk"):
			teams["Risk"] = append(teams["Risk"], report)
		default:
			teams["Analysis"] = append(teams["Analysis"], report)
		}
	}
	
	// Display each team's reports
	for teamName, teamReports := range teams {
		if len(teamReports) > 0 {
			content.WriteString(fmt.Sprintf("🏢 %s Team:\n", teamName))
			for _, report := range teamReports {
				content.WriteString(fmt.Sprintf("  📋 %s - %s\n",
					report.Agent,
					truncateString(report.Title, 40),
				))
			}
			content.WriteString("\n")
		}
	}
	
	fmt.Println(reportsStyle.Render(content.String()))
}

// DisplayCompleteReport shows the final complete report
func DisplayCompleteReport(session *AnalysisSession) {
	fmt.Println()
	fmt.Println(titleStyle.Render("🎉 ANALYSIS COMPLETE! 🎉"))
	
	summary := fmt.Sprintf(`
┌─────────────────────────────────────────────────────────────────────────────┐
│                           📊 ANALYSIS SUMMARY                               │
├─────────────────────────────────────────────────────────────────────────────┤
│ Company: %s                                                          │
│ Date: %s                                                          │
│ Duration: %s                                                        │
│ Agents Completed: %d/%d                                                     │
│ Tool Calls: %d                                                              │
│ LLM Calls: %d                                                               │
│ Reports Generated: %d                                                       │
│ Results Directory: %s                                                       │
└─────────────────────────────────────────────────────────────────────────────┘
`,
		session.Selections.Ticker,
		session.Selections.AnalysisDate.Format("2006-01-02"),
		session.Stats.ElapsedTime.Round(time.Second),
		session.Progress.GetCompletedCount(),
		session.Progress.GetTotalAgents(),
		session.Stats.ToolCallsCount,
		session.Stats.LLMCallsCount,
		session.Stats.ReportsGenerated,
		session.ResultsDir,
	)
	
	fmt.Println(summary)
	
	if len(session.Reports) > 0 {
		fmt.Println("📋 Generated Reports:")
		for _, report := range session.Reports {
			fmt.Printf("  • %s: %s\n", report.Agent, report.Title)
			if report.FilePath != "" {
				fmt.Printf("    💾 Saved to: %s\n", report.FilePath)
			}
		}
	}
	
	fmt.Println()
}

// DisplayError shows an error message
func DisplayError(err error) {
	errorMsg := fmt.Sprintf("❌ Error: %s", err.Error())
	fmt.Println(errorStyle.Render(errorMsg))
}

// DisplayInfo shows an info message
func DisplayInfo(message string) {
	infoMsg := fmt.Sprintf("ℹ️  %s", message)
	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6")).Render(infoMsg))
}

// DisplaySuccess shows a success message
func DisplaySuccess(message string) {
	successMsg := fmt.Sprintf("✅ %s", message)
	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Render(successMsg))
}

// Helper functions

func formatAgentStatus(name string, status AgentStatus) string {
	var style lipgloss.Style
	var icon string
	
	switch status {
	case StatusPending:
		style = pendingStyle
		icon = "⏳"
	case StatusInProgress:
		style = inProgressStyle
		icon = "🔄"
	case StatusCompleted:
		style = completedStyle
		icon = "✅"
	case StatusError:
		style = errorStyle
		icon = "❌"
	default:
		style = pendingStyle
		icon = "❓"
	}
	
	return fmt.Sprintf("%s %s %s\n", icon, style.Render(name), style.Render(string(status)))
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// UpdateDisplay refreshes the entire display
func UpdateDisplay(session *AnalysisSession) {
	// Clear screen for better user experience
	if os.Getenv("CORTEXGO_NO_CLEAR") == "" {
		ClearScreen()
	}
	
	// Display header
	DisplayAnalysisHeader(session)
	
	// Display progress panel
	DisplayProgressPanel(&session.Progress, &session.Stats)
	
	// Display messages panel (show last 10 messages)
	DisplayMessagesPanel(session.Messages, 10)
	
	// Display reports panel
	DisplayReportsPanel(session.Reports)
}

// DisplayInitializing shows an initializing message
func DisplayInitializing() {
	fmt.Println(inProgressStyle.Render("🚀 Initializing trading analysis system..."))
}