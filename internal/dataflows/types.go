package dataflows

import (
	"time"

	"github.com/dyike/CortexGo/internal/config"
	"github.com/shopspring/decimal"
)

// Config is an alias for the main application config
type Config = config.Config

// MarketData represents stock price data
type MarketData struct {
	Symbol    string          `json:"symbol"`
	Date      time.Time       `json:"date"`
	Open      decimal.Decimal `json:"open"`
	High      decimal.Decimal `json:"high"`
	Low       decimal.Decimal `json:"low"`
	Close     decimal.Decimal `json:"close"`
	AdjClose  decimal.Decimal `json:"adj_close"`
	Volume    int64           `json:"volume"`
	Timestamp time.Time       `json:"timestamp"`
}

// NewsArticle represents a news article
type NewsArticle struct {
	Title       string            `json:"title"`
	Content     string            `json:"content"`
	URL         string            `json:"url"`
	Source      string            `json:"source"`
	PublishedAt time.Time         `json:"published_at"`
	Sentiment   float64           `json:"sentiment,omitempty"`
	Keywords    []string          `json:"keywords,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// RedditPost represents a Reddit post
type RedditPost struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	URL         string    `json:"url"`
	Subreddit   string    `json:"subreddit"`
	Author      string    `json:"author"`
	Score       int       `json:"score"`
	Comments    int       `json:"comments"`
	CreatedAt   time.Time `json:"created_at"`
	Sentiment   float64   `json:"sentiment,omitempty"`
	IsStickied  bool      `json:"is_stickied"`
	IsLocked    bool      `json:"is_locked"`
}

// InsiderTransaction represents insider trading data
type InsiderTransaction struct {
	Symbol          string          `json:"symbol"`
	PersonName      string          `json:"person_name"`
	Share           int64           `json:"share"`
	Change          int64           `json:"change"`
	FilingDate      time.Time       `json:"filing_date"`
	TransactionDate time.Time       `json:"transaction_date"`
	TransactionCode string          `json:"transaction_code"`
	TransactionPrice decimal.Decimal `json:"transaction_price"`
}

// InsiderSentiment represents aggregate insider sentiment
type InsiderSentiment struct {
	Symbol string          `json:"symbol"`
	Year   int             `json:"year"`
	Month  int             `json:"month"`
	Change int64           `json:"change"`
	MSPR   decimal.Decimal `json:"mspr"` // Monthly Share Purchase Ratio
}

// FinancialStatement represents fundamental financial data
type FinancialStatement struct {
	Symbol      string                 `json:"symbol"`
	Period      string                 `json:"period"`
	Year        int                    `json:"year"`
	Quarter     int                    `json:"quarter,omitempty"`
	ReportDate  time.Time              `json:"report_date"`
	Data        map[string]interface{} `json:"data"`
	StatementType string               `json:"statement_type"` // balance_sheet, income_statement, cash_flow
}

// DataResponse is a generic wrapper for API responses
type DataResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error,omitempty"`
	Source  string      `json:"source"`
	CachedAt *time.Time `json:"cached_at,omitempty"`
}

// DateRange represents a time period for data queries
type DateRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// QueryParams represents common query parameters
type QueryParams struct {
	Symbol    string     `json:"symbol"`
	DateRange *DateRange `json:"date_range,omitempty"`
	Limit     int        `json:"limit,omitempty"`
	Offset    int        `json:"offset,omitempty"`
	Source    string     `json:"source,omitempty"`
}