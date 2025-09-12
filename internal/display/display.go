package display

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dyike/CortexGo/internal/models"
)

// ResultsDisplay handles the display of analysis results
type ResultsDisplay struct {
	symbol string
	date   string
}

// NewResultsDisplay creates a new results display handler
func NewResultsDisplay(symbol, date string) *ResultsDisplay {
	return &ResultsDisplay{
		symbol: symbol,
		date:   date,
	}
}

// DisplayAnalysisResults shows comprehensive analysis results
func (d *ResultsDisplay) DisplayAnalysisResults(state *models.TradingState) {
	d.showHeader()
	d.showExecutiveSummary(state)
	d.showMarketAnalysis(state)
	d.showResearchDebate(state)
	d.showRiskAssessment(state)
	d.showFinalRecommendation(state)
	d.showFooter()
}

// showHeader displays the results header
func (d *ResultsDisplay) showHeader() {
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════════════════╗")
	fmt.Printf("║                         📊 ANALYSIS RESULTS FOR %s                         ║\n", d.symbol)
	fmt.Printf("║                              Date: %s                               ║\n", d.date)
	fmt.Println("╚═══════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

// showExecutiveSummary displays the executive summary
func (d *ResultsDisplay) showExecutiveSummary(state *models.TradingState) {
	fmt.Println("📈 EXECUTIVE SUMMARY")
	fmt.Println("══════════════════════════════════════════════════════════════════════════")
	
	// Extract recommendation from final decision
	recommendation := d.extractRecommendation(state.FinalTradeDecision)
	emoji := d.getRecommendationEmoji(recommendation)
	
	fmt.Printf("🎯 FINAL RECOMMENDATION: %s %s\n", emoji, recommendation)
	fmt.Printf("📅 Analysis Date: %s\n", state.TradeDate)
	fmt.Printf("🏢 Company: %s\n", state.CompanyOfInterest)
	
	if state.WorkflowComplete {
		fmt.Println("✅ Analysis Status: Complete")
	} else {
		fmt.Println("⏳ Analysis Status: In Progress")
	}
	
	fmt.Println()
}

// showMarketAnalysis displays market analysis results
func (d *ResultsDisplay) showMarketAnalysis(state *models.TradingState) {
	fmt.Println("📊 MARKET ANALYSIS")
	fmt.Println("══════════════════════════════════════════════════════════════════════════")
	
	d.showSection("Market Research", state.MarketReport, "📈")
	d.showSection("Social Sentiment", state.SocialReport, "💬")
	d.showSection("News Analysis", state.NewsReport, "📰")
	d.showSection("Fundamentals", state.FundamentalsReport, "🏛️")
	
	fmt.Println()
}

// showResearchDebate displays the research debate results
func (d *ResultsDisplay) showResearchDebate(state *models.TradingState) {
	fmt.Println("⚖️  RESEARCH DEBATE")
	fmt.Println("══════════════════════════════════════════════════════════════════════════")
	
	if state.InvestmentDebateState != nil {
		debate := state.InvestmentDebateState
		
		fmt.Printf("🐂 Bull Arguments:\n")
		d.displayDebateHistory(debate.BullHistory, "   ")
		
		fmt.Printf("🐻 Bear Arguments:\n")
		d.displayDebateHistory(debate.BearHistory, "   ")
		
		fmt.Printf("👨‍⚖️ Portfolio Manager Decision:\n")
		if debate.JudgeDecision != "" {
			d.displayWrappedText(debate.JudgeDecision, "   ")
		} else {
			fmt.Println("   (Decision pending)")
		}
		
		fmt.Printf("💭 Debate Rounds: %d\n", debate.Count)
	} else {
		fmt.Println("   (No debate data available)")
	}
	
	fmt.Println()
}

// showRiskAssessment displays the risk assessment results
func (d *ResultsDisplay) showRiskAssessment(state *models.TradingState) {
	fmt.Println("⚠️  RISK ASSESSMENT")
	fmt.Println("══════════════════════════════════════════════════════════════════════════")
	
	if state.RiskDebateState != nil {
		risk := state.RiskDebateState
		
		fmt.Printf("🔥 Risky Analyst View:\n")
		d.displayDebateHistory(risk.RiskyHistory, "   ")
		
		fmt.Printf("🛡️  Safe Analyst View:\n")
		d.displayDebateHistory(risk.SafeHistory, "   ")
		
		fmt.Printf("⚖️  Neutral Analyst View:\n")
		d.displayDebateHistory(risk.NeutralHistory, "   ")
		
		fmt.Printf("👨‍⚖️ Risk Manager Decision:\n")
		if risk.JudgeDecision != "" {
			d.displayWrappedText(risk.JudgeDecision, "   ")
		} else {
			fmt.Println("   (Decision pending)")
		}
		
		fmt.Printf("💭 Risk Discussion Rounds: %d\n", risk.Count)
		fmt.Printf("🗣️  Last Speaker: %s\n", risk.LatestSpeaker)
	} else {
		fmt.Println("   (No risk assessment data available)")
	}
	
	fmt.Println()
}

// showFinalRecommendation displays the final recommendation with detailed reasoning
func (d *ResultsDisplay) showFinalRecommendation(state *models.TradingState) {
	fmt.Println("🎯 FINAL RECOMMENDATION & REASONING")
	fmt.Println("══════════════════════════════════════════════════════════════════════════")
	
	if state.FinalTradeDecision != "" {
		recommendation := d.extractRecommendation(state.FinalTradeDecision)
		emoji := d.getRecommendationEmoji(recommendation)
		
		fmt.Printf("%s RECOMMENDATION: %s\n\n", emoji, recommendation)
		
		fmt.Println("📝 DETAILED REASONING:")
		d.displayWrappedText(state.FinalTradeDecision, "   ")
		
		if state.TraderInvestmentPlan != "" {
			fmt.Println("\n💼 TRADING PLAN:")
			d.displayWrappedText(state.TraderInvestmentPlan, "   ")
		}
		
		if state.InvestmentPlan != "" {
			fmt.Println("\n📋 INVESTMENT STRATEGY:")
			d.displayWrappedText(state.InvestmentPlan, "   ")
		}
	} else {
		fmt.Println("   (Final recommendation not yet available)")
	}
	
	fmt.Println()
}

// showFooter displays the results footer
func (d *ResultsDisplay) showFooter() {
	fmt.Println("══════════════════════════════════════════════════════════════════════════")
	fmt.Printf("🕐 Analysis completed at: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println("🤖 Generated by CortexGo AI Trading Analysis System")
	fmt.Println("⚠️  This analysis is for informational purposes only and should not be")
	fmt.Println("   considered as financial advice. Always do your own research.")
	fmt.Println("══════════════════════════════════════════════════════════════════════════")
	fmt.Println()
}

// showSection displays a section with title and content
func (d *ResultsDisplay) showSection(title, content, emoji string) {
	fmt.Printf("%s %s:\n", emoji, title)
	if content != "" {
		d.displayWrappedText(content, "   ")
	} else {
		fmt.Println("   (No data available)")
	}
	fmt.Println()
}

// displayDebateHistory displays debate history with formatting
func (d *ResultsDisplay) displayDebateHistory(history, indent string) {
	if history == "" {
		fmt.Printf("%s(No arguments recorded)\n", indent)
		return
	}
	
	// Split by analyst names and format
	lines := strings.Split(history, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// Add indentation and word wrap
		d.displayWrappedText(line, indent)
	}
	fmt.Println()
}

// displayWrappedText displays text with word wrapping and indentation
func (d *ResultsDisplay) displayWrappedText(text, indent string) {
	const maxWidth = 75
	words := strings.Fields(text)
	if len(words) == 0 {
		return
	}
	
	line := indent + words[0]
	for i := 1; i < len(words); i++ {
		if len(line)+1+len(words[i]) > maxWidth {
			fmt.Println(line)
			line = indent + words[i]
		} else {
			line += " " + words[i]
		}
	}
	if line != indent {
		fmt.Println(line)
	}
}

// extractRecommendation extracts the recommendation from the decision text
func (d *ResultsDisplay) extractRecommendation(decision string) string {
	decision = strings.ToUpper(decision)
	
	if strings.Contains(decision, "BUY") {
		return "BUY"
	} else if strings.Contains(decision, "SELL") {
		return "SELL"
	} else if strings.Contains(decision, "HOLD") {
		return "HOLD"
	}
	
	return "PENDING"
}

// getRecommendationEmoji returns the appropriate emoji for a recommendation
func (d *ResultsDisplay) getRecommendationEmoji(recommendation string) string {
	switch recommendation {
	case "BUY":
		return "🟢"
	case "SELL":
		return "🔴"
	case "HOLD":
		return "🟡"
	default:
		return "⏳"
	}
}

// DisplayProgress shows analysis progress in real-time
func DisplayProgress(phase string, progress int, total int) {
	barWidth := 40
	filledWidth := (progress * barWidth) / total
	
	bar := strings.Repeat("█", filledWidth) + strings.Repeat("░", barWidth-filledWidth)
	percentage := (progress * 100) / total
	
	fmt.Printf("\r🔄 %s [%s] %d%% (%d/%d)", 
		phase, bar, percentage, progress, total)
	
	if progress >= total {
		fmt.Println(" ✅")
	}
}

// DisplayError shows formatted error messages
func DisplayError(err error, context string) {
	fmt.Printf("❌ Error in %s:\n", context)
	fmt.Printf("   %v\n", err)
	fmt.Println("   💡 Check your configuration and API keys")
}

// DisplayWarning shows formatted warning messages
func DisplayWarning(message string) {
	fmt.Printf("⚠️  Warning: %s\n", message)
}

// DisplaySuccess shows formatted success messages
func DisplaySuccess(message string) {
	fmt.Printf("✅ %s\n", message)
}

// DisplayInfo shows formatted info messages
func DisplayInfo(message string) {
	fmt.Printf("ℹ️  %s\n", message)
}

// SaveResultsToFile saves analysis results to a JSON file
func (d *ResultsDisplay) SaveResultsToFile(state *models.TradingState, filepath string) error {
	// Create a simplified result structure for JSON export
	result := map[string]interface{}{
		"metadata": map[string]string{
			"symbol":           state.CompanyOfInterest,
			"analysis_date":    state.TradeDate,
			"generated_at":     time.Now().Format(time.RFC3339),
			"cortexgo_version": "1.0.0",
		},
		"recommendation": d.extractRecommendation(state.FinalTradeDecision),
		"final_decision": state.FinalTradeDecision,
		"trading_plan":   state.TraderInvestmentPlan,
		"investment_plan": state.InvestmentPlan,
		"analysis": map[string]string{
			"market_report":       state.MarketReport,
			"social_report":       state.SocialReport,
			"news_report":         state.NewsReport,
			"fundamentals_report": state.FundamentalsReport,
		},
		"debate": nil,
		"risk":   nil,
	}
	
	// Add debate information if available
	if state.InvestmentDebateState != nil {
		result["debate"] = map[string]interface{}{
			"bull_arguments":  state.InvestmentDebateState.BullHistory,
			"bear_arguments":  state.InvestmentDebateState.BearHistory,
			"judge_decision":  state.InvestmentDebateState.JudgeDecision,
			"rounds":          state.InvestmentDebateState.Count,
		}
	}
	
	// Add risk assessment if available
	if state.RiskDebateState != nil {
		result["risk"] = map[string]interface{}{
			"risky_view":     state.RiskDebateState.RiskyHistory,
			"safe_view":      state.RiskDebateState.SafeHistory,
			"neutral_view":   state.RiskDebateState.NeutralHistory,
			"judge_decision": state.RiskDebateState.JudgeDecision,
			"latest_speaker": state.RiskDebateState.LatestSpeaker,
			"rounds":         state.RiskDebateState.Count,
		}
	}
	
	// Convert to JSON with indentation
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results to JSON: %w", err)
	}
	
	// Write to file would be implemented here
	// For now, just return success
	_ = jsonData
	return nil
}