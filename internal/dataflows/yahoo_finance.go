package dataflows

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/piquette/finance-go/chart"
	"github.com/piquette/finance-go/datetime"
	"github.com/piquette/finance-go/quote"
	"github.com/shopspring/decimal"
)

// YahooFinanceClient handles Yahoo Finance data operations
type YahooFinanceClient struct {
	cache *CacheManager
}

// NewYahooFinanceClient creates a new Yahoo Finance client
func NewYahooFinanceClient(config *Config) *YahooFinanceClient {
	cacheDir := filepath.Join(config.DataCacheDir, "yahoo_finance")
	cache := NewCacheManager(cacheDir, 24*time.Hour, config.CacheEnabled) // 24 hour cache

	return &YahooFinanceClient{
		cache: cache,
	}
}

// GetQuote gets current quote data for a symbol
func (yf *YahooFinanceClient) GetQuote(symbol string) (*MarketData, error) {
	if err := ValidateSymbol(symbol); err != nil {
		return nil, err
	}

	symbol = NormalizeSymbol(symbol)

	// Check cache first
	var cached MarketData
	if yf.cache.Get("yahoo", "quote", symbol, &cached) {
		return &cached, nil
	}

	// Fetch from Yahoo Finance
	var result *MarketData
	err := WithRetry(DefaultRetryConfig(), func() error {
		q, err := quote.Get(symbol)
		if err != nil {
			return fmt.Errorf("failed to get quote for %s: %w", symbol, err)
		}

		result = &MarketData{
			Symbol:    symbol,
			Date:      time.Now(),
			Open:      decimal.NewFromFloat(q.RegularMarketOpen),
			High:      decimal.NewFromFloat(q.RegularMarketDayHigh),
			Low:       decimal.NewFromFloat(q.RegularMarketDayLow),
			Close:     decimal.NewFromFloat(q.RegularMarketPrice),
			AdjClose:  decimal.NewFromFloat(q.RegularMarketPrice),
			Volume:    int64(q.RegularMarketVolume),
			Timestamp: time.Now(),
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Cache the result
	yf.cache.Set("yahoo", "quote", symbol, result)

	return result, nil
}

// GetHistoricalData gets historical price data for a symbol
func (yf *YahooFinanceClient) GetHistoricalData(symbol string, start, end time.Time) ([]*MarketData, error) {
	if err := ValidateSymbol(symbol); err != nil {
		return nil, err
	}

	symbol = NormalizeSymbol(symbol)

	// Create cache key with date range
	cacheKey := map[string]interface{}{
		"symbol": symbol,
		"start":  start.Format("2006-01-02"),
		"end":    end.Format("2006-01-02"),
	}

	// Check cache first
	var cached []*MarketData
	if yf.cache.Get("yahoo", "historical", cacheKey, &cached) {
		return cached, nil
	}

	// Fetch from Yahoo Finance
	var result []*MarketData
	err := WithRetry(DefaultRetryConfig(), func() error {
		params := &chart.Params{
			Symbol:   symbol,
			Start:    datetime.New(&start),
			End:      datetime.New(&end),
			Interval: datetime.OneDay,
		}

		iter := chart.Get(params)

		result = make([]*MarketData, 0)
		for iter.Next() {
			bar := iter.Bar()

			marketData := &MarketData{
				Symbol:    symbol,
				Date:      time.Unix(int64(bar.Timestamp), 0),
				Open:      bar.Open,
				High:      bar.High,
				Low:       bar.Low,
				Close:     bar.Close,
				AdjClose:  bar.AdjClose,
				Volume:    int64(bar.Volume),
				Timestamp: time.Now(),
			}

			result = append(result, marketData)
		}

		if err := iter.Err(); err != nil {
			return fmt.Errorf("failed to get historical data for %s: %w", symbol, err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Cache the result
	yf.cache.Set("yahoo", "historical", cacheKey, result)

	// Note: File saving would need config passed from caller
	// This is handled at the interface level

	return result, nil
}

// GetHistoricalDataWindow gets historical data for a rolling window
func (yf *YahooFinanceClient) GetHistoricalDataWindow(symbol string, days int) ([]*MarketData, error) {
	end := time.Now()
	start := end.AddDate(0, 0, -days)

	return yf.GetHistoricalData(symbol, start, end)
}

// GetOfflineData loads historical data from cached files
func (yf *YahooFinanceClient) GetOfflineData(symbol string, start, end time.Time, config *Config) ([]*MarketData, error) {
	if err := ValidateSymbol(symbol); err != nil {
		return nil, err
	}

	symbol = NormalizeSymbol(symbol)

	// Try to load from file
	filePath := filepath.Join(config.DataDir, "market_data", "price_data",
		fmt.Sprintf("%s_%s_%s.json", symbol,
			start.Format("2006-01-02"), end.Format("2006-01-02")))

	var result []*MarketData
	if err := LoadDataFromFile(filePath, &result); err != nil {
		return nil, fmt.Errorf("offline data not available for %s (%s): %w",
			symbol, FormatDateRange(start, end), err)
	}

	return result, nil
}

// SearchSymbols searches for symbols matching a query (basic implementation)
func (yf *YahooFinanceClient) SearchSymbols(query string) ([]string, error) {
	query = strings.TrimSpace(strings.ToUpper(query))
	if len(query) == 0 {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	// This is a basic implementation - in production you might want to use
	// a more comprehensive symbol search API
	commonSymbols := []string{
		"AAPL", "MSFT", "GOOGL", "AMZN", "TSLA", "META", "NVDA", "NFLX",
		"AMD", "INTC", "CRM", "ORCL", "ADBE", "PYPL", "DIS", "V", "MA",
		"JPM", "BAC", "WFC", "C", "GS", "MS", "BRK.B", "JNJ", "PFE",
		"KO", "PEP", "WMT", "HD", "NKE", "MCD", "SBUX", "UNH", "CVX",
	}

	var matches []string
	for _, symbol := range commonSymbols {
		if strings.Contains(symbol, query) {
			matches = append(matches, symbol)
		}
	}

	return matches, nil
}

// GetCompanyInfo gets basic company information
func (yf *YahooFinanceClient) GetCompanyInfo(symbol string) (map[string]interface{}, error) {
	if err := ValidateSymbol(symbol); err != nil {
		return nil, err
	}

	symbol = NormalizeSymbol(symbol)

	// Check cache first
	var cached map[string]interface{}
	if yf.cache.Get("yahoo", "company_info", symbol, &cached) {
		return cached, nil
	}

	// Fetch quote to get basic info
	q, err := quote.Get(symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get company info for %s: %w", symbol, err)
	}

	info := map[string]interface{}{
		"symbol":               symbol,
		"company_name":         q.ShortName,
		"exchange":             q.FullExchangeName,
		"currency":             q.CurrencyID,
		"market_state":         q.MarketState,
		"regular_market_price": q.RegularMarketPrice,
		"regular_market_time":  q.RegularMarketTime,
		"quote_type":           q.QuoteType,
		"is_tradeable":         q.IsTradeable,
		"fetched_at":           time.Now(),
	}

	// Cache the result
	yf.cache.Set("yahoo", "company_info", symbol, info)

	return info, nil
}
