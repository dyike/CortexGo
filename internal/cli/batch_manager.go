package cli

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dyike/CortexGo/config"
	"github.com/dyike/CortexGo/internal/display"
)

// BatchManager handles batch analysis operations
type BatchManager struct {
	config *config.Config
}

// BatchProgress tracks progress of batch analysis
type BatchProgress struct {
	Total       int
	Completed   int
	Failed      int
	InProgress  int
	Results     []BatchResult
	StartTime   time.Time
	mutex       sync.RWMutex
}

// BatchResult represents the result of a single analysis in batch
type BatchResult struct {
	Symbol    string
	Date      string
	Status    BatchStatus
	Error     string
	Duration  time.Duration
	StartTime time.Time
	EndTime   time.Time
}

// BatchStatus represents the status of batch analysis item
type BatchStatus int

const (
	BatchPending BatchStatus = iota
	BatchRunning
	BatchCompleted
	BatchFailed
)

// String returns string representation of BatchStatus
func (bs BatchStatus) String() string {
	switch bs {
	case BatchPending:
		return "⏳ Pending"
	case BatchRunning:
		return "🔄 Running"
	case BatchCompleted:
		return "✅ Completed"
	case BatchFailed:
		return "❌ Failed"
	default:
		return "❓ Unknown"
	}
}

// NewBatchManager creates a new batch manager
func NewBatchManager(cfg *config.Config) *BatchManager {
	return &BatchManager{
		config: cfg,
	}
}

// RunBatchAnalysis runs analysis on multiple symbols
func (bm *BatchManager) RunBatchAnalysis(symbols []string, date string, concurrent int) error {
	if len(symbols) == 0 {
		return fmt.Errorf("no symbols provided for batch analysis")
	}

	// Set default date if not provided
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	// Validate date format
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return fmt.Errorf("invalid date format: %s (use YYYY-MM-DD)", date)
	}

	// Validate concurrent limit
	if concurrent <= 0 || concurrent > 10 {
		concurrent = 3 // Default to 3 concurrent analyses
	}

	display.DisplayInfo(fmt.Sprintf("Starting batch analysis for %d symbols on %s", len(symbols), date))
	display.DisplayInfo(fmt.Sprintf("Concurrent analyses: %d", concurrent))

	// Initialize progress tracker
	progress := &BatchProgress{
		Total:     len(symbols),
		Results:   make([]BatchResult, len(symbols)),
		StartTime: time.Now(),
	}

	// Initialize results
	for i, symbol := range symbols {
		progress.Results[i] = BatchResult{
			Symbol: strings.ToUpper(symbol),
			Date:   date,
			Status: BatchPending,
		}
	}

	// Start progress display in background
	stopProgress := make(chan bool)
	go bm.displayBatchProgress(progress, stopProgress)

	// Create semaphore to limit concurrent operations
	semaphore := make(chan struct{}, concurrent)
	var wg sync.WaitGroup

	// Process each symbol
	for i := range progress.Results {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			semaphore <- struct{}{} // Acquire
			defer func() { <-semaphore }() // Release

			bm.processSingleAnalysis(progress, idx)
		}(i)
	}

	// Wait for all analyses to complete
	wg.Wait()
	close(stopProgress)

	// Display final results
	bm.displayBatchSummary(progress)

	return nil
}

// processSingleAnalysis processes a single symbol analysis
func (bm *BatchManager) processSingleAnalysis(progress *BatchProgress, idx int) {
	progress.mutex.Lock()
	progress.Results[idx].Status = BatchRunning
	progress.Results[idx].StartTime = time.Now()
	progress.InProgress++
	progress.mutex.Unlock()

	symbol := progress.Results[idx].Symbol
	date := progress.Results[idx].Date

	// Simulate analysis - replace with actual trading session execution
	// For now, we'll simulate the analysis with some processing time
	duration := time.Duration(2000+idx*500) * time.Millisecond
	time.Sleep(duration)

	// Simulate success/failure (90% success rate for demo)
	success := (idx%10 != 9) // 9 out of 10 succeed

	progress.mutex.Lock()
	defer progress.mutex.Unlock()

	progress.Results[idx].EndTime = time.Now()
	progress.Results[idx].Duration = progress.Results[idx].EndTime.Sub(progress.Results[idx].StartTime)
	progress.InProgress--

	if success {
		progress.Results[idx].Status = BatchCompleted
		progress.Completed++
		
		// Create mock result file for demonstration
		bm.createMockResultFile(symbol, date)
	} else {
		progress.Results[idx].Status = BatchFailed
		progress.Results[idx].Error = "Analysis timeout or API limit reached"
		progress.Failed++
	}
}

// createMockResultFile creates a mock result file for demonstration
func (bm *BatchManager) createMockResultFile(symbol, date string) {
	filename := fmt.Sprintf("%s/%s_%s_analysis.json", bm.config.ResultsDir, symbol, date)
	
	// Ensure results directory exists
	os.MkdirAll(bm.config.ResultsDir, 0755)
	
	mockResult := fmt.Sprintf(`{
  "metadata": {
    "symbol": "%s",
    "analysis_date": "%s",
    "generated_at": "%s",
    "cortexgo_version": "1.0.0"
  },
  "recommendation": "HOLD",
  "confidence": 0.75,
  "final_decision": "Based on batch analysis, holding position is recommended for %s",
  "analysis": {
    "market_report": "Technical indicators show neutral trend",
    "social_report": "Mixed sentiment on social media",
    "news_report": "No significant news impact",
    "fundamentals_report": "Stable fundamentals"
  }
}`, symbol, date, time.Now().Format(time.RFC3339), symbol)

	ioutil.WriteFile(filename, []byte(mockResult), 0644)
}

// displayBatchProgress shows real-time progress of batch analysis
func (bm *BatchManager) displayBatchProgress(progress *BatchProgress, stop chan bool) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			bm.printProgressUpdate(progress)
		}
	}
}

// printProgressUpdate prints current progress
func (bm *BatchManager) printProgressUpdate(progress *BatchProgress) {
	progress.mutex.RLock()
	defer progress.mutex.RUnlock()

	// Clear previous line and print update
	fmt.Print("\r\033[K") // Clear line
	
	elapsed := time.Since(progress.StartTime)
	completionRate := float64(progress.Completed) / float64(progress.Total)
	
	var eta time.Duration
	if completionRate > 0 {
		eta = time.Duration(float64(elapsed) / completionRate) - elapsed
	}

	fmt.Printf("📊 Progress: %d/%d completed, %d running, %d failed | Elapsed: %s | ETA: %s",
		progress.Completed, progress.Total, progress.InProgress, progress.Failed,
		elapsed.Round(time.Second), eta.Round(time.Second))
}

// displayBatchSummary displays final batch analysis summary
func (bm *BatchManager) displayBatchSummary(progress *BatchProgress) {
	fmt.Println() // New line after progress updates
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    BATCH ANALYSIS SUMMARY                     ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	totalDuration := time.Since(progress.StartTime)
	
	display.DisplayInfo(fmt.Sprintf("Total Symbols: %d", progress.Total))
	display.DisplaySuccess(fmt.Sprintf("Completed: %d", progress.Completed))
	if progress.Failed > 0 {
		display.DisplayError(fmt.Errorf("failed: %d", progress.Failed), "batch analysis")
	}
	display.DisplayInfo(fmt.Sprintf("Total Time: %s", totalDuration.Round(time.Second)))
	
	if progress.Completed > 0 {
		avgDuration := totalDuration / time.Duration(progress.Completed)
		display.DisplayInfo(fmt.Sprintf("Average Time per Analysis: %s", avgDuration.Round(time.Second)))
	}

	fmt.Println()
	fmt.Println("📋 DETAILED RESULTS:")
	fmt.Println("═══════════════════")

	// Display results in table format
	fmt.Printf("%-10s %-12s %-15s %-10s %s\n", "SYMBOL", "DATE", "STATUS", "DURATION", "ERROR")
	fmt.Println(strings.Repeat("─", 70))

	for _, result := range progress.Results {
		duration := "N/A"
		if result.Duration > 0 {
			duration = result.Duration.Round(time.Second).String()
		}

		errorMsg := ""
		if result.Error != "" {
			errorMsg = result.Error
			if len(errorMsg) > 25 {
				errorMsg = errorMsg[:22] + "..."
			}
		}

		fmt.Printf("%-10s %-12s %-15s %-10s %s\n",
			result.Symbol, result.Date, result.Status, duration, errorMsg)
	}

	fmt.Println()
	if progress.Completed > 0 {
		display.DisplaySuccess(fmt.Sprintf("Results saved in: %s", bm.config.ResultsDir))
		fmt.Println("💡 Use 'cortexgo results list' to view all results")
		fmt.Println("💡 Use 'cortexgo results show <SYMBOL> <DATE>' to view detailed analysis")
	}
}

// LoadSymbolsFromFile loads symbols from a text file (one symbol per line)
func (bm *BatchManager) LoadSymbolsFromFile(filename string) ([]string, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read symbols file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	var symbols []string

	for _, line := range lines {
		symbol := strings.TrimSpace(strings.ToUpper(line))
		if symbol != "" && !strings.HasPrefix(symbol, "#") { // Skip empty lines and comments
			symbols = append(symbols, symbol)
		}
	}

	if len(symbols) == 0 {
		return nil, fmt.Errorf("no valid symbols found in file: %s", filename)
	}

	return symbols, nil
}

// ValidateSymbols performs basic validation on symbol list
func (bm *BatchManager) ValidateSymbols(symbols []string) ([]string, []string) {
	var valid []string
	var invalid []string

	for _, symbol := range symbols {
		symbol = strings.TrimSpace(strings.ToUpper(symbol))
		
		// Basic validation: 1-5 characters, letters only
		if len(symbol) >= 1 && len(symbol) <= 5 && isAlpha(symbol) {
			valid = append(valid, symbol)
		} else {
			invalid = append(invalid, symbol)
		}
	}

	return valid, invalid
}

// isAlpha checks if string contains only alphabetic characters
func isAlpha(s string) bool {
	for _, r := range s {
		if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') {
			return false
		}
	}
	return true
}

// EstimateBatchDuration estimates total duration for batch analysis
func (bm *BatchManager) EstimateBatchDuration(symbolCount, concurrent int) time.Duration {
	// Estimate based on average analysis time of 3 minutes per symbol
	avgAnalysisTime := 3 * time.Minute
	
	// Calculate parallel execution time
	totalBatches := (symbolCount + concurrent - 1) / concurrent
	estimatedDuration := time.Duration(totalBatches) * avgAnalysisTime
	
	return estimatedDuration
}