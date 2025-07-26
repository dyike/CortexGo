package dataflows

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/shopspring/decimal"
)

// FinnhubClient handles Finnhub API operations
type FinnhubClient struct {
	client *resty.Client
	cache  *CacheManager
	apiKey string
}

// NewFinnhubClient creates a new Finnhub client
func NewFinnhubClient(config *Config) *FinnhubClient {
	cacheDir := filepath.Join(config.DataCacheDir, "finnhub")
	cache := NewCacheManager(cacheDir, 6*time.Hour, config.CacheEnabled) // 6 hour cache for news
	
	client := resty.New()
	client.SetBaseURL("https://finnhub.io/api/v1")
	client.SetTimeout(30 * time.Second)
	
	return &FinnhubClient{
		client: client,
		cache:  cache,
		apiKey: config.FinnhubAPIKey,
	}
}

// FinnhubNews represents news from Finnhub API
type FinnhubNews struct {
	Category string    `json:"category"`
	DateTime int64     `json:"datetime"`
	Headline string    `json:"headline"`
	ID       int64     `json:"id"`
	Image    string    `json:"image"`
	Related  string    `json:"related"`
	Source   string    `json:"source"`
	Summary  string    `json:"summary"`
	URL      string    `json:"url"`
}

// GetCompanyNews gets news articles for a specific company
func (fc *FinnhubClient) GetCompanyNews(symbol string, from, to time.Time, config *Config) ([]*NewsArticle, error) {
	if fc.apiKey == "" {
		return nil, fmt.Errorf("Finnhub API key not configured")
	}
	
	if err := ValidateSymbol(symbol); err != nil {
		return nil, err
	}
	
	symbol = NormalizeSymbol(symbol)
	
	// Create cache key
	cacheKey := map[string]interface{}{
		"symbol": symbol,
		"from":   from.Format("2006-01-02"),
		"to":     to.Format("2006-01-02"),
	}
	
	// Check cache first
	var cached []*NewsArticle
	if fc.cache.Get("finnhub", "company_news", cacheKey, &cached) {
		return cached, nil
	}
	
	// Fetch from Finnhub
	var result []*NewsArticle
	err := WithRetry(DefaultRetryConfig(), func() error {
		resp, err := fc.client.R().
			SetQueryParams(map[string]string{
				"symbol": symbol,
				"from":   from.Format("2006-01-02"),
				"to":     to.Format("2006-01-02"),
				"token":  fc.apiKey,
			}).
			Get("/company-news")
		
		if err != nil {
			return fmt.Errorf("failed to fetch news for %s: %w", symbol, err)
		}
		
		if resp.StatusCode() != 200 {
			return fmt.Errorf("API error %d: %s", resp.StatusCode(), resp.String())
		}
		
		var finnhubNews []FinnhubNews
		if err := json.Unmarshal(resp.Body(), &finnhubNews); err != nil {
			return fmt.Errorf("failed to parse news response: %w", err)
		}
		
		// Convert to our format
		result = make([]*NewsArticle, 0, len(finnhubNews))
		for _, news := range finnhubNews {
			article := &NewsArticle{
				Title:       news.Headline,
				Content:     news.Summary,
				URL:         news.URL,
				Source:      news.Source,
				PublishedAt: time.Unix(news.DateTime, 0),
				Keywords:    []string{symbol},
				Metadata: map[string]string{
					"category": news.Category,
					"image":    news.Image,
					"related":  news.Related,
					"id":       strconv.FormatInt(news.ID, 10),
				},
			}
			result = append(result, article)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	// Cache the result
	fc.cache.Set("finnhub", "company_news", cacheKey, result)
	
	// Save to file
	filePath := filepath.Join(config.DataDir, "finnhub_data", 
		fmt.Sprintf("news_%s_%s_%s.json", symbol, 
			from.Format("2006-01-02"), to.Format("2006-01-02")))
	SaveDataToFile(result, filePath)
	
	return result, nil
}

// FinnhubInsiderTransaction represents insider transaction data
type FinnhubInsiderTransaction struct {
	Symbol              string  `json:"symbol"`
	PersonName          string  `json:"personName"`
	Share               int64   `json:"share"`
	Change              int64   `json:"change"`
	FilingDate          string  `json:"filingDate"`
	TransactionDate     string  `json:"transactionDate"`
	TransactionCode     string  `json:"transactionCode"`
	TransactionPrice    float64 `json:"transactionPrice"`
}

// GetInsiderTransactions gets insider trading data for a company
func (fc *FinnhubClient) GetInsiderTransactions(symbol string, from, to time.Time) ([]*InsiderTransaction, error) {
	if fc.apiKey == "" {
		return nil, fmt.Errorf("Finnhub API key not configured")
	}
	
	if err := ValidateSymbol(symbol); err != nil {
		return nil, err
	}
	
	symbol = NormalizeSymbol(symbol)
	
	// Create cache key
	cacheKey := map[string]interface{}{
		"symbol": symbol,
		"from":   from.Format("2006-01-02"),
		"to":     to.Format("2006-01-02"),
	}
	
	// Check cache first
	var cached []*InsiderTransaction
	if fc.cache.Get("finnhub", "insider_transactions", cacheKey, &cached) {
		return cached, nil
	}
	
	// Fetch from Finnhub
	var result []*InsiderTransaction
	err := WithRetry(DefaultRetryConfig(), func() error {
		resp, err := fc.client.R().
			SetQueryParams(map[string]string{
				"symbol": symbol,
				"from":   from.Format("2006-01-02"),
				"to":     to.Format("2006-01-02"),
				"token":  fc.apiKey,
			}).
			Get("/stock/insider-transactions")
		
		if err != nil {
			return fmt.Errorf("failed to fetch insider transactions for %s: %w", symbol, err)
		}
		
		if resp.StatusCode() != 200 {
			return fmt.Errorf("API error %d: %s", resp.StatusCode(), resp.String())
		}
		
		var apiResponse struct {
			Data []FinnhubInsiderTransaction `json:"data"`
		}
		
		if err := json.Unmarshal(resp.Body(), &apiResponse); err != nil {
			return fmt.Errorf("failed to parse insider transactions response: %w", err)
		}
		
		// Convert to our format
		result = make([]*InsiderTransaction, 0, len(apiResponse.Data))
		for _, trans := range apiResponse.Data {
			filingDate, _ := ParseDateString(trans.FilingDate)
			transactionDate, _ := ParseDateString(trans.TransactionDate)
			
			transaction := &InsiderTransaction{
				Symbol:           trans.Symbol,
				PersonName:       trans.PersonName,
				Share:            trans.Share,
				Change:           trans.Change,
				FilingDate:       filingDate,
				TransactionDate:  transactionDate,
				TransactionCode:  trans.TransactionCode,
				TransactionPrice: decimal.NewFromFloat(trans.TransactionPrice),
			}
			result = append(result, transaction)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	// Cache the result
	fc.cache.Set("finnhub", "insider_transactions", cacheKey, result)
	
	return result, nil
}

// FinnhubInsiderSentiment represents insider sentiment data
type FinnhubInsiderSentiment struct {
	Symbol string  `json:"symbol"`
	Year   int     `json:"year"`
	Month  int     `json:"month"`
	Change int64   `json:"change"`
	MSPR   float64 `json:"mspr"`
}

// GetInsiderSentiment gets insider sentiment data for a company
func (fc *FinnhubClient) GetInsiderSentiment(symbol string, from, to time.Time) ([]*InsiderSentiment, error) {
	if fc.apiKey == "" {
		return nil, fmt.Errorf("Finnhub API key not configured")
	}
	
	if err := ValidateSymbol(symbol); err != nil {
		return nil, err
	}
	
	symbol = NormalizeSymbol(symbol)
	
	// Create cache key
	cacheKey := map[string]interface{}{
		"symbol": symbol,
		"from":   from.Format("2006-01-02"),
		"to":     to.Format("2006-01-02"),
	}
	
	// Check cache first
	var cached []*InsiderSentiment
	if fc.cache.Get("finnhub", "insider_sentiment", cacheKey, &cached) {
		return cached, nil
	}
	
	// Fetch from Finnhub
	var result []*InsiderSentiment
	err := WithRetry(DefaultRetryConfig(), func() error {
		resp, err := fc.client.R().
			SetQueryParams(map[string]string{
				"symbol": symbol,
				"from":   from.Format("2006-01-02"),
				"to":     to.Format("2006-01-02"),
				"token":  fc.apiKey,
			}).
			Get("/stock/insider-sentiment")
		
		if err != nil {
			return fmt.Errorf("failed to fetch insider sentiment for %s: %w", symbol, err)
		}
		
		if resp.StatusCode() != 200 {
			return fmt.Errorf("API error %d: %s", resp.StatusCode(), resp.String())
		}
		
		var apiResponse struct {
			Data []FinnhubInsiderSentiment `json:"data"`
		}
		
		if err := json.Unmarshal(resp.Body(), &apiResponse); err != nil {
			return fmt.Errorf("failed to parse insider sentiment response: %w", err)
		}
		
		// Convert to our format
		result = make([]*InsiderSentiment, 0, len(apiResponse.Data))
		for _, sentiment := range apiResponse.Data {
			insiderSentiment := &InsiderSentiment{
				Symbol: sentiment.Symbol,
				Year:   sentiment.Year,
				Month:  sentiment.Month,
				Change: sentiment.Change,
				MSPR:   decimal.NewFromFloat(sentiment.MSPR),
			}
			result = append(result, insiderSentiment)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	// Cache the result
	fc.cache.Set("finnhub", "insider_sentiment", cacheKey, result)
	
	return result, nil
}

// GetGeneralNews gets general market news
func (fc *FinnhubClient) GetGeneralNews(category string) ([]*NewsArticle, error) {
	if fc.apiKey == "" {
		return nil, fmt.Errorf("Finnhub API key not configured")
	}
	
	// Check cache first
	var cached []*NewsArticle
	if fc.cache.Get("finnhub", "general_news", category, &cached) {
		return cached, nil
	}
	
	// Fetch from Finnhub
	var result []*NewsArticle
	err := WithRetry(DefaultRetryConfig(), func() error {
		resp, err := fc.client.R().
			SetQueryParams(map[string]string{
				"category": category,
				"token":    fc.apiKey,
			}).
			Get("/news")
		
		if err != nil {
			return fmt.Errorf("failed to fetch general news: %w", err)
		}
		
		if resp.StatusCode() != 200 {
			return fmt.Errorf("API error %d: %s", resp.StatusCode(), resp.String())
		}
		
		var finnhubNews []FinnhubNews
		if err := json.Unmarshal(resp.Body(), &finnhubNews); err != nil {
			return fmt.Errorf("failed to parse news response: %w", err)
		}
		
		// Convert to our format
		result = make([]*NewsArticle, 0, len(finnhubNews))
		for _, news := range finnhubNews {
			article := &NewsArticle{
				Title:       news.Headline,
				Content:     news.Summary,
				URL:         news.URL,
				Source:      news.Source,
				PublishedAt: time.Unix(news.DateTime, 0),
				Metadata: map[string]string{
					"category": news.Category,
					"image":    news.Image,
					"related":  news.Related,
					"id":       strconv.FormatInt(news.ID, 10),
				},
			}
			result = append(result, article)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	// Cache the result
	fc.cache.Set("finnhub", "general_news", category, result)
	
	return result, nil
}