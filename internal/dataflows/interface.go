package dataflows

import (
	"fmt"
	"time"
)

// DataFlowInterface provides high-level access to all data sources
type DataFlowInterface struct {
	yahooFinance *YahooFinanceClient
	finnhub      *FinnhubClient
	newsScraper  *NewsScraperClient
	config       *Config
}

// NewDataFlowInterface creates a new data flow interface
func NewDataFlowInterface(config *Config) *DataFlowInterface {
	return &DataFlowInterface{
		yahooFinance: NewYahooFinanceClient(config),
		finnhub:      NewFinnhubClient(config),
		newsScraper:  NewNewsScraperClient(config),
		config:       config,
	}
}

// Market Data Functions

// GetYFinData gets historical Yahoo Finance data (offline first)
func (dfi *DataFlowInterface) GetYFinData(symbol string, start, end time.Time) ([]*MarketData, error) {
	// Try offline first
	if data, err := dfi.yahooFinance.GetOfflineData(symbol, start, end, dfi.config); err == nil {
		return data, nil
	}

	// If not available offline and online tools are enabled, fetch online
	if dfi.config.OnlineTools {
		return dfi.yahooFinance.GetHistoricalData(symbol, start, end)
	}

	return nil, fmt.Errorf("offline data not available for %s and online tools disabled", symbol)
}

// GetYFinDataOnline gets real-time Yahoo Finance data
func (dfi *DataFlowInterface) GetYFinDataOnline(symbol string) (*MarketData, error) {
	if !dfi.config.OnlineTools {
		return nil, fmt.Errorf("online tools are disabled")
	}

	return dfi.yahooFinance.GetQuote(symbol)
}

// GetYFinDataWindow gets Yahoo Finance data for a rolling window
func (dfi *DataFlowInterface) GetYFinDataWindow(symbol string, days int) ([]*MarketData, error) {
	if !dfi.config.OnlineTools {
		// Try to get offline data for the approximate window
		end := time.Now()
		start := end.AddDate(0, 0, -days)
		return dfi.yahooFinance.GetOfflineData(symbol, start, end, dfi.config)
	}

	return dfi.yahooFinance.GetHistoricalDataWindow(symbol, days)
}

// GetCompanyInfo gets basic company information
func (dfi *DataFlowInterface) GetCompanyInfo(symbol string) (map[string]interface{}, error) {
	if !dfi.config.OnlineTools {
		return nil, fmt.Errorf("online tools are disabled")
	}

	return dfi.yahooFinance.GetCompanyInfo(symbol)
}

// News Functions

// GetFinnhubNews gets company news from Finnhub
func (dfi *DataFlowInterface) GetFinnhubNews(symbol string, from, to time.Time) ([]*NewsArticle, error) {
	if !dfi.config.OnlineTools {
		return nil, fmt.Errorf("online tools are disabled")
	}

	return dfi.finnhub.GetCompanyNews(symbol, from, to, dfi.config)
}

// GetFinnhubGeneralNews gets general market news from Finnhub
func (dfi *DataFlowInterface) GetFinnhubGeneralNews(category string) ([]*NewsArticle, error) {
	if !dfi.config.OnlineTools {
		return nil, fmt.Errorf("online tools are disabled")
	}

	return dfi.finnhub.GetGeneralNews(category)
}

// GetGoogleNews searches Google News for articles
func (dfi *DataFlowInterface) GetGoogleNews(query string, startDate, endDate time.Time, maxResults int) ([]*NewsArticle, error) {
	if !dfi.config.OnlineTools {
		return nil, fmt.Errorf("online tools are disabled")
	}

	params := GoogleNewsParams{
		Query:      query,
		StartDate:  startDate,
		EndDate:    endDate,
		MaxResults: maxResults,
	}

	return dfi.newsScraper.GetGoogleNews(params, dfi.config)
}

// GetGlobalNews gets general global news
func (dfi *DataFlowInterface) GetGlobalNews() ([]*NewsArticle, error) {
	if !dfi.config.OnlineTools {
		return nil, fmt.Errorf("online tools are disabled")
	}

	return dfi.finnhub.GetGeneralNews("general")
}

// Insider Trading Functions

// GetFinnhubCompanyInsiderSentiment gets insider sentiment for a company
func (dfi *DataFlowInterface) GetFinnhubCompanyInsiderSentiment(symbol string, from, to time.Time) ([]*InsiderSentiment, error) {
	if !dfi.config.OnlineTools {
		return nil, fmt.Errorf("online tools are disabled")
	}

	return dfi.finnhub.GetInsiderSentiment(symbol, from, to)
}

// GetFinnhubCompanyInsiderTransactions gets insider transactions for a company
func (dfi *DataFlowInterface) GetFinnhubCompanyInsiderTransactions(symbol string, from, to time.Time) ([]*InsiderTransaction, error) {
	if !dfi.config.OnlineTools {
		return nil, fmt.Errorf("online tools are disabled")
	}

	return dfi.finnhub.GetInsiderTransactions(symbol, from, to)
}

// Utility Functions

// GetNewsFromURL scrapes a specific news article URL
func (dfi *DataFlowInterface) GetNewsFromURL(url string) (*NewsArticle, error) {
	if !dfi.config.OnlineTools {
		return nil, fmt.Errorf("online tools are disabled")
	}

	return dfi.newsScraper.GetNewsFromURL(url)
}

// ValidateAndNormalizeSymbol validates and normalizes a stock symbol
func (dfi *DataFlowInterface) ValidateAndNormalizeSymbol(symbol string) (string, error) {
	if err := ValidateSymbol(symbol); err != nil {
		return "", err
	}
	return NormalizeSymbol(symbol), nil
}

// SearchSymbols searches for symbols matching a query
func (dfi *DataFlowInterface) SearchSymbols(query string) ([]string, error) {
	return dfi.yahooFinance.SearchSymbols(query)
}

// Convenience Functions with Default Parameters

// GetRecentNews gets recent news for a company (last 7 days)
func (dfi *DataFlowInterface) GetRecentNews(symbol string) ([]*NewsArticle, error) {
	end := time.Now()
	start := end.AddDate(0, 0, -7) // Last 7 days

	return dfi.GetFinnhubNews(symbol, start, end)
}

// GetRecentMarketData gets recent market data (last 30 days)
func (dfi *DataFlowInterface) GetRecentMarketData(symbol string) ([]*MarketData, error) {
	return dfi.GetYFinDataWindow(symbol, 30)
}

// GetRecentInsiderActivity gets recent insider activity (last 90 days)
func (dfi *DataFlowInterface) GetRecentInsiderActivity(symbol string) ([]*InsiderTransaction, error) {
	end := time.Now()
	start := end.AddDate(0, 0, -90) // Last 90 days

	return dfi.GetFinnhubCompanyInsiderTransactions(symbol, start, end)
}

// GetMarketOverview gets a comprehensive market overview for a symbol
func (dfi *DataFlowInterface) GetMarketOverview(symbol string) (map[string]interface{}, error) {
	overview := make(map[string]interface{})

	// Get company info
	if info, err := dfi.GetCompanyInfo(symbol); err == nil {
		overview["company_info"] = info
	}

	// Get recent quote
	if quote, err := dfi.GetYFinDataOnline(symbol); err == nil {
		overview["current_quote"] = quote
	}

	// Get recent market data
	if marketData, err := dfi.GetRecentMarketData(symbol); err == nil {
		overview["recent_data"] = marketData
	}

	// Get recent news
	if news, err := dfi.GetRecentNews(symbol); err == nil {
		overview["recent_news"] = news
	}

	if len(overview) == 0 {
		return nil, fmt.Errorf("unable to fetch any market overview data for %s", symbol)
	}

	return overview, nil
}

// Batch Operations

// GetMultipleSymbolsData gets market data for multiple symbols
func (dfi *DataFlowInterface) GetMultipleSymbolsData(symbols []string, days int) (map[string][]*MarketData, error) {
	result := make(map[string][]*MarketData)

	for _, symbol := range symbols {
		if data, err := dfi.GetYFinDataWindow(symbol, days); err == nil {
			result[symbol] = data
		}
	}

	return result, nil
}

// GetMultipleSymbolsNews gets news for multiple symbols
func (dfi *DataFlowInterface) GetMultipleSymbolsNews(symbols []string) (map[string][]*NewsArticle, error) {
	result := make(map[string][]*NewsArticle)

	for _, symbol := range symbols {
		if news, err := dfi.GetRecentNews(symbol); err == nil {
			result[symbol] = news
		}
	}

	return result, nil
}
