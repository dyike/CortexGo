package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dyike/CortexGo/config"
	"github.com/dyike/CortexGo/internal/display"
)

// ResultsManager handles analysis results management
type ResultsManager struct {
	config     *config.Config
	resultsDir string
}

// ResultSummary represents a summary of an analysis result
type ResultSummary struct {
	Symbol        string    `json:"symbol"`
	Date          string    `json:"date"`
	Recommendation string   `json:"recommendation"`
	CreatedAt     time.Time `json:"created_at"`
	FilePath      string    `json:"file_path"`
	FileSize      int64     `json:"file_size"`
}

// NewResultsManager creates a new results manager
func NewResultsManager(cfg *config.Config) *ResultsManager {
	return &ResultsManager{
		config:     cfg,
		resultsDir: cfg.ResultsDir,
	}
}

// ListResults lists all available analysis results
func (rm *ResultsManager) ListResults(sortBy string, reverse bool) ([]ResultSummary, error) {
	var results []ResultSummary

	// Ensure results directory exists
	if err := os.MkdirAll(rm.resultsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create results directory: %w", err)
	}

	// Walk through results directory
	err := filepath.WalkDir(rm.resultsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-JSON files
		if d.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}

		// Parse filename for symbol and date
		filename := filepath.Base(path)
		if !strings.HasSuffix(filename, "_analysis.json") {
			return nil
		}

		// Extract symbol and date from filename: SYMBOL_DATE_analysis.json
		parts := strings.Split(strings.TrimSuffix(filename, "_analysis.json"), "_")
		if len(parts) < 2 {
			return nil // Skip malformed filenames
		}

		symbol := strings.Join(parts[:len(parts)-1], "_")
		dateStr := parts[len(parts)-1]

		// Get file info
		info, err := d.Info()
		if err != nil {
			return nil
		}

		// Try to read the file to get recommendation
		recommendation := "Unknown"
		if data, err := ioutil.ReadFile(path); err == nil {
			var result map[string]interface{}
			if json.Unmarshal(data, &result) == nil {
				if rec, ok := result["recommendation"].(string); ok {
					recommendation = rec
				}
			}
		}

		results = append(results, ResultSummary{
			Symbol:        symbol,
			Date:          dateStr,
			Recommendation: recommendation,
			CreatedAt:     info.ModTime(),
			FilePath:      path,
			FileSize:      info.Size(),
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan results directory: %w", err)
	}

	// Sort results
	rm.sortResults(results, sortBy, reverse)

	return results, nil
}

// sortResults sorts results by the specified criteria
func (rm *ResultsManager) sortResults(results []ResultSummary, sortBy string, reverse bool) {
	switch strings.ToLower(sortBy) {
	case "date", "created":
		sort.Slice(results, func(i, j int) bool {
			if reverse {
				return results[i].CreatedAt.After(results[j].CreatedAt)
			}
			return results[i].CreatedAt.Before(results[j].CreatedAt)
		})
	case "symbol":
		sort.Slice(results, func(i, j int) bool {
			if reverse {
				return results[i].Symbol > results[j].Symbol
			}
			return results[i].Symbol < results[j].Symbol
		})
	case "recommendation", "rec":
		sort.Slice(results, func(i, j int) bool {
			if reverse {
				return results[i].Recommendation > results[j].Recommendation
			}
			return results[i].Recommendation < results[j].Recommendation
		})
	case "size":
		sort.Slice(results, func(i, j int) bool {
			if reverse {
				return results[i].FileSize > results[j].FileSize
			}
			return results[i].FileSize < results[j].FileSize
		})
	default:
		// Default sort by creation date (newest first)
		sort.Slice(results, func(i, j int) bool {
			return results[i].CreatedAt.After(results[j].CreatedAt)
		})
	}
}

// ShowResult displays a specific analysis result
func (rm *ResultsManager) ShowResult(symbol, date string) error {
	filename := fmt.Sprintf("%s/%s_%s_analysis.json", rm.resultsDir, symbol, date)

	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return fmt.Errorf("analysis result not found: %s on %s", symbol, date)
	}

	// Read and parse the JSON file
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read result file: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("failed to parse result file: %w", err)
	}

	// Display the result using our display system
	rm.displayResultSummary(result, symbol, date)

	return nil
}

// displayResultSummary displays a formatted summary of the analysis result
func (rm *ResultsManager) displayResultSummary(result map[string]interface{}, symbol, date string) {
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Printf("║                ANALYSIS RESULT: %s on %s                ║\n", symbol, date)
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Show metadata
	if metadata, ok := result["metadata"].(map[string]interface{}); ok {
		display.DisplayInfo("Analysis Information:")
		if generatedAt, ok := metadata["generated_at"].(string); ok {
			fmt.Printf("  📅 Generated At: %s\n", generatedAt)
		}
		if version, ok := metadata["cortexgo_version"].(string); ok {
			fmt.Printf("  🔧 Version: %s\n", version)
		}
		fmt.Println()
	}

	// Show recommendation
	if rec, ok := result["recommendation"].(string); ok {
		emoji := getRecommendationEmoji(rec)
		fmt.Printf("🎯 RECOMMENDATION: %s %s\n\n", emoji, rec)
	}

	// Show final decision
	if decision, ok := result["final_decision"].(string); ok {
		fmt.Println("📝 FINAL DECISION:")
		displayWrappedText(decision, "   ")
		fmt.Println()
	}

	// Show analysis sections
	if analysis, ok := result["analysis"].(map[string]interface{}); ok {
		fmt.Println("📊 ANALYSIS OVERVIEW:")
		fmt.Println("═══════════════════════════════════════")
		
		sections := []struct {
			key   string
			emoji string
			title string
		}{
			{"market_report", "📈", "Market Analysis"},
			{"social_report", "💬", "Social Sentiment"},
			{"news_report", "📰", "News Analysis"},
			{"fundamentals_report", "🏛️", "Fundamentals"},
		}

		for _, section := range sections {
			if content, ok := analysis[section.key].(string); ok && content != "" {
				fmt.Printf("\n%s %s:\n", section.emoji, section.title)
				preview := rm.getTextPreview(content, 200)
				displayWrappedText(preview, "   ")
				if len(content) > 200 {
					fmt.Println("   ...")
				}
			}
		}
		fmt.Println()
	}

	// Show debate summary if available
	if debate, ok := result["debate"].(map[string]interface{}); ok {
		fmt.Println("⚖️  DEBATE SUMMARY:")
		fmt.Println("═══════════════════")
		if rounds, ok := debate["rounds"].(float64); ok {
			fmt.Printf("   💭 Debate Rounds: %.0f\n", rounds)
		}
		if decision, ok := debate["judge_decision"].(string); ok && decision != "" {
			fmt.Println("   👨‍⚖️ Judge Decision:")
			preview := rm.getTextPreview(decision, 150)
			displayWrappedText(preview, "      ")
			if len(decision) > 150 {
				fmt.Println("      ...")
			}
		}
		fmt.Println()
	}

	// Show risk assessment summary if available
	if risk, ok := result["risk"].(map[string]interface{}); ok {
		fmt.Println("⚠️  RISK ASSESSMENT:")
		fmt.Println("══════════════════")
		if rounds, ok := risk["rounds"].(float64); ok {
			fmt.Printf("   💭 Risk Rounds: %.0f\n", rounds)
		}
		if speaker, ok := risk["latest_speaker"].(string); ok {
			fmt.Printf("   🗣️  Last Speaker: %s\n", speaker)
		}
		fmt.Println()
	}

	fmt.Println("💡 Use 'cortexgo results export %s %s json' for full details", symbol, date)
}

// getTextPreview returns a preview of text content
func (rm *ResultsManager) getTextPreview(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength] + "..."
}

// DeleteResult deletes a specific analysis result
func (rm *ResultsManager) DeleteResult(symbol, date string) error {
	filename := fmt.Sprintf("%s/%s_%s_analysis.json", rm.resultsDir, symbol, date)

	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return fmt.Errorf("analysis result not found: %s on %s", symbol, date)
	}

	// Delete the file
	if err := os.Remove(filename); err != nil {
		return fmt.Errorf("failed to delete result file: %w", err)
	}

	display.DisplaySuccess(fmt.Sprintf("Deleted analysis result for %s on %s", symbol, date))
	return nil
}

// ExportResults exports results in different formats
func (rm *ResultsManager) ExportResults(symbol, date, format string) error {
	inputFile := fmt.Sprintf("%s/%s_%s_analysis.json", rm.resultsDir, symbol, date)

	// Check if source file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return fmt.Errorf("analysis result not found: %s on %s", symbol, date)
	}

	// Read source data
	data, err := ioutil.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("failed to parse source file: %w", err)
	}

	outputFile := fmt.Sprintf("%s/%s_%s_analysis.%s", rm.resultsDir, symbol, date, format)

	switch strings.ToLower(format) {
	case "json":
		// Pretty print JSON
		prettyJSON, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format JSON: %w", err)
		}
		return ioutil.WriteFile(outputFile, prettyJSON, 0644)

	case "csv":
		return rm.exportToCSV(result, outputFile)

	case "txt", "text":
		return rm.exportToText(result, outputFile, symbol, date)

	default:
		return fmt.Errorf("unsupported export format: %s. Supported: json, csv, txt", format)
	}
}

// exportToCSV exports result to CSV format
func (rm *ResultsManager) exportToCSV(result map[string]interface{}, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"Field", "Value"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Helper function to write a field
	writeField := func(key, value string) error {
		return writer.Write([]string{key, value})
	}

	// Write basic fields
	if rec, ok := result["recommendation"].(string); ok {
		writeField("Recommendation", rec)
	}
	if decision, ok := result["final_decision"].(string); ok {
		writeField("Final Decision", decision)
	}
	if plan, ok := result["trading_plan"].(string); ok {
		writeField("Trading Plan", plan)
	}

	// Write analysis data
	if analysis, ok := result["analysis"].(map[string]interface{}); ok {
		for key, value := range analysis {
			if str, ok := value.(string); ok {
				writeField(strings.Title(strings.ReplaceAll(key, "_", " ")), str)
			}
		}
	}

	return nil
}

// exportToText exports result to plain text format
func (rm *ResultsManager) exportToText(result map[string]interface{}, filename, symbol, date string) error {
	var content strings.Builder

	content.WriteString(fmt.Sprintf("CortexGo Analysis Report: %s on %s\n", symbol, date))
	content.WriteString(strings.Repeat("=", 60) + "\n\n")

	// Metadata
	if metadata, ok := result["metadata"].(map[string]interface{}); ok {
		content.WriteString("METADATA:\n")
		content.WriteString("---------\n")
		for key, value := range metadata {
			content.WriteString(fmt.Sprintf("%s: %v\n", strings.Title(strings.ReplaceAll(key, "_", " ")), value))
		}
		content.WriteString("\n")
	}

	// Recommendation
	if rec, ok := result["recommendation"].(string); ok {
		content.WriteString(fmt.Sprintf("RECOMMENDATION: %s\n\n", rec))
	}

	// Final decision
	if decision, ok := result["final_decision"].(string); ok {
		content.WriteString("FINAL DECISION:\n")
		content.WriteString("---------------\n")
		content.WriteString(decision + "\n\n")
	}

	// Trading plan
	if plan, ok := result["trading_plan"].(string); ok {
		content.WriteString("TRADING PLAN:\n")
		content.WriteString("-------------\n")
		content.WriteString(plan + "\n\n")
	}

	// Analysis sections
	if analysis, ok := result["analysis"].(map[string]interface{}); ok {
		content.WriteString("DETAILED ANALYSIS:\n")
		content.WriteString("==================\n\n")

		sections := []struct {
			key   string
			title string
		}{
			{"market_report", "Market Analysis"},
			{"social_report", "Social Sentiment"},
			{"news_report", "News Analysis"},
			{"fundamentals_report", "Fundamentals Analysis"},
		}

		for _, section := range sections {
			if str, ok := analysis[section.key].(string); ok && str != "" {
				content.WriteString(fmt.Sprintf("%s:\n", section.title))
				content.WriteString(strings.Repeat("-", len(section.title)+1) + "\n")
				content.WriteString(str + "\n\n")
			}
		}
	}

	return ioutil.WriteFile(filename, []byte(content.String()), 0644)
}

// CleanupResults removes old result files based on age or count
func (rm *ResultsManager) CleanupResults(maxAge time.Duration, maxCount int) error {
	results, err := rm.ListResults("date", true) // Sort by date, newest first
	if err != nil {
		return fmt.Errorf("failed to list results: %w", err)
	}

	var deletedCount int
	now := time.Now()

	for i, result := range results {
		shouldDelete := false

		// Check age limit
		if maxAge > 0 && now.Sub(result.CreatedAt) > maxAge {
			shouldDelete = true
		}

		// Check count limit
		if maxCount > 0 && i >= maxCount {
			shouldDelete = true
		}

		if shouldDelete {
			if err := os.Remove(result.FilePath); err != nil {
				display.DisplayWarning(fmt.Sprintf("Failed to delete %s: %v", result.FilePath, err))
			} else {
				deletedCount++
			}
		}
	}

	if deletedCount > 0 {
		display.DisplaySuccess(fmt.Sprintf("Cleaned up %d old result files", deletedCount))
	} else {
		display.DisplayInfo("No result files needed cleanup")
	}

	return nil
}

// Helper functions

func getRecommendationEmoji(recommendation string) string {
	switch strings.ToUpper(recommendation) {
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

func displayWrappedText(text, indent string) {
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