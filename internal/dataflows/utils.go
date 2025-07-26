package dataflows

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CacheManager handles file-based caching for data
type CacheManager struct {
	cacheDir     string
	ttl          time.Duration
	cacheEnabled bool
}

// NewCacheManager creates a new cache manager
func NewCacheManager(cacheDir string, ttl time.Duration, cacheEnabled bool) *CacheManager {
	return &CacheManager{
		cacheDir:     cacheDir,
		ttl:          ttl,
		cacheEnabled: cacheEnabled,
	}
}

// getCacheKey generates a cache key from parameters
func (cm *CacheManager) getCacheKey(source, method string, params interface{}) string {
	data, _ := json.Marshal(params)
	hash := md5.Sum(data)
	return fmt.Sprintf("%s_%s_%x.json", source, method, hash)
}

// Get retrieves data from cache if not expired
func (cm *CacheManager) Get(source, method string, params interface{}, result interface{}) bool {
	if !cm.cacheEnabled {
		return false
	}

	key := cm.getCacheKey(source, method, params)
	filePath := filepath.Join(cm.cacheDir, key)

	// Check if file exists and is not expired
	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}

	if time.Since(info.ModTime()) > cm.ttl {
		os.Remove(filePath) // Remove expired cache
		return false
	}

	// Read and unmarshal cached data
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	return json.Unmarshal(data, result) == nil
}

// Set stores data in cache
func (cm *CacheManager) Set(source, method string, params interface{}, data interface{}) error {
	if !cm.cacheEnabled {
		return nil
	}

	key := cm.getCacheKey(source, method, params)
	filePath := filepath.Join(cm.cacheDir, key)

	// Ensure cache directory exists
	if err := os.MkdirAll(cm.cacheDir, 0755); err != nil {
		return err
	}

	// Marshal and write data
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, jsonData, 0644)
}

// RetryConfig configures retry behavior
type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	Multiplier float64
}

// DefaultRetryConfig returns sensible retry defaults
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries: 3,
		BaseDelay:  1 * time.Second,
		MaxDelay:   30 * time.Second,
		Multiplier: 2.0,
	}
}

// WithRetry executes a function with exponential backoff retry
func WithRetry(config *RetryConfig, fn func() error) error {
	var lastErr error
	
	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(float64(config.BaseDelay) * 
				pow(config.Multiplier, float64(attempt-1)))
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}
			time.Sleep(delay)
		}
		
		if err := fn(); err != nil {
			lastErr = err
			continue
		}
		
		return nil
	}
	
	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// pow is a simple power function for floats
func pow(base, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}

// ValidateSymbol checks if a stock symbol is valid format
func ValidateSymbol(symbol string) error {
	symbol = strings.TrimSpace(strings.ToUpper(symbol))
	if len(symbol) == 0 {
		return fmt.Errorf("symbol cannot be empty")
	}
	if len(symbol) > 10 {
		return fmt.Errorf("symbol too long: %s", symbol)
	}
	return nil
}

// NormalizeSymbol converts symbol to standard format
func NormalizeSymbol(symbol string) string {
	return strings.TrimSpace(strings.ToUpper(symbol))
}

// FormatDateRange creates a human-readable date range string
func FormatDateRange(start, end time.Time) string {
	return fmt.Sprintf("%s to %s", 
		start.Format("2006-01-02"), 
		end.Format("2006-01-02"))
}

// ParseDateString parses common date formats
func ParseDateString(dateStr string) (time.Time, error) {
	formats := []string{
		"2006-01-02",
		"2006-01-02 15:04:05",
		"01/02/2006",
		"01-02-2006",
		time.RFC3339,
	}
	
	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}
	
	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// SaveDataToFile saves structured data to a JSON file
func SaveDataToFile(data interface{}, filePath string) error {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, jsonData, 0644)
}

// LoadDataFromFile loads structured data from a JSON file
func LoadDataFromFile(filePath string, result interface{}) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	
	return json.Unmarshal(data, result)
}

// FileExists checks if a file exists
func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}